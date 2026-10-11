package serverboot

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
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

	"integin/internal/advisory"
	"integin/internal/advisoryhttp"
	"integin/internal/assetentitlementhttp"
	"integin/internal/assetentitlementpg"
	"integin/internal/assurancehttp"
	"integin/internal/assurancepg"
	"integin/internal/atexhttp"
	"integin/internal/certificatepg"
	"integin/internal/certificatepublichttp"
	"integin/internal/certificaterender"
	"integin/internal/certtemplatepg"
	"integin/internal/deviceenrollhttp"
	domainrender "integin/internal/domain/certificaterender"
	"integin/internal/domain/certificatetemplate"
	"integin/internal/domain/device_trust"
	"integin/internal/domain/dpp"
	"integin/internal/domain/scheduling"
	domainsync "integin/internal/domain/sync"
	"integin/internal/dpphttp"
	"integin/internal/dpppg"
	"integin/internal/evidencepackhttp"
	"integin/internal/evidencepackpg"
	"integin/internal/federation"
	"integin/internal/federationhttp"
	"integin/internal/formdefinitionhttp"
	"integin/internal/formdefinitionpg"
	"integin/internal/identity"
	"integin/internal/jurisdictionhttp"
	"integin/internal/localprovision"
	"integin/internal/oidcauth"
	"integin/internal/oidchttp"
	"integin/internal/photogrammetryhttp"
	"integin/internal/pipinghttp"
	"integin/internal/qrnfchttp"
	"integin/internal/qrnfcpg"
	"integin/internal/queryenginehttp"
	"integin/internal/queue"
	"integin/internal/reinspectionhttp"
	"integin/internal/riskhttp"
	"integin/internal/robotichttp"
	"integin/internal/server"
	"integin/internal/shared/types"
	"integin/internal/shortlinkhttp"
	"integin/internal/shortlinkpg"
	"integin/internal/shortlinksvc"
	"integin/internal/storage"
	"integin/internal/syncapi"
	"integin/internal/syncstate"
	"integin/internal/telemetrystream"
	"integin/internal/telemetrystreamhttp"
	"integin/internal/timestamp"
	"integin/internal/workbenchhttp"
	"integin/pkg/contextground"
	"integin/pkg/engine/cognitive"
	"integin/pkg/engine/eventbus"
	"integin/pkg/jurisdictions"
	"integin/pkg/onboarding"
	"integin/pkg/queryengine"
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
// bootConfig is the validated, environment-sourced configuration Run needs
// before it touches any infrastructure.
type bootConfig struct {
	secrets          map[string]string
	deployMode       DeployMode
	bindAddr         string
	allowLANBind     bool
	attestRoots      *x509.CertPool
	appleAttestRoots *x509.CertPool
	appleAppID       string
	tsa              *timestamp.TSA
}

// loadBootConfig reads and validates every boot-time configuration source.
// Any missing or malformed input is fatal: a half-configured rig must not
// reach the point of serving traffic.
func loadBootConfig() bootConfig {
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
	attestRoots, appleAttestRoots, appleAppID, err := onboarding.ProvisionAttestationFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	tsaClient, err := timestamp.ProvisionFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return bootConfig{
		secrets:          secrets,
		deployMode:       deployMode,
		bindAddr:         bindAddr,
		allowLANBind:     allowLANBind,
		attestRoots:      attestRoots,
		appleAttestRoots: appleAttestRoots,
		appleAppID:       appleAppID,
		tsa:              tsaClient,
	}
}

