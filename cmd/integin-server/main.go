package main

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
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/identity"
	"integin/internal/localprovision"
	"integin/internal/oidcauth"
	"integin/internal/oidchttp"
	"integin/internal/server"
	"integin/internal/shared/types"
	"integin/internal/storage"
	"integin/internal/syncstate"
)

func main() {
	secret := strings.TrimSpace(os.Getenv("INTEGIN_SYNC_SECRET"))
	if secret == "" {
		log.Fatal("INTEGIN_SYNC_SECRET is required")
	}
	var database *sql.DB
	var stateRepo syncstate.SyncStateRepository
	var devices []device_trust.Device
	var authorities []device_trust.AuthorityPackage
	if dbURL := strings.TrimSpace(os.Getenv("INTEGIN_DB_URL")); dbURL != "" {
		var err error
		database, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatal(err)
		}
		database.SetMaxOpenConns(envInt("INTEGIN_DB_MAX_OPEN_CONNS", 20))
		database.SetMaxIdleConns(envInt("INTEGIN_DB_MAX_IDLE_CONNS", 5))
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
		loadContext, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
		devices, authorities, err = loadFromPostgres(loadContext, repository, envList("INTEGIN_TENANT_IDS", "INTEGIN_TENANT_ID"))
		loadCancel()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var err error
		authorities, err = loadAuthorities(os.Getenv("INTEGIN_AUTHORITY_FILE"))
		if err != nil {
			log.Fatal(err)
		}
		devices, err = loadDevices(os.Getenv("INTEGIN_DEVICE_FILE"))
		if err != nil {
			log.Fatal(err)
		}
		log.Print("INTEGIN_DB_URL is not configured; using legacy JSON device/authority bootstrap")
	}
	processor, err := domainsync.NewProcessorWithState(secret, stateRepo)
	if err != nil {
		log.Fatal(err)
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
	address := os.Getenv("INTEGIN_HTTP_ADDR")
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
			Repository: repository, Processor: processor, SigningSecret: secret,
			TenantID:          firstEnv("INTEGIN_LOCAL_PROVISIONING_TENANT_ID", "INTEGIN_TENANT_ID"),
			OrganizationID:    strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_ORGANIZATION_ID")),
			UserID:            strings.TrimSpace(os.Getenv("INTEGIN_LOCAL_PROVISIONING_USER_ID")),
			AuthorityLifetime: time.Duration(envInt("INTEGIN_LOCAL_PROVISIONING_AUTHORITY_MINUTES", 30)) * time.Minute,
		})
		if handlerErr != nil {
			log.Fatal(handlerErr)
		}
		localProvisioning = handler
	}
	var oidcSessionHandler http.Handler
	oidcConfig, oidcConfigErr := oidcauth.LoadConfig(os.Getenv)
	if oidcConfigErr != nil {
		log.Fatal(oidcConfigErr)
	}
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
		resolver, resolverErr := identity.NewPostgresResolver(database)
		if resolverErr != nil {
			log.Fatal(resolverErr)
		}
		sessionHandler, sessionHandlerErr := oidchttp.NewSessionHandler(validator, resolver, oidchttp.SessionCapability)
		if sessionHandlerErr != nil {
			log.Fatal(sessionHandlerErr)
		}
		oidcSessionHandler = sessionHandler
	}

	log.Printf("loaded authority packages for HTTP sync registry: count=%d", len(authorities))
	httpServer := &http.Server{
		Addr: address,
		Handler: server.NewMux(server.Dependencies{SyncProcessor: processor, Devices: devices, Authorities: authorities, EvidenceStore: evidenceStore, LocalProvisioning: localProvisioning, OIDCSessionHandler: oidcSessionHandler,
			PilotManifestHandler: pilotManifestHandler,
			AuthorityRegistry:    pilotAuthorityRegistry, Readiness: readiness}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
	switch strings.ToLower(strings.TrimSpace(os.Getenv("INTEGIN_EVIDENCE_STORE"))) {
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

func loadAuthorities(path string) ([]device_trust.AuthorityPackage, error) {
	if path == "" {
		return nil, nil
	}
	content, err := os.ReadFile(path)
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
	content, err := os.ReadFile(path)
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
