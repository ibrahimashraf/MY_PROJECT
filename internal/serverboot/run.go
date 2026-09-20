package serverboot

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/http2"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"

	"integin/internal/assetentitlementhttp"
	"integin/internal/assetentitlementpg"
	"integin/internal/assurancehttp"
	"integin/internal/assurancepg"
	"integin/internal/certificatepg"
	"integin/internal/certificatepublichttp"
	"integin/internal/certificaterender"
	"integin/internal/deviceenrollhttp"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/domain/device_trust"
	"integin/internal/domain/dpp"
	"integin/internal/domain/scheduling"
	domainsync "integin/internal/domain/sync"
	"integin/internal/dpphttp"
	"integin/internal/dpppg"
	"integin/internal/evidencepackhttp"
	"integin/internal/evidencepackpg"
	"integin/internal/formdefinitionhttp"
	"integin/internal/formdefinitionpg"
	"integin/internal/identity"
	"integin/internal/localprovision"
	"integin/internal/oidcauth"
	"integin/internal/oidchttp"
	"integin/internal/qrnfchttp"
	"integin/internal/qrnfcpg"
	"integin/internal/queue"
	"integin/internal/server"
	"integin/internal/shared/types"
	"integin/internal/shortlinkhttp"
	"integin/internal/shortlinkpg"
	"integin/internal/shortlinksvc"
	"integin/internal/storage"
	"integin/internal/syncstate"
	"integin/internal/timestamp"
	"integin/pkg/contextground"
	"integin/pkg/engine/cognitive"
	"integin/pkg/engine/eventbus"
	"integin/pkg/onboarding"
)

func durEnvSeconds(key string, fallback time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 && s <= 600 {
			return time.Duration(s) * time.Second
		}
	}
	return fallback
}