func Run() {
	cfg := loadBootConfig()
	secrets := cfg.secrets
	var database *sql.DB
	var deps server.Dependencies
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
		handler := mustHandler(localprovision.NewHandler(localprovision.Config{
			Repository: repository, Processor: processor, SigningSecret: cfg.secrets["default"],
			TenantID:          firstEnv("INTEGIN_LOCAL_PROVISIONING_TENANT_ID", "INTEGIN_TENANT_ID"),
			OrganizationID:    strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_ORGANIZATION_ID")),
			UserID:            strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_USER_ID")),
			AuthorityLifetime: time.Duration(envInt("INTEGIN_LOCAL_PROVISIONING_AUTHORITY_MINUTES", 30)) * time.Minute,
		}))
		deps.LocalProvisioning = handler
	}
	// Pilot-only in-memory device enrollment (pkg/onboarding EnrollServer): lets
	// the field_app simulator submission reach ProcessDeviceEnrollment over
	// loopback. Off by default; never expose on a non-loopback bind.
	if pilotAuthorityRegistry == nil {
		pilotAuthorityRegistry = syncapi.NewAuthorityRegistry()
	}
	var enrollHandler http.Handler
	if envBool("INTEGIN_PILOT_ENROLL_ENABLED") {
		if !isLoopbackAddress(address) && !envBool("INTEGIN_PILOT_ENROLL_ALLOW_BIND_ALL") {
			log.Fatal("pilot device enrollment requires INTEGIN_HTTP_ADDR to bind to localhost only")
		}
		sim, simErr := onboarding.NewEnrollmentSimulator(
			onboarding.WithAttestationRoots(cfg.attestRoots),
			onboarding.WithAppleAttestRoots(cfg.appleAttestRoots),
			onboarding.WithExpectedAppID(cfg.appleAppID),
		)
		if simErr != nil {
			log.Fatal(simErr)
		}
		enrollSrv := onboarding.NewEnrollServer(sim)
		enrollSrv.OnEnrolled = func(record onboarding.DeviceTrustRecord) {
			// Bridge pilot enrollment to the sync processor so enrolled
			// devices can immediately sync without a DB round-trip.
			pubBytes, err := hex.DecodeString(record.DevicePublicKey)
			if err != nil || len(pubBytes) != ed25519.PublicKeySize {
				log.Printf("pilot enrollment: cannot decode device public key: %v", err)
				return
			}
			pubKeyBase64 := base64.StdEncoding.EncodeToString(pubBytes)
			device, err := device_trust.NewDevice(record.DeviceID, record.TenantID, record.OrganizationID, record.UserID, pubKeyBase64)
			if err != nil {
				log.Printf("pilot enrollment: cannot create device: %v", err)
				return
			}
			if err := device.Trust(); err != nil {
				log.Printf("pilot enrollment: cannot trust device: %v", err)
				return
			}
			processor.RegisterDevice(device)
			now := time.Now().UTC()
			authorityID := "pilot-auth-" + record.DeviceID
			scopes := []string{"inspection.perform", "inspection.submit"}
			authority, err := device_trust.IssueAuthorityPackage(device, authorityID, cfg.secrets["default"], scopes, now, 24*time.Hour)
			if err != nil {
				log.Printf("pilot enrollment: cannot issue authority: %v", err)
				return
			}
			pilotAuthorityRegistry.Register(authority)
			log.Printf("pilot enrollment: device %s enrolled and registered with sync processor (org=%s user=%s)", record.DeviceID, record.OrganizationID, record.UserID)
		}
		enrollHandler = enrollSrv.Handler()
	}
	var tusHandler http.Handler
	var certificateRepository *certificatepg.Repository
	if database != nil {
		repository, certificateErr := certificatepg.NewRepository(database)
		if certificateErr != nil {
			log.Fatal(certificateErr)
		}
		certificateRepository = repository
		certificateRepository.SetTSAClient(cfg.tsa)
		trustedProxies := envList("INTEGIN_TRUSTED_PROXIES", "INTEGIN_TRUSTED_PROXY")
		deps.CertificatePublicHandler = &certificatepublichttp.Handler{
			Verifier:       certificateRepository,
			TrustedProxies: trustedProxies,
		}
	}
	oidcConfig, oidcConfigErr := oidcauth.LoadConfig(os.Getenv)
	if oidcConfigErr != nil {
		log.Fatal(oidcConfigErr)
	}
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
		deps.CertificateHandler = mustHandler(server.NewCertificateHandlerWithStore(database, validator, activeResolver, evidenceStore))
		sessionHandler := mustHandler(oidchttp.NewSessionHandler(validator, activeResolver, oidchttp.SessionCapability))
		deps.OIDCSessionHandler = sessionHandler

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
		deps.SessionRevocationHandler = mustHandler(oidchttp.NewSessionRevocationHandler(validator, sessionManager, false))
		deps.SessionRevokeAllHandler = mustHandler(oidchttp.NewSessionRevocationHandler(validator, sessionManager, true))

		sessionAwareValidator, sessionAwareErr := oidchttp.NewSessionAwareValidator(validator, sessionManager)
		if sessionAwareErr != nil {
			log.Fatal(sessionAwareErr)
		}

		deps.WorkOrderHandler = mustHandler(server.NewWorkOrderPartialSubmissionHandler(database, sessionAwareValidator, activeResolver))
		deps.WorkOrderEvidenceHandler = mustHandler(server.NewWorkOrderEvidenceHandler(database, sessionAwareValidator, activeResolver))
		deps.WorkOrderAssignmentHandler = mustHandler(server.NewWorkOrderAssignmentHandler(database, sessionAwareValidator, activeResolver))
		deps.WorkOrderHandoverHandler = mustHandler(server.NewWorkOrderHandoverHandler(database, sessionAwareValidator, activeResolver))
		if evidenceStore != nil {
			deps.EvidenceRegistrationHandler = mustHandler(server.NewEvidenceMetadataRegistrationHandler(database, validator, activeResolver, evidenceStore))
		}
		deps.LicenseHandler = mustHandler(server.NewLicenseHandler(database, validator, activeResolver))
		deps.SchedulingHandler = mustHandler(server.NewSchedulingHandler(database, validator, activeResolver))

		enrollCfg := deviceenrollhttp.Config{
			SigningSecret: secrets["default"],
		}
		if syncRepo != nil {
			enrollCfg.Repo = deviceenrollhttp.SyncRepository(syncRepo)
			// Revocation must reach the request-time caches, not just the
			// database: without this a revoked device keeps syncing until the
			// next restart.
			enrollCfg.OnDeviceRevoked = func(deviceID string) {
				processor.RevokeDevice(deviceID)
				revokedAuthorities := pilotAuthorityRegistry.RevokeDeviceAuthorities(deviceID)
				log.Printf("device revocation applied to live caches: device=%s authorities=%d", deviceID, revokedAuthorities)
			}
		} else {
			log.Printf("WARNING: device revocation is NOT available — INTEGIN_DB_URL is missing, so no device can be revoked at runtime")
		}
		rawEnrollHandler, enrollErr := deviceenrollhttp.NewHandler(enrollCfg)
		if enrollErr != nil {
			log.Fatal(enrollErr)
		}
		sessionAuth, authErr := oidchttp.NewSessionAuthenticator(validator, activeResolver, sessionManager, identity.MFAPolicy{RequireMFA: false})
		if authErr != nil {
			log.Fatal(authErr)
		}
		deps.DeviceEnrollmentHandler = sessionAuth.Middleware(oidchttp.RejectTenantScopeConflict(rawEnrollHandler.Routes()))
	}

	log.Printf("loaded authority packages for HTTP sync registry: count=%d", len(authorities)) //nolint:gosec // count from DB, not user input
	deps.FlagAdminHandler = mustHandler(server.NewFeatureFlagAdminHandler(database))
	deps.IdentityGrantHandler = mustHandler(server.NewIdentityGrantHandler(database, activeValidator, activeResolver))
	deps.TrainingHandler = mustHandler(server.NewTrainingHandler(database))
	deps.SettingsHandler = mustHandler(server.NewSettingsHandler(database))
	deps.InspectionHandler = mustHandler(server.NewInspectionHandler(database))
	deps.SearchHandler = mustHandler(server.NewSearchHandler(database))
	deps.AuditLogHandler = mustHandler(server.NewAuditLogHandler(database))
	deps.AnalyticsHandler = mustHandler(server.NewAnalyticsHandler(database))
	deps.ReportsHandler = mustHandler(server.NewReportsHandler(database))
	deps.LiftViewExportHandler = mustHandler(server.NewLiftViewExportHandler())
	deps.WorkbenchHandler = workbenchhttp.NewHandler(database, devices)
	deps.AdvisoryHandler = advisoryhttp.NewHandler(advisory.NewRegistry())

	jurisdictionsRegistry, jErr := jurisdictions.NewRegistry(jurisdictions.TopJurisdictions())
	if jErr != nil {
		log.Fatalf("failed to initialize jurisdictions registry: %v", jErr)
	}
	deps.JurisdictionHandler = jurisdictionhttp.NewHandler(jurisdictionsRegistry)

	queryRegistry, qErr := queryengine.NewSchemaRegistry(queryenginehttp.CanonicalSchema())
	if qErr != nil {
		log.Fatalf("failed to initialize query engine schema registry: %v", qErr)
	}
	deps.QueryHandler = queryenginehttp.NewHandler(queryRegistry, database)

	if database != nil {
		if _, ctErr := certtemplatepg.NewRepository(database, certificatetemplate.DefaultCatalog()); ctErr != nil {
			log.Printf("warning: certtemplatepg repository initialization failed: %v", ctErr)
		}
	}

	var riverQueue *queue.Queue

	if database != nil {
		codeLen := envInt("SHORT_LINK_CODE_LENGTH", 6)
		if v := strings.TrimSpace(os.Getenv("QR_CODE_LENGTH")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				codeLen = n
			}
		}
		shortLinkSvc := shortlinksvc.New(shortlinkpg.New(database), codeLen, "")
		deps.ShortLinkHandler = shortlinkhttp.NewWithAuth(shortLinkSvc, activeValidator, activeResolver)

		// QR/NFC authenticated entry requires OIDC auth and the shared identity resolver.
		if activeValidator != nil && activeResolver != nil {
			qrnfcRepo, qrnfcErr := qrnfcpg.NewRepository(database)
			if qrnfcErr != nil {
				log.Fatal(qrnfcErr)
			}
			deps.QRNFCHandler = qrnfchttp.NewHandler(activeValidator, activeResolver, qrnfcRepo, qrnfcRepo, nil)
		}
		projRepo, projErr := assurancepg.NewProjectionRepository(database)
		if projErr != nil {
			log.Fatal(projErr)
		}
		workRepo, workErr := assurancepg.NewWorkRepository(database)
		if workErr != nil {
			log.Fatal(workErr)
		}
		deps.AssuranceHandler = assurancehttp.NewHandler(activeValidator, activeResolver, projRepo, workRepo, nil)

		formDefRepo, formDefErr := formdefinitionpg.NewRepository(database)
		if formDefErr != nil {
			log.Fatal(formDefErr)
		}
		deps.FormDefinitionHandler = formdefinitionhttp.NewHandler(activeValidator, activeResolver, formDefRepo, nil)

		packRepo, packErr := evidencepackpg.NewPackRepository(database)
		if packErr != nil {
			log.Fatal(packErr)
		}
		releaseRepo, releaseErr := evidencepackpg.NewReleaseRepository(database)
		if releaseErr != nil {
			log.Fatal(releaseErr)
		}
		deps.EvidencePackHandler = evidencepackhttp.NewHandler(activeValidator, activeResolver, packRepo, releaseRepo, nil)

		entRepo, entErr := assetentitlementpg.NewRepository(database)
		if entErr != nil {
			log.Fatal(entErr)
		}
		entPkgRepo, entPkgErr := assetentitlementpg.NewPackageRepository(database)
		if entPkgErr != nil {
			log.Fatal(entPkgErr)
		}
		deps.AssetEntitlementHandler = assetentitlementhttp.NewHandler(activeValidator, activeResolver, entRepo, entPkgRepo, nil)

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
		deps.DPPHandler = dpphttp.NewHandler(dppRepo, activeValidator, dppResolver)

		// Background worker queue: webhook delivery, certificate rendering and
		// corrective work orders.
		riverQueue = startWorkers(database, shortLinkSvc, evidenceStore, certificateRepository)
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
	warnIfUnconfigured(cfg.tsa != nil, tusHandler != nil, activeValidator != nil, evidenceStore != nil)

	deps.RoboticsHandler = robotichttp.NewHandler(nil)
	telemetryBuffer := telemetrystream.NewStreamBuffer(database, telemetrystream.DefaultIngestConfig())
	deps.TelemetryStreamHandler = telemetrystreamhttp.NewHandler(telemetryBuffer)
	deps.CARHandler = reinspectionhttp.NewHandler(nil)
	localCell := federation.CellSaudiCentral01
	if cell := os.Getenv("INTEGIN_SOVEREIGN_CELL_ID"); cell != "" {
		localCell = federation.SovereignCellID(cell)
	}
	fedGate := federation.NewFederationGatekeeper(localCell)
	deps.FederationHandler = federationhttp.NewHandler(fedGate)
	deps.RiskHandler = riskhttp.NewHandler()
	deps.ATEXHandler = atexhttp.NewHandler()
	deps.PhotogrammetryHandler = photogrammetryhttp.NewHandler()
	deps.PipingHandler = pipinghttp.NewHandler()

	handler := server.NewMux(server.Dependencies{DB: database, SyncProcessor: processor, Devices: devices, Authorities: authorities, EvidenceStore: evidenceStore, Validator: activeValidator, Resolver: activeResolver, WorkOrderReconciliationHandler: server.NewWorkOrderReconciliationHandler(),
		PilotManifestHandler: pilotManifestHandler,
		AuthorityRegistry:    pilotAuthorityRegistry, Readiness: readiness, TUSHandler: tusHandler, UploadTokenSecret: cfg.secrets["default"], UploadAuthorityRegistry: pilotAuthorityRegistry,
		DeviceEnrollmentHandler: deps.DeviceEnrollmentHandler,
		WorkbenchHandler:        deps.WorkbenchHandler,
		AdvisoryHandler:         deps.AdvisoryHandler,
		JurisdictionHandler:     deps.JurisdictionHandler,
		QueryHandler:            deps.QueryHandler,
		RoboticsHandler:         deps.RoboticsHandler,
		TelemetryStreamHandler:  deps.TelemetryStreamHandler,
		CARHandler:              deps.CARHandler,
		FederationHandler:       deps.FederationHandler,
		RiskHandler:             deps.RiskHandler,
		ATEXHandler:             deps.ATEXHandler,
		PhotogrammetryHandler:   deps.PhotogrammetryHandler,
		PipingHandler:           deps.PipingHandler,
		PilotEnrollHandler:      enrollHandler,
		ContextGroundHandler:    contextground.HTTPHandler(contextground.New())})
	if enrollHandler != nil {
		log.Printf("pilot enrollment server mounted at /enroll/")
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
	serveHTTP(httpServer, riverQueue, database)
}