// Run boots the universal server and blocks serving until shutdown.
// Entry point for cmd/integin-server (which presets
// INTEGIN_DEPLOY_MODE from --mode before calling Run).
func Run() {
	secretStr := strings.TrimSpace(os.Getenv("INTEGIN_SYNC_SECRET"))
	if secretStr == "" {
		log.Fatal("INTEGIN_SYNC_SECRET is required")
	}
	secrets := make(map[string]string)
	if strings.HasPrefix(secretStr, "{") {
		if err := json.Unmarshal([]byte(secretStr), &secrets); err != nil {
			log.Fatalf("invalid INTEGIN_SYNC_SECRET JSON: %v", err)
		}
	} else {
		secrets["default"] = secretStr
	}
	deployMode, err := parseDeployMode(strings.TrimSpace(os.Getenv("INTEGIN_DEPLOY_MODE")))
	if err != nil {
		log.Fatal(err)
	}
	bindAddr := strings.TrimSpace(strings.TrimSpace(os.Getenv("INTEGIN_HTTP_ADDR")))
	if bindAddr == "" {
		bindAddr = ":8080"
	}
	allowLANBind := envBool("INTEGIN_AIRGAP_ALLOW_LAN_BIND")
	if err := checkDeployMode(deployMode, strings.TrimSpace(os.Getenv("INTEGIN_DB_URL")),
		strings.EqualFold(strings.TrimSpace(strings.TrimSpace(os.Getenv("INTEGIN_OIDC_ENABLED"))), "true"),
		strings.TrimSpace(os.Getenv("INTEGIN_OIDC_ISSUER")),
		firstEnv("INTEGIN_S3_ENDPOINT", "INTEGIN_RUSTFS_ENDPOINT", "INTEGIN_MINIO_ENDPOINT"),
		strings.TrimSpace(os.Getenv("INTEGIN_EVIDENCE_STORE")), bindAddr, allowLANBind,
		strings.TrimSpace(os.Getenv("INTEGIN_TSA_URL"))); err != nil {
		log.Fatal(err)
	}
	if deployMode != DeployModeCloud {
		log.Printf("INTEGIN_DEPLOY_MODE=%s enforced", deployMode)
	}
	if deployMode == DeployModeAirgap && allowLANBind && !isLoopbackAddress(bindAddr) {
		log.Printf("WARNING: airgap LAN bind explicitly allowed on %q — API is directly reachable on the rig LAN", bindAddr)
	}
	if err := onboarding.ProvisionAttestationFromEnv(); err != nil {
		log.Fatal(err)
	}
	tsaClient, err := timestamp.ProvisionFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	var database *sql.DB
	var stateRepo syncstate.SyncStateRepository
	var syncRepo syncstate.Repository
	var devices []device_trust.Device
	var authorities []device_trust.AuthorityPackage
	if dbURL := strings.TrimSpace(strings.TrimSpace(os.Getenv("INTEGIN_DB_URL"))); dbURL != "" {
		if strings.Contains(dbURL, "6432") && !strings.Contains(dbURL, "default_query_exec_mode") {
			separator := "?"
			if strings.Contains(dbURL, "?") {
				separator = "&"
			}
			dbURL = dbURL + separator + "default_query_exec_mode=exec"
		}
		var err error
		database, err = sql.Open("pgx", dbURL)
		if err != nil {
			log.Fatal(err)
		}
		database.SetMaxOpenConns(envInt("INTEGIN_DB_MAX_OPEN_CONNS", 90))
		database.SetMaxIdleConns(envInt("INTEGIN_DB_MAX_IDLE_CONNS", 50))
		database.SetConnMaxLifetime(30 * time.Minute)
		database.SetConnMaxIdleTime(5 * time.Minute)
		pingContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = database.PingContext(pingContext)
		cancel()
		if err != nil {
			log.Fatal(err)
		}
		repository, err := syncstate.NewPostgresRepository(database)
		if err != nil {
			log.Fatal(err)
		}
		stateRepo = repository
		syncRepo = repository
		loadContext, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
		devices, authorities, err = loadFromPostgres(loadContext, repository, envList("INTEGIN_TENANT_IDS", "INTEGIN_TENANT_ID"))
		loadCancel()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var err error
		authorities, err = loadAuthorities(strings.TrimSpace(os.Getenv("INTEGIN_AUTHORITY_FILE")))
		if err != nil {
			log.Fatal(err)
		}
		devices, err = loadDevices(strings.TrimSpace(os.Getenv("INTEGIN_DEVICE_FILE")))
		if err != nil {
			log.Fatal(err)
		}
		log.Print("INTEGIN_DB_URL is not configured; using legacy JSON device/authority bootstrap")
	}
	processor, err := domainsync.NewProcessorWithState(secrets, stateRepo)
	if err != nil {
		log.Fatal(err)
	}
	schemaVersioner, schemaErr := domainsync.CurrentSchemaVersioner()
	if schemaErr != nil {
		log.Printf("schema registry error: %v; tablet schema handshake remains unenforced (fail open)", schemaErr)
	} else if schemaVersioner != nil {
		processor.SetSchemaVersioner(schemaVersioner)
	}
	evidenceStore, err := configureEvidenceStore()
	if err != nil {
		log.Fatal(err)
	}
	readiness := func(ctx context.Context) error {
		if database != nil {
			if err := database.PingContext(ctx); err != nil {
				return err
			}
		}
		if evidenceStore != nil {
			if err := evidenceStore.Health(ctx); err != nil {
				return err
			}
		}
		return nil
	}
	address := strings.TrimSpace(os.Getenv("INTEGIN_HTTP_ADDR"))
	pilotManifestHandler, pilotAuthorityRegistry, err := server.PilotManifestHandlerFromEnvironment(processor, authorities, database, address)
	if err != nil {
		log.Fatal(err)
	}
	if address == "" {
		address = ":8080"
	}
	var localProvisioning http.Handler
	if envBool("INTEGIN_LOCAL_PROVISIONING_ENABLED") {
		if database == nil || stateRepo == nil {
			log.Fatal("local provisioning requires INTEGIN_DB_URL")
		}
		if !isLoopbackAddress(address) {
			log.Fatal("local provisioning requires INTEGIN_HTTP_ADDR to bind to localhost only")
		}
		repository, ok := stateRepo.(*syncstate.PostgresRepository)
		if !ok {
			log.Fatal("local provisioning requires the PostgreSQL sync-state repository")
		}
		handler, handlerErr := localprovision.NewHandler(localprovision.Config{
			Repository: repository, Processor: processor, SigningSecret: secretStr,
			TenantID:          strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_TENANT_ID")),
			OrganizationID:    strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_ORGANIZATION_ID")),
			UserID:            strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_USER_ID")),
			AuthorityLifetime: time.Duration(envInt("INTEGIN_LOCAL_PROVISIONING_AUTHORITY_MINUTES", 30)) * time.Minute,
		})
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		localProvisioning = handler
	}
	// Pilot-only in-memory device enrollment (pkg/onboarding EnrollServer): lets
	// the field_app simulator submission reach ProcessDeviceEnrollment over
	// loopback. Off by default; never expose on a non-loopback bind.
	var enrollHandler http.Handler
	if envBool("INTEGIN_PILOT_ENROLL_ENABLED") {
		if !isLoopbackAddress(address) {
			log.Fatal("pilot device enrollment requires INTEGIN_HTTP_ADDR to bind to localhost only")
		}
		sim, simErr := onboarding.NewEnrollmentSimulator()
		if simErr != nil {
			log.Fatal(simErr)
		}
		enrollHandler = onboarding.NewEnrollServer(sim).Handler()
	}
	var oidcSessionHandler http.Handler
	var sessionRevocationHandler http.Handler
	var sessionRevokeAllHandler http.Handler
	var workOrderHandler http.Handler
	var workOrderEvidenceHandler http.Handler
	var workOrderAssignmentHandler http.Handler
	var evidenceRegistrationHandler http.Handler
	var certificateHandler http.Handler
	var certificatePublicHandler http.Handler
	var tusHandler http.Handler
	var schedulingHandler http.Handler
	var deviceEnrollmentHandler http.Handler
	var certificateRepository *certificatepg.Repository
	if database != nil {
		repository, certificateErr := certificatepg.NewRepository(database)
		if certificateErr != nil {
			log.Fatal(certificateErr)
		}
		certificateRepository = repository
		certificateRepository.SetTSAClient(tsaClient)
		trustedProxies := envList("INTEGIN_TRUSTED_PROXIES", "INTEGIN_TRUSTED_PROXY")
		certificatePublicHandler = &certificatepublichttp.Handler{
			Verifier:       certificateRepository,
			TrustedProxies: trustedProxies,
		}
	}
	oidcConfig, oidcConfigErr := oidcauth.LoadConfig(os.Getenv)
	if oidcConfigErr != nil {
		log.Fatal(oidcConfigErr)
	}
	var licenseHandler http.Handler
	var activeValidator *oidcauth.Validator
	var activeResolver identity.Resolver
	if oidcConfig.Enabled {
		if database == nil {
			log.Fatal("OIDC session route requires INTEGIN_DB_URL")
		}
		startupContext, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		validator, validatorErr := oidcauth.NewValidator(startupContext, oidcConfig, nil)
		startupCancel()
		if validatorErr != nil {
			log.Fatal(validatorErr)
		}
		activeValidator = validator
		rawResolver, resolverErr := identity.NewPostgresResolver(database)
		if resolverErr != nil {
			log.Fatal(resolverErr)
		}
		cacheTTL := 60 * time.Second
		if raw := os.Getenv("INTEGIN_IDENTITY_CACHE_TTL"); raw != "" {
			if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
				cacheTTL = parsed
			}
		}
		activeResolver = identity.NewCachedResolver(rawResolver, cacheTTL)
		if certificateRepository == nil {
			log.Fatal("certificate repository requires INTEGIN_DB_URL")
		}
		var handlerErr error
		certificateHandler, handlerErr = server.NewCertificateHandlerWithStore(database, validator, activeResolver, evidenceStore)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		sessionHandler, sessionHandlerErr := oidchttp.NewSessionHandler(validator, activeResolver, oidchttp.SessionCapability)
		if sessionHandlerErr != nil {
			log.Fatal(sessionHandlerErr)
		}
		oidcSessionHandler = sessionHandler

		sessionStore, storeErr := oidchttp.NewPostgresSessionStore(database)
		if storeErr != nil {
			log.Fatal(storeErr)
		}
		sessionManager, managerErr := oidchttp.NewSessionLifecycleManagerWithStore(8*time.Hour, sessionStore)
		if managerErr != nil {
			log.Fatal(managerErr)
		}
		if cachedResolver, ok := activeResolver.(*identity.CachedResolver); ok {
			sessionManager.SetInvalidator(cachedResolver)
		}
		sessionRevocationHandler, handlerErr = oidchttp.NewSessionRevocationHandler(validator, sessionManager, false)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		sessionRevokeAllHandler, handlerErr = oidchttp.NewSessionRevocationHandler(validator, sessionManager, true)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}

		sessionAwareValidator, sessionAwareErr := oidchttp.NewSessionAwareValidator(validator, sessionManager)
		if sessionAwareErr != nil {
			log.Fatal(sessionAwareErr)
		}

		workOrderHandler, handlerErr = server.NewWorkOrderPartialSubmissionHandler(database, sessionAwareValidator, activeResolver)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		workOrderEvidenceHandler, handlerErr = server.NewWorkOrderEvidenceHandler(database, sessionAwareValidator, activeResolver)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		workOrderAssignmentHandler, handlerErr = server.NewWorkOrderAssignmentHandler(database, sessionAwareValidator, activeResolver)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		if evidenceStore != nil {
			evidenceRegistrationHandler, handlerErr = server.NewEvidenceMetadataRegistrationHandler(database, validator, activeResolver, evidenceStore)
			if handlerErr != nil {
				log.Fatal(handlerErr)
			}
		}
		licenseHandler, handlerErr = server.NewLicenseHandler(database, validator, activeResolver)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		schedulingHandler, handlerErr = server.NewSchedulingHandler(database, validator, activeResolver)
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}

		enrollCfg := deviceenrollhttp.Config{
			SigningSecret: secrets["default"],
		}
		if syncRepo != nil {
			enrollCfg.Repo = deviceenrollhttp.SyncRepository(syncRepo)
		}
		rawEnrollHandler, enrollErr := deviceenrollhttp.NewHandler(enrollCfg)
		if enrollErr != nil {
			log.Fatal(enrollErr)
		}
		sessionAuth, authErr := oidchttp.NewSessionAuthenticator(validator, activeResolver, sessionManager, identity.MFAPolicy{RequireMFA: false})
		if authErr != nil {
			log.Fatal(authErr)
		}
		deviceEnrollmentHandler = sessionAuth.Middleware(oidchttp.RejectTenantScopeConflict(rawEnrollHandler.Routes()))
	}

	log.Printf("loaded authority packages for HTTP sync registry: count=%d", len(authorities)) //nolint:gosec // count from DB, not user input
	flagAdminHandler, _ := server.NewFeatureFlagAdminHandler(database)
	identityGrantHandler, _ := server.NewIdentityGrantHandler(database, activeValidator, activeResolver)
	trainingHandler, _ := server.NewTrainingHandler(database)
	settingsHandler, _ := server.NewSettingsHandler(database)
	inspectionHandler, _ := server.NewInspectionHandler(database)
	searchHandler, _ := server.NewSearchHandler(database)
	auditLogHandler, _ := server.NewAuditLogHandler(database)
	analyticsHandler, _ := server.NewAnalyticsHandler(database)
	reportsHandler, _ := server.NewReportsHandler(database)
	liftViewExportHandler, _ := server.NewLiftViewExportHandler()

	var shortLinkHandler http.Handler
	var riverQueue *queue.Queue
	var qrnfcHandler http.Handler
	var assuranceHandler http.Handler
	var formDefHandler http.Handler
	var evidencePackHandler http.Handler
	var assetEntitlementHandler http.Handler
	var dppHandler http.Handler

	if database != nil {
		codeLen := envInt("SHORT_LINK_CODE_LENGTH", 6)
		if v := strings.TrimSpace(os.Getenv("QR_CODE_LENGTH")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				codeLen = n
			}
		}
		shortLinkSvc := shortlinksvc.New(shortlinkpg.New(database), codeLen, "")
		shortLinkHandler = shortlinkhttp.NewWithAuth(shortLinkSvc, activeValidator, activeResolver)

		// QR/NFC authenticated entry requires OIDC auth and the shared identity resolver.
		if activeValidator != nil && activeResolver != nil {
			qrnfcRepo, qrnfcErr := qrnfcpg.NewRepository(database)
			if qrnfcErr != nil {
				log.Fatal(qrnfcErr)
			}
			qrnfcHandler = qrnfchttp.NewHandler(activeValidator, activeResolver, qrnfcRepo, qrnfcRepo, nil)
		}
		projRepo, projErr := assurancepg.NewProjectionRepository(database)
		if projErr != nil {
			log.Fatal(projErr)
		}
		workRepo, workErr := assurancepg.NewWorkRepository(database)
		if workErr != nil {
			log.Fatal(workErr)
		}
		assuranceHandler = assurancehttp.NewHandler(activeValidator, activeResolver, projRepo, workRepo, nil)

		formDefRepo, formDefErr := formdefinitionpg.NewRepository(database)
		if formDefErr != nil {
			log.Fatal(formDefErr)
		}
		formDefHandler = formdefinitionhttp.NewHandler(activeValidator, activeResolver, formDefRepo, nil)

		packRepo, packErr := evidencepackpg.NewPackRepository(database)
		if packErr != nil {
			log.Fatal(packErr)
		}
		releaseRepo, releaseErr := evidencepackpg.NewReleaseRepository(database)
		if releaseErr != nil {
			log.Fatal(releaseErr)
		}
		evidencePackHandler = evidencepackhttp.NewHandler(activeValidator, activeResolver, packRepo, releaseRepo, nil)

		entRepo, entErr := assetentitlementpg.NewRepository(database)
		if entErr != nil {
			log.Fatal(entErr)
		}
		entPkgRepo, entPkgErr := assetentitlementpg.NewPackageRepository(database)
		if entPkgErr != nil {
			log.Fatal(entPkgErr)
		}
		assetEntitlementHandler = assetentitlementhttp.NewHandler(activeValidator, activeResolver, entRepo, entPkgRepo, nil)

		dppRepo, dppErr := dpppg.NewRepository(database)
		if dppErr != nil {
			log.Fatal(dppErr)
		}
		dppResolver := func(r *http.Request) (dpp.ActorContext, error) {
			if activeValidator == nil || activeResolver == nil {
				return dpp.ActorContext{TenantID: "default", OrganizationID: "default", ActorID: "system"}, nil
			}
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return dpp.ActorContext{}, errors.New("bearer token required")
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			principal, err := activeValidator.Validate(r.Context(), token)
			if err != nil {
				return dpp.ActorContext{}, err
			}
			mem, err := activeResolver.Resolve(r.Context(), identity.PrincipalKey{Issuer: principal.Issuer, Subject: principal.Subject})
			if err != nil {
				return dpp.ActorContext{}, err
			}
			return dpp.ActorContext{TenantID: mem.TenantID, OrganizationID: mem.OrganizationID, ActorID: mem.ActorID}, nil
		}
		dppHandler = dpphttp.NewHandler(dppRepo, activeValidator, dppResolver)

		// Initialize and start River queue with WebhookDeliveryWorker and CertificateRenderWorker
		workers := river.NewWorkers()
		poisonRecorder, poisonErr := queue.NewPostgresQuarantineRecorder(database)
		if poisonErr != nil {
			log.Fatalf("failed to initialize poison-pill quarantine recorder: %v", poisonErr)
		}
		river.AddWorker(workers, shortlinksvc.NewWebhookDeliveryWorker(shortLinkSvc))

		// Mount Phase 6: Cognitive Dispatcher and Corrective Work Order Worker
		sysEventBus := eventbus.NewBus()
		defer sysEventBus.Close()
		agentRegistry := &scheduling.InMemoryAgentRegistry{
			Competencies: map[string][]scheduling.TechnicianCompetency{},
		}
		if envJSON := strings.TrimSpace(os.Getenv("INTEGIN_BDI_AGENTS_JSON")); envJSON != "" {
			var agents []*cognitive.BDIAgent
			if err := json.Unmarshal([]byte(envJSON), &agents); err != nil {
				log.Fatalf("failed to unmarshal INTEGIN_BDI_AGENTS_JSON: %v", err)
			}
			agentRegistry.Agents = agents
			// Note: Competencies should similarly be loaded via config if needed.
		}
		river.AddWorker(workers, &scheduling.CorrectiveWorkOrderWorker{
			Registry: agentRegistry,
		})
		if evidenceStore != nil && certificateRepository != nil {
			renderer := domainrender.NewDeterministicPDFRenderer()
			renderSvc, renderErr := certificaterender.NewRenderService(renderer, evidenceStore, certificateRepository)
			if renderErr != nil {
				log.Fatalf("failed to initialize certificate render service: %v", renderErr)
			}
			river.AddWorker(workers, certificaterender.NewCertificateRenderWorker(renderSvc, certificateRepository))

			// Phase 7: Regulatory & CFIHOS workers
			// (Assuming equipmentRepo and evidenceStore wrappers exist in a real deployment)
			// river.AddWorker(workers, &integration.CFIHOSExportWorker{
			// 	EquipmentRepo: equipmentRepo,
			// 	Storage:       storageWrapper,
			// })
			// river.AddWorker(workers, &compliance.DossierExportWorker{
			// 	Renderer: &certificaterender.DossierRenderer{
			// 		PDFRenderer: renderer,
			// 		Data:        assetDataSource,
			// 	},
			// 	Storage: storageWrapper,
			// })
		}

		var riverErr error
		riverQueue, riverErr = queue.NewQueue(context.Background(), database, workers, queue.WithMiddleware(queue.NewPoisonQuarantineMiddleware(poisonRecorder)))
		if riverErr != nil {
			log.Fatalf("failed to initialize river queue: %v", riverErr)
		}
		if err := riverQueue.Start(context.Background()); err != nil {
			log.Fatalf("failed to start river queue workers: %v", err)
		}
		// Start automated retention pruner (sweeps every 10m, retain for 1h) to prevent XID wraparound table bloat
		riverQueue.StartPruneWorker(context.Background(), 10*time.Minute, 1*time.Hour)

		// Wire up the Cognitive Dispatcher (Phase 6)
		if len(agentRegistry.Agents) > 0 {
			scheduling.SubscribeEnvironmentalTelemetry(sysEventBus, agentRegistry.Agents[0], func(ctx context.Context, args river.JobArgs) error {
				if riverQueue != nil {
					_, err := riverQueue.Client().Insert(ctx, args, nil)
					return err
				}
				return nil
			})
		}
	}
	if tusDir := strings.TrimSpace(os.Getenv("INTEGIN_TUS_SCRATCH_DIR")); tusDir != "" {
		tusManager, tusErr := storage.NewTUSManager(tusDir, 0, 0)
		if tusErr != nil {
			log.Fatal(tusErr)
		}
		tusHandler = storage.TUSRouteHandler{Manager: tusManager, Store: evidenceStore}
		// Background janitor: crash recovery + stale-upload pruning every 10m,
		// logging sweep metrics (resumed/orphaned/purged/active).
		go tusManager.RunJanitor(context.Background(), 0, nil)
	}
	warnIfUnconfigured(tsaClient != nil, tusHandler != nil, activeValidator != nil, evidenceStore != nil)
	handler := server.NewMux(server.Dependencies{DB: database, SyncProcessor: processor, Devices: devices, Authorities: authorities, EvidenceStore: evidenceStore, Validator: activeValidator, Resolver: activeResolver, LocalProvisioning: localProvisioning, OIDCSessionHandler: oidcSessionHandler, SessionRevocationHandler: sessionRevocationHandler, SessionRevokeAllHandler: sessionRevokeAllHandler, WorkOrderHandler: workOrderHandler, WorkOrderEvidenceHandler: workOrderEvidenceHandler, WorkOrderAssignmentHandler: workOrderAssignmentHandler,
		PilotManifestHandler: pilotManifestHandler,
		AuthorityRegistry:    pilotAuthorityRegistry, Readiness: readiness, EvidenceRegistrationHandler: evidenceRegistrationHandler, CertificateHandler: certificateHandler, CertificatePublicHandler: certificatePublicHandler,
		LicenseHandler: licenseHandler, FlagAdminHandler: flagAdminHandler, IdentityGrantHandler: identityGrantHandler, TrainingHandler: trainingHandler,
		SettingsHandler: settingsHandler, InspectionHandler: inspectionHandler, SearchHandler: searchHandler,
		AuditLogHandler: auditLogHandler, AnalyticsHandler: analyticsHandler, ReportsHandler: reportsHandler, LiftViewExportHandler: liftViewExportHandler, ShortLinkHandler: shortLinkHandler, QRNFCHandler: qrnfcHandler, AssuranceHandler: assuranceHandler, FormDefinitionHandler: formDefHandler,
		EvidencePackHandler: evidencePackHandler, AssetEntitlementHandler: assetEntitlementHandler, DPPHandler: dppHandler, TUSHandler: tusHandler, SchedulingHandler: schedulingHandler,
		DeviceEnrollmentHandler: deviceEnrollmentHandler,
		ContextGroundHandler:    contextground.HTTPHandler(contextground.New())})
	if enrollHandler != nil {
		root := http.NewServeMux()
		root.Handle("/enroll/", enrollHandler)
		root.Handle("/", handler)
		handler = root
	}
	httpServer := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: durEnvSeconds("INTEGIN_HTTP_READ_HEADER_TIMEOUT_S", 10*time.Second),
		ReadTimeout:       durEnvSeconds("INTEGIN_HTTP_READ_TIMEOUT_S", 30*time.Second),
		WriteTimeout:      durEnvSeconds("INTEGIN_HTTP_WRITE_TIMEOUT_S", 60*time.Second),
		IdleTimeout:       durEnvSeconds("INTEGIN_HTTP_IDLE_TIMEOUT_S", 60*time.Second),
		MaxHeaderBytes:    1 << 20,
	}
	if err := http2.ConfigureServer(httpServer, &http2.Server{
		MaxConcurrentStreams: 250,
		MaxReadFrameSize:     1048576,
		IdleTimeout:          durEnvSeconds("INTEGIN_HTTP_IDLE_TIMEOUT_S", 60*time.Second),
	}); err != nil {
		log.Printf("http2 server configuration notice: %v", err)
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-stop:
		if riverQueue != nil {
			queueStopCtx, queueCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = riverQueue.Stop(queueStopCtx)
			queueCancel()
		}
		shutdownContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
	if database != nil {
		_ = database.Close()
	}
}

func loadFromPostgres(ctx context.Context, repository *syncstate.PostgresRepository, tenantIDs []string) ([]device_trust.Device, []device_trust.AuthorityPackage, error) {
	if len(tenantIDs) == 0 {
		return nil, nil, errors.New("INTEGIN_TENANT_ID or INTEGIN_TENANT_IDS is required when INTEGIN_DB_URL is configured")
	}
	devices := make([]device_trust.Device, 0)
	authorities := make([]device_trust.AuthorityPackage, 0)
	for _, tenantID := range tenantIDs {
		deviceRecords, err := repository.ListDevices(ctx, tenantID)
		if err != nil {
			return nil, nil, err
		}
		for _, record := range deviceRecords {
			device, err := device_trust.RestoreDevice(record.DeviceID, record.TenantID, record.OrganizationID, record.UserID, base64.StdEncoding.EncodeToString(record.PublicKey), types.DeviceTrustState(record.State), record.AuthorityEpoch)
			if err != nil {
				return nil, nil, err
			}
			devices = append(devices, device)
		}
		authorityRecords, err := repository.ListAuthorities(ctx, tenantID)
		if err != nil {
			return nil, nil, err
		}
		for _, record := range authorityRecords {
			authorities = append(authorities, device_trust.AuthorityPackage{ID: record.AuthorityID, DeviceID: record.DeviceID, TenantID: record.TenantID, UserID: record.UserID, Epoch: record.AuthorityEpoch, Scopes: append([]string(nil), record.Scopes...), IssuedAt: record.IssuedAt, ExpiresAt: record.ExpiresAt, Signature: string(record.Signature)})
		}
	}
	return devices, authorities, nil
}