// mustHandler collapses the "construct a handler, abort if it failed" pattern
// that repeats for every route. A handler that cannot be wired must stop the
// process rather than silently leave a route unmounted and unprotected.
func mustHandler[T any](h T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return h
}

// startWorkers builds and starts the background River queue: webhook delivery,
// certificate rendering, and corrective work orders, plus the cognitive
// dispatcher subscription. Extracted from Run so the background subsystem is
// wired in one place instead of inline in the boot sequence.
//
// Returns the agent registry (nil when there is no database) and the started
// queue, so Run can later drain the queue on shutdown.
func startWorkers(
	database *sql.DB,
	shortLinkSvc *shortlinksvc.Service,
	evidenceStore storage.Store,
	certificateRepository *certificatepg.Repository,
) *queue.Queue {
	if database == nil {
		return nil
	}

	workers := river.NewWorkers()
	poisonRecorder, err := queue.NewPostgresQuarantineRecorder(database)
	if err != nil {
		log.Fatalf("failed to initialize poison-pill quarantine recorder: %v", err)
	}
	river.AddWorker(workers, shortlinksvc.NewWebhookDeliveryWorker(shortLinkSvc))

	agentRegistry := &scheduling.InMemoryAgentRegistry{
		Competencies: map[string][]scheduling.TechnicianCompetency{},
	}
	if envJSON := strings.TrimSpace(os.Getenv("INTEGIN_BDI_AGENTS_JSON")); envJSON != "" {
		var agents []*cognitive.BDIAgent
		if err := json.Unmarshal([]byte(envJSON), &agents); err != nil {
			log.Fatalf("failed to unmarshal INTEGIN_BDI_AGENTS_JSON: %v", err)
		}
		agentRegistry.Agents = agents
	}
	river.AddWorker(workers, &scheduling.CorrectiveWorkOrderWorker{Registry: agentRegistry})

	if evidenceStore != nil && certificateRepository != nil {
		renderer := domainrender.NewDeterministicPDFRenderer()
		renderSvc, err := certificaterender.NewRenderService(renderer, evidenceStore, certificateRepository)
		if err != nil {
			log.Fatalf("failed to initialize certificate render service: %v", err)
		}
		river.AddWorker(workers, certificaterender.NewCertificateRenderWorker(renderSvc, certificateRepository))
	}

	q, err := queue.NewQueue(context.Background(), database, workers,
		queue.WithMiddleware(queue.NewPoisonQuarantineMiddleware(poisonRecorder)))
	if err != nil {
		log.Fatalf("failed to initialize river queue: %v", err)
	}
	if err := q.Start(context.Background()); err != nil {
		log.Fatalf("failed to start river queue workers: %v", err)
	}
	// Sweeps every 10m, retaining 1h, to prevent XID wraparound table bloat.
	q.StartPruneWorker(context.Background(), 10*time.Minute, 1*time.Hour)

	if len(agentRegistry.Agents) > 0 {
		sysEventBus := eventbus.NewBus()
		defer sysEventBus.Close()
		scheduling.SubscribeEnvironmentalTelemetry(sysEventBus, agentRegistry.Agents[0], func(ctx context.Context, args river.JobArgs) error {
			_, err := q.Client().Insert(ctx, args, nil)
			return err
		})
	}
	return q
}

// serveHTTP runs the HTTP server until it fails or the process is signalled,
// then drains the worker queue and closes the database. Extracted from Run so
// the lifecycle is independently exercisable and Run stays a boot sequence.
func serveHTTP(httpServer *http.Server, riverQueue *queue.Queue, database *sql.DB) {
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
	defer signal.Stop(stop)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-stop:
		stopQueue(riverQueue)
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

// stopQueue drains in-flight River jobs so a shutdown does not abandon
// halfway-processed webhook deliveries or certificate renders.
func stopQueue(q *queue.Queue) {
	if q == nil {
		return
	}
	queueStopCtx, queueCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer queueCancel()
	_ = q.Stop(queueStopCtx)
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