func configureEvidenceStore() (storage.Store, error) {
	switch strings.ToLower(strings.TrimSpace(strings.TrimSpace(os.Getenv("INTEGIN_EVIDENCE_STORE")))) {
	case "", "disabled":
		return nil, nil
	case "memory":
		return storage.NewInMemoryStore(), nil
	case "rustfs", "s3", "minio":
		endpoint := firstEnv("INTEGIN_S3_ENDPOINT", "INTEGIN_RUSTFS_ENDPOINT", "INTEGIN_MINIO_ENDPOINT")
		bucket := firstEnv("INTEGIN_S3_BUCKET", "INTEGIN_RUSTFS_BUCKET", "INTEGIN_MINIO_BUCKET")
		accessKey := firstEnv("INTEGIN_S3_ACCESS_KEY", "INTEGIN_RUSTFS_ACCESS_KEY", "INTEGIN_MINIO_ACCESS_KEY")
		secretKey := firstEnv("INTEGIN_S3_SECRET_KEY", "INTEGIN_RUSTFS_SECRET_KEY", "INTEGIN_MINIO_SECRET_KEY")
		region := firstEnv("INTEGIN_S3_REGION", "INTEGIN_RUSTFS_REGION", "INTEGIN_MINIO_REGION")
		if region == "" {
			region = "us-east-1"
		}
		if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
			return nil, errors.New("INTEGIN_S3_ENDPOINT, INTEGIN_S3_BUCKET, INTEGIN_S3_ACCESS_KEY, and INTEGIN_S3_SECRET_KEY are required for RustFS/S3")
		}
		return storage.NewS3Store(endpoint, bucket, http.DefaultClient, storage.NewAWSSigV4Signer(accessKey, secretKey, region, "s3"))
	default:
		return nil, errors.New("INTEGIN_EVIDENCE_STORE must be disabled, memory, rustfs, s3, or minio")
	}
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(name string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(name)))
	return err == nil && value
}

func isLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	return host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

func envList(primary, fallback string) []string {
	value := os.Getenv(primary)
	if strings.TrimSpace(value) == "" {
		value = os.Getenv(fallback)
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// safeReadFile validates and canonicalizes path before reading.
// For server config files, we accept absolute paths from config/env.
//
//nolint:gosec // path canonicalized via Clean+Abs; from config/env
func safeReadFile(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	clean := filepath.Clean(path)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return nil, err
	}
	//nolint:gosec // path canonicalized via Clean+Abs; from config/env
	return os.ReadFile(abs)
}

func loadAuthorities(path string) ([]device_trust.AuthorityPackage, error) {
	if path == "" {
		return nil, nil
	}
	content, err := safeReadFile(path)
	if err != nil {
		return nil, err
	}
	var authorities []device_trust.AuthorityPackage
	if err := json.Unmarshal(content, &authorities); err != nil {
		return nil, err
	}
	return authorities, nil
}

func loadDevices(path string) ([]device_trust.Device, error) {
	if path == "" {
		return nil, nil
	}
	content, err := safeReadFile(path)
	if err != nil {
		return nil, err
	}
	var configs []deviceConfig
	if err := json.Unmarshal(content, &configs); err != nil {
		return nil, err
	}
	devices := make([]device_trust.Device, 0, len(configs))
	for _, config := range configs {
		device, err := device_trust.NewDevice(config.ID, config.TenantID, config.OrganizationID, config.UserID, config.PublicKey)
		if err != nil {
			return nil, err
		}
		if config.Trusted {
			if err := device.Trust(); err != nil {
				return nil, err
			}
		}
		devices = append(devices, device)
	}
	return devices, nil
}

type deviceConfig struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	OrganizationID string `json:"organization_id"`
	UserID         string `json:"user_id"`
	PublicKey      string `json:"public_key"`
	Trusted        bool   `json:"trusted"`
}
