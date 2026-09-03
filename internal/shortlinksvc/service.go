package shortlinksvc

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	"sync"
	"time"

	"integin/internal/domain/shortlink"
)

var (
	ErrNotFound      = errors.New("short link not found")
	ErrRevoked       = errors.New("short link revoked")
	ErrExpired       = errors.New("short link expired")
	ErrCodeCollision = errors.New("code collision")
)

type pgError interface {
	Code() string
}

type Service struct {
	repo        shortlink.Repository
	codeLen     int
	defaultHMAC string
	httpClient  *http.Client
}

func New(repo shortlink.Repository, codeLen int, defaultHMAC string) *Service {
	if codeLen <= 0 {
		codeLen = 6
	}
	return &Service{
		repo:        repo,
		codeLen:     codeLen,
		defaultHMAC: defaultHMAC,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (s *Service) CreateShortLink(ctx context.Context, req shortlink.CreateRequest) (string, error) {
	var expiresAt *time.Time
	if req.TTL > 0 {
		t := time.Now().Add(req.TTL)
		expiresAt = &t
	}

	hmacAlgorithm := "HS256"
	if req.HMACAlgorithm != nil && *req.HMACAlgorithm != "" {
		hmacAlgorithm = *req.HMACAlgorithm
	}

	var hmacSecretRef *string
	var hmacSignature *string

	if req.HMACSecretRef != nil && *req.HMACSecretRef != "" {
		hmacSecretRef = req.HMACSecretRef
	} else if s.defaultHMAC != "" {
		hmacSecretRef = &s.defaultHMAC
	}

	for i := 0; i < 5; i++ {
		code := generateCode(s.codeLen)
		if hmacSecretRef != nil {
			signature := s.signCode(code, *hmacSecretRef, hmacAlgorithm)
			hmacSignature = &signature
		}
		err := s.repo.Create(ctx, code, req.TargetURL, expiresAt, req.WebhookURL, req.CustomDomain, hmacSecretRef, &hmacAlgorithm, hmacSignature)
		if err == nil {
			if hmacSecretRef != nil {
				return code + "." + *hmacSignature, nil
			}
			return code, nil
		}
		var pgErr pgError
		if errors.As(err, &pgErr) && pgErr.Code() == "23505" {
			continue
		}
		return "", err
	}
	return "", ErrCodeCollision
}

func (s *Service) signCode(code, secret, algorithm string) string {
	if algorithm != "HS256" {
		algorithm = "HS256"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(code))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Service) verifyCode(code, signature, secret, algorithm string) bool {
	expected := s.signCode(code, secret, algorithm)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *Service) ResolveShortLink(ctx context.Context, code string) (string, error) {
	baseCode, signature := splitCode(code)

	sl, err := s.repo.Get(ctx, baseCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	if sl.HMACSecretRef != nil && *sl.HMACSecretRef != "" {
		if signature == "" {
			return "", shortlink.ErrHMACRequired
		}
		algorithm := "HS256"
		if sl.HMACAlgorithm != nil && *sl.HMACAlgorithm != "" {
			algorithm = *sl.HMACAlgorithm
		}
		secret, err := s.getHMACSecret(ctx, *sl.HMACSecretRef)
		if err != nil {
			return "", shortlink.ErrInvalidHMAC
		}
		if !s.verifyCode(baseCode, signature, secret, algorithm) {
			return "", shortlink.ErrInvalidHMAC
		}
	}

	if sl.RevokedAt != nil {
		return "", ErrRevoked
	}
	if sl.ExpiresAt != nil && time.Now().After(*sl.ExpiresAt) {
		return "", ErrExpired
	}
	// Fire and forget scan count increment
	go s.repo.IncrementScanCount(context.Background(), baseCode)
	return sl.TargetURL, nil
}

func splitCode(code string) (baseCode, signature string) {
	parts := strings.SplitN(code, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return code, ""
}

func (s *Service) getHMACSecret(ctx context.Context, ref string) (string, error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid secret ref format")
	}
	tenantID := parts[0]
	version := 0
	if parts[1] != "latest" {
		fmt.Sscanf(parts[1], "%d", &version)
	}
	var secret string
	if version > 0 {
		hmacSecret, err := s.repo.GetHMACSecret(ctx, tenantID, version)
		if err != nil {
			return "", err
		}
		secret = hmacSecret.Secret
	} else {
		hmacSecret, err := s.repo.GetActiveHMACSecret(ctx, tenantID)
		if err != nil {
			return "", err
		}
		secret = hmacSecret.Secret
	}
	return secret, nil
}

func (s *Service) RecordScan(ctx context.Context, req shortlink.ScanEventRequest) error {
	// Parse user agent to extract device, OS, browser
	deviceType, os, browser := parseUserAgent(req.UserAgent)

	// Determine country/region/city from IP (would use GeoIP in production)
	country, region, city := lookupGeoIP(req.IP)

	event := shortlink.ScanEvent{
		ShortLinkCode: req.Code,
		Timestamp:     time.Now(),
		IP:            req.IP,
		Country:       country,
		Region:        region,
		City:          city,
		DeviceType:    deviceType,
		OS:            os,
		Browser:       browser,
		Referrer:      req.Referrer,
		UTMSource:     req.UTMSource,
		UTMMedium:     req.UTMMedium,
		UTMCampaign:   req.UTMCampaign,
		UTMTerm:       req.UTMTerm,
		UTMContent:    req.UTMContent,
	}

	// Increment scan count, record event, check anomalies and deliver webhook
	go func() {
		ctx := context.Background()
		_ = s.repo.IncrementScanCount(ctx, req.Code)
		_ = s.repo.RecordScanEvent(ctx, event)
		if _, err := s.CheckAnomalies(ctx, event); err == nil {
			_ = s.ProcessAlertWebhooks(ctx)
		}
		if err := s.DeliverWebhook(ctx, req.Code, event); err == nil {
			_ = s.ProcessWebhookRetries(ctx)
		}
	}()

	return nil
}

func (s *Service) RevokeShortLink(ctx context.Context, code string) error {
	return s.repo.Revoke(ctx, code)
}

func (s *Service) GetStats(ctx context.Context, code string) (*shortlink.ShortLink, error) {
	return s.repo.GetStats(ctx, code)
}

func (s *Service) GetScanStats(ctx context.Context, code string, since *time.Time) (*shortlink.ScanStats, error) {
	return s.repo.GetScanStats(ctx, code, since)
}

func (s *Service) GetScanEvents(ctx context.Context, code string, limit, offset int) ([]shortlink.ScanEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetScanEvents(ctx, code, limit, offset)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]shortlink.ShortLink, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *Service) GenerateQRZip(_ context.Context, codes []string, size int, level string, baseURL string) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	if size > 1024 {
		size = 1024
	}
	recLevel := qrcode.Medium
	switch level {
	case "L":
		recLevel = qrcode.Low
	case "M":
		recLevel = qrcode.Medium
	case "Q":
		recLevel = qrcode.High
	case "H":
		recLevel = qrcode.Highest
	}
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for _, code := range codes {
		url := baseURL + "/" + code
		png, err := qrcode.Encode(url, recLevel, size)
		if err != nil {
			continue
		}
		w, err := zw.Create(code + ".png")
		if err != nil {
			continue
		}
		_, _ = w.Write(png)
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Service) DeliverWebhook(ctx context.Context, code string, event shortlink.ScanEvent) error {
	sl, err := s.repo.Get(ctx, code)
	if err != nil {
		return err
	}
	if sl.WebhookURL == nil || *sl.WebhookURL == "" {
		return nil // No webhook configured
	}

	payload := map[string]interface{}{
		"event":      "short_link.scan",
		"code":       code,
		"target_url": sl.TargetURL,
		"scan": map[string]interface{}{
			"timestamp":   event.Timestamp,
			"ip":          event.IP,
			"country":     event.Country,
			"region":      event.Region,
			"city":        event.City,
			"device_type": event.DeviceType,
			"os":          event.OS,
			"browser":     event.Browser,
			"referrer":    event.Referrer,
			"utm": map[string]string{
				"source":   event.UTMSource,
				"medium":   event.UTMMedium,
				"campaign": event.UTMCampaign,
				"term":     event.UTMTerm,
				"content":  event.UTMContent,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create webhook delivery record for retry tracking
	deliveryReq := shortlink.CreateWebhookDeliveryRequest{
		ShortLinkCode: code,
		Payload:       payloadBytes,
		MaxAttempts:   6,
	}
	delivery, err := s.repo.CreateWebhookDelivery(ctx, deliveryReq)
	if err != nil {
		// If we can't create delivery record, fall back to direct delivery
		return s.deliverWebhookDirect(ctx, *sl.WebhookURL, payloadBytes)
	}

	// Attempt immediate delivery
	return s.attemptWebhookDelivery(ctx, delivery, *sl.WebhookURL)
}

func (s *Service) deliverWebhookDirect(ctx context.Context, webhookURL string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "INTEGIN-ShortLink/1.0")
	payloadHash := sha256.Sum256(payload)
	req.Header.Set("Idempotency-Key", "integin-direct-"+hex.EncodeToString(payloadHash[:8]))

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook delivery failed with status %d", resp.StatusCode)
	}

	return nil
}

func (s *Service) attemptWebhookDelivery(ctx context.Context, delivery *shortlink.WebhookDelivery, webhookURL string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(delivery.Payload))
	if err != nil {
		s.handleDeliveryFailure(ctx, delivery, err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "INTEGIN-ShortLink/1.0")

	// Native Idempotency & Delivery Tracking (Konveyor / Enterprise Distributed Standard)
	// Guarantees downstream ERP/billing systems can deduplicate retries cleanly.
	payloadHash := sha256.Sum256(delivery.Payload)
	idempotencyKey := fmt.Sprintf("integin-%d-%s", delivery.ID, hex.EncodeToString(payloadHash[:8]))
	req.Header.Set("Idempotency-Key", idempotencyKey)
	req.Header.Set("X-Integin-Delivery-ID", strconv.FormatInt(delivery.ID, 10))
	req.Header.Set("X-Integin-Attempt", strconv.Itoa(delivery.Attempt+1))

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		s.handleDeliveryFailure(ctx, delivery, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err := fmt.Errorf("webhook delivery failed with status %d", resp.StatusCode)
		s.handleDeliveryFailure(ctx, delivery, err)
		return err
	}

	// Success - mark as delivered
	return s.repo.UpdateWebhookDeliveryStatus(ctx, delivery.ID, shortlink.WebhookDeliveryStatusDelivered, delivery.Attempt, "", nil)
}

func (s *Service) handleDeliveryFailure(ctx context.Context, delivery *shortlink.WebhookDelivery, err error) {
	newAttempt := delivery.Attempt + 1
	lastError := err.Error()

	if newAttempt >= delivery.MaxAttempts {
		// Max retries exceeded - move to DLQ
		_ = s.repo.CreateDLQEntry(ctx, delivery, fmt.Sprintf("Max retries (%d) exceeded: %s", delivery.MaxAttempts, lastError))
		return
	}

	// Schedule retry
	nextRetryAt := s.calculateNextRetry(newAttempt)
	_ = s.repo.UpdateWebhookDeliveryStatus(ctx, delivery.ID, shortlink.WebhookDeliveryStatusFailed, newAttempt, lastError, nextRetryAt)
}

func (s *Service) calculateNextRetry(attempt int) *time.Time {
	intervals := []time.Duration{
		1 * time.Minute,  // 1st retry (attempt = 1): 1 minute
		5 * time.Minute,  // 2nd retry (attempt = 2): 5 minutes
		15 * time.Minute, // 3rd retry (attempt = 3): 15 minutes
		1 * time.Hour,    // 4th retry (attempt = 4): 1 hour
		6 * time.Hour,    // 5th retry (attempt = 5): 6 hours
		24 * time.Hour,   // 6th retry (attempt = 6): 24 hours
	}
	idx := attempt - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(intervals) {
		idx = len(intervals) - 1
	}
	t := time.Now().Add(intervals[idx])
	return &t
}

func (s *Service) ProcessWebhookRetries(ctx context.Context) error {
	deliveries, err := s.repo.GetPendingWebhookDeliveries(ctx, 100)
	if err != nil {
		return err
	}

	if len(deliveries) == 0 {
		return nil
	}

	// Batch fetch all short links for the deliveries
	codes := make([]string, len(deliveries))
	for i, d := range deliveries {
		codes[i] = d.ShortLinkCode
	}

	shortLinks, err := s.repo.GetByCodes(ctx, codes)
	if err != nil {
		return err
	}

	// Build a map for quick lookup
	shortLinkMap := make(map[string]*shortlink.ShortLink, len(shortLinks))
	for _, sl := range shortLinks {
		shortLinkMap[sl.Code] = sl
	}

	// Process deliveries concurrently with bounded parallelism (max 5 concurrent HTTP calls)
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, delivery := range deliveries {
		sl, ok := shortLinkMap[delivery.ShortLinkCode]
		if !ok {
			// Short link not found, mark as dead letter
			_ = s.repo.UpdateWebhookDeliveryStatus(ctx, delivery.ID, shortlink.WebhookDeliveryStatusDeadLetter, delivery.Attempt, "Short link not found", nil)
			continue
		}
		if sl.WebhookURL == nil || *sl.WebhookURL == "" {
			_ = s.repo.UpdateWebhookDeliveryStatus(ctx, delivery.ID, shortlink.WebhookDeliveryStatusDeadLetter, delivery.Attempt, "Webhook URL not configured", nil)
			continue
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break
		}

		// If context was cancelled while waiting for semaphore, abort remaining batch
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(dl shortlink.WebhookDelivery, webhookURL string) {
			defer wg.Done()
			defer func() { <-sem }()
			s.attemptWebhookDelivery(ctx, &dl, webhookURL)
		}(delivery, *sl.WebhookURL)
	}

	wg.Wait()
	return nil
}

func (s *Service) GetDLQEntries(ctx context.Context, req shortlink.ListDLQRequest) ([]shortlink.DLQEntry, int, error) {
	return s.repo.GetDLQEntries(ctx, req)
}

func (s *Service) RetryDLQEntry(ctx context.Context, req shortlink.RetryDLQRequest) error {
	delivery, err := s.repo.RetryDLQEntry(ctx, req)
	if err != nil {
		return err
	}

	sl, err := s.repo.Get(ctx, delivery.ShortLinkCode)
	if err != nil {
		return err
	}
	if sl.WebhookURL == nil || *sl.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured for short link %s", delivery.ShortLinkCode)
	}

	// Attempt immediate delivery
	return s.attemptWebhookDelivery(ctx, delivery, *sl.WebhookURL)
}

func (s *Service) ResolveDLQEntry(ctx context.Context, id int64, resolvedBy string) error {
	return s.repo.ResolveDLQEntry(ctx, id, resolvedBy)
}

func generateCode(length int) string {
	b := make([]byte, (length*3+3)/4) // base64url encoding overhead
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

// parseUserAgent extracts device type, OS, and browser from User-Agent string
func parseUserAgent(ua string) (deviceType, os, browser string) {
	if ua == "" {
		return "Unknown", "Unknown", "Unknown"
	}
	uaLower := toLower(ua)

	// Device type
	switch {
	case contains(uaLower, "mobile") || contains(uaLower, "android") || contains(uaLower, "iphone"):
		deviceType = "Mobile"
	case contains(uaLower, "tablet") || contains(uaLower, "ipad"):
		deviceType = "Tablet"
	default:
		deviceType = "Desktop"
	}

	// OS
	switch {
	case contains(uaLower, "windows"):
		os = "Windows"
	case contains(uaLower, "macintosh") || contains(uaLower, "mac os"):
		os = "macOS"
	case contains(uaLower, "linux"):
		os = "Linux"
	case contains(uaLower, "android"):
		os = "Android"
	case contains(uaLower, "iphone") || contains(uaLower, "ipad") || contains(uaLower, "ipod"):
		os = "iOS"
	default:
		os = "Unknown"
	}

	// Browser
	switch {
	case contains(uaLower, "edg"):
		browser = "Edge"
	case contains(uaLower, "chrome") && !contains(uaLower, "edg"):
		browser = "Chrome"
	case contains(uaLower, "firefox"):
		browser = "Firefox"
	case contains(uaLower, "safari") && !contains(uaLower, "chrome"):
		browser = "Safari"
	case contains(uaLower, "opera") || contains(uaLower, "opr"):
		browser = "Opera"
	default:
		browser = "Unknown"
	}

	return deviceType, os, browser
}

func lookupGeoIP(ip string) (country, region, city string) {
	// In production, integrate with a GeoIP database like MaxMind
	// For now, return empty strings - can be enhanced with a GeoIP library
	return "", "", ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = byte(r + 32)
		} else {
			result[i] = byte(r)
		}
	}
	return string(result)
}

// Anomaly detection methods

func (s *Service) CreateAnomalyRule(ctx context.Context, req shortlink.CreateAnomalyRuleRequest) (*shortlink.AnomalyRule, error) {
	if req.Config.Cooldown == 0 {
		req.Config.Cooldown = 15 * time.Minute
	}
	if req.Config.Threshold == 0 {
		req.Config.Threshold = 100
	}
	if req.Config.Window == 0 {
		req.Config.Window = 1 * time.Hour
	}

	rule := &shortlink.AnomalyRule{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Enabled:     true,
	}
	err := s.repo.CreateAnomalyRule(ctx, rule)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *Service) GetAnomalyRule(ctx context.Context, id int64) (*shortlink.AnomalyRule, error) {
	return s.repo.GetAnomalyRule(ctx, id)
}

func (s *Service) UpdateAnomalyRule(ctx context.Context, id int64, req shortlink.UpdateAnomalyRuleRequest) (*shortlink.AnomalyRule, error) {
	err := s.repo.UpdateAnomalyRule(ctx, id, req)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAnomalyRule(ctx, id)
}

func (s *Service) DeleteAnomalyRule(ctx context.Context, id int64) error {
	return s.repo.DeleteAnomalyRule(ctx, id)
}

func (s *Service) ListAnomalyRules(ctx context.Context, req shortlink.ListAnomalyRulesRequest) ([]shortlink.AnomalyRule, int, error) {
	return s.repo.ListAnomalyRules(ctx, req)
}

func (s *Service) CreateAnomalyAlert(ctx context.Context, req shortlink.CreateAnomalyAlertRequest) (*shortlink.AnomalyAlert, error) {
	alert := &shortlink.AnomalyAlert{
		RuleID:        req.RuleID,
		TenantID:      req.TenantID,
		ShortLinkCode: req.ShortLinkCode,
		Type:          req.Type,
		Status:        shortlink.AlertStatusFiring,
		Message:       req.Message,
		Details:       req.Details,
	}
	err := s.repo.CreateAnomalyAlert(ctx, alert)
	if err != nil {
		return nil, err
	}
	return alert, nil
}

func (s *Service) GetAnomalyAlert(ctx context.Context, id int64) (*shortlink.AnomalyAlert, error) {
	return s.repo.GetAnomalyAlert(ctx, id)
}

func (s *Service) UpdateAnomalyAlert(ctx context.Context, id int64, req shortlink.UpdateAnomalyAlertRequest) (*shortlink.AnomalyAlert, error) {
	err := s.repo.UpdateAnomalyAlert(ctx, id, req)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAnomalyAlert(ctx, id)
}

func (s *Service) ListAnomalyAlerts(ctx context.Context, req shortlink.ListAnomalyAlertsRequest) ([]shortlink.AnomalyAlert, int, error) {
	return s.repo.ListAnomalyAlerts(ctx, req)
}

func (s *Service) CheckAnomalies(ctx context.Context, event shortlink.ScanEvent) ([]shortlink.AnomalyCheckResult, error) {
	// Get tenant from short link - we need to extract it from the short link
	// For now, we'll need a way to get tenant ID. Let's assume it's in the short link code or we fetch it
	sl, err := s.repo.Get(ctx, event.ShortLinkCode)
	if err != nil {
		return nil, err
	}

	// For now, use a default tenant or extract from HMAC secret ref
	tenantID := "default"
	if sl.HMACSecretRef != nil {
		parts := strings.SplitN(*sl.HMACSecretRef, ":", 2)
		if len(parts) == 2 {
			tenantID = parts[0]
		}
	}

	rules, err := s.repo.GetActiveAnomalyRules(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var results []shortlink.AnomalyCheckResult
	now := time.Now()

	for _, rule := range rules {
		triggered, alert := s.evaluateRule(ctx, &rule, event, sl, now)
		if triggered && alert != nil {
			results = append(results, shortlink.AnomalyCheckResult{
				Triggered: true,
				Rule:      &rule,
				Alert:     alert,
			})
		}
	}

	return results, nil
}

func (s *Service) evaluateRule(ctx context.Context, rule *shortlink.AnomalyRule, event shortlink.ScanEvent, sl *shortlink.ShortLink, now time.Time) (bool, *shortlink.AnomalyAlert) {
	// Check cooldown/deduplication
	if rule.Config.Cooldown > 0 {
		since := now.Add(-rule.Config.Cooldown)
		recentAlert, err := s.repo.GetRecentAlertForDedup(ctx, rule.ID, event.ShortLinkCode, since)
		if err != nil {
			return false, nil
		}
		if recentAlert != nil {
			return false, nil // Deduplicated
		}
	}

	var triggered bool
	var message string
	details := make(map[string]interface{})

	switch rule.Type {
	case shortlink.AnomalyTypeGeo:
		triggered, message, details = s.checkGeoAnomaly(rule, event)
	case shortlink.AnomalyTypeFrequency:
		triggered, message, details = s.checkFrequencyAnomaly(ctx, rule, event)
	case shortlink.AnomalyTypePattern:
		triggered, message, details = s.checkPatternAnomaly(ctx, rule, event)
	}

	if triggered {
		alert := &shortlink.AnomalyAlert{
			RuleID:        rule.ID,
			TenantID:      rule.TenantID,
			ShortLinkCode: event.ShortLinkCode,
			Type:          rule.Type,
			Status:        shortlink.AlertStatusFiring,
			Message:       message,
			Details:       details,
		}
		return true, alert
	}

	return false, nil
}

func (s *Service) checkGeoAnomaly(rule *shortlink.AnomalyRule, event shortlink.ScanEvent) (bool, string, map[string]interface{}) {
	details := map[string]interface{}{
		"country": event.Country,
	}

	if len(rule.Config.Countries) > 0 {
		// Check if country is in allowed list
		allowed := false
		for _, c := range rule.Config.Countries {
			if strings.EqualFold(c, event.Country) {
				allowed = true
				break
			}
		}
		if !allowed && event.Country != "" {
			details["allowed_countries"] = rule.Config.Countries
			return true, fmt.Sprintf("Unexpected country: %s (not in allowed list)", event.Country), details
		}
	}
	return false, "", details
}

func (s *Service) checkFrequencyAnomaly(ctx context.Context, rule *shortlink.AnomalyRule, event shortlink.ScanEvent) (bool, string, map[string]interface{}) {
	since := time.Now().Add(-rule.Config.Window)
	stats, err := s.repo.GetScanStats(ctx, event.ShortLinkCode, &since)
	if err != nil {
		return false, "", nil
	}

	details := map[string]interface{}{
		"total_scans":    stats.TotalScans,
		"window_seconds": rule.Config.Window.Seconds(),
		"threshold":      rule.Config.Threshold,
	}

	if stats.TotalScans >= rule.Config.Threshold {
		return true, fmt.Sprintf("Scan burst detected: %d scans in %s (threshold: %d)", stats.TotalScans, rule.Config.Window.String(), rule.Config.Threshold), details
	}
	return false, "", details
}

func (s *Service) checkPatternAnomaly(ctx context.Context, rule *shortlink.AnomalyRule, event shortlink.ScanEvent) (bool, string, map[string]interface{}) {
	// Check for bot-like behavior: high scans from same IP, no referrer, etc.
	since := time.Now().Add(-rule.Config.Window)
	stats, err := s.repo.GetScanStats(ctx, event.ShortLinkCode, &since)
	if err != nil {
		return false, "", nil
	}

	details := map[string]interface{}{
		"total_scans":    stats.TotalScans,
		"unique_ips":     stats.UniqueIPs,
		"window_seconds": rule.Config.Window.Seconds(),
		"threshold":      rule.Config.Threshold,
	}

	// Bot-like patterns:
	// 1. Many scans from few IPs
	// 2. No referrer (direct access)
	// 3. Consistent timing

	if stats.TotalScans > 0 && stats.UniqueIPs > 0 {
		ratio := float64(stats.TotalScans) / float64(stats.UniqueIPs)
		details["scans_per_ip"] = ratio
		
		// If ratio > threshold and low unique IPs, might be bot
		if ratio >= float64(rule.Config.Threshold) && stats.UniqueIPs < 10 {
			return true, fmt.Sprintf("Bot-like pattern detected: %.1f scans/IP from %d unique IPs", ratio, stats.UniqueIPs), details
		}
	}

	return false, "", details
}

func (s *Service) ProcessAlertWebhooks(ctx context.Context) error {
	alerts, err := s.repo.GetPendingAlertWebhooks(ctx, 50)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		rule, err := s.repo.GetAnomalyRule(ctx, alert.RuleID)
		if err != nil {
			continue
		}

		// Check if webhook is configured
		if rule.Config.WebhookURL == nil || *rule.Config.WebhookURL == "" {
			continue
		}

		payload := map[string]interface{}{
			"event":        "anomaly_alert",
			"alert_id":     alert.ID,
			"rule_id":      alert.RuleID,
			"rule_name":    rule.Name,
			"type":         alert.Type,
			"status":       alert.Status,
			"message":      alert.Message,
			"details":      alert.Details,
			"short_link":   alert.ShortLinkCode,
			"tenant_id":    alert.TenantID,
			"fired_at":     alert.FiredAt,
		}

		payloadBytes, _ := json.Marshal(payload)

		req := shortlink.WebhookDeliveryRequest{
			URL:       *rule.Config.WebhookURL,
			Payload:   payloadBytes,
			Timeout:   10 * time.Second,
			MaxRetries: 3,
		}

		err = s.repo.DeliverAlertWebhook(ctx, req)
		if err != nil {
			s.repo.UpdateAlertWebhookStatus(ctx, alert.ID, shortlink.AlertStatusFiring, err.Error())
		}
	}

	return nil
}

func (s *Service) GetHMACSecret(ctx context.Context, tenantID string, version int) (*shortlink.HMACSecret, error) {
	return s.repo.GetHMACSecret(ctx, tenantID, version)
}

func (s *Service) GetActiveHMACSecret(ctx context.Context, tenantID string) (*shortlink.HMACSecret, error) {
	return s.repo.GetActiveHMACSecret(ctx, tenantID)
}

func (s *Service) CreateHMACSecret(ctx context.Context, tenantID string, secret string, algorithm string) (*shortlink.HMACSecret, error) {
	if algorithm == "" {
		algorithm = "HS256"
	}
	return s.repo.CreateHMACSecret(ctx, tenantID, secret, algorithm)
}

func (s *Service) RevokeHMACSecret(ctx context.Context, tenantID string, version int) error {
	return s.repo.RevokeHMACSecret(ctx, tenantID, version)
}

func (s *Service) ListHMACSecrets(ctx context.Context, tenantID string) ([]shortlink.HMACSecret, error) {
	return s.repo.ListHMACSecrets(ctx, tenantID)
}

// Dashboard Analytics methods

func (s *Service) GetDashboardAnalytics(ctx context.Context, req shortlink.AnalyticsRequest) (*shortlink.DashboardAnalytics, error) {
	since, until := s.parseTimeRange(req.TimeRange, req.CustomStart, req.CustomEnd)
	
	overview, err := s.repo.GetDashboardOverview(ctx, req.TenantID, since, until)
	if err != nil {
		return nil, err
	}

	timeSeries, err := s.repo.GetTimeSeries(ctx, req.TenantID, since, until, "")
	if err != nil {
		return nil, err
	}

	geoHeatmap, err := s.repo.GetGeoHeatmap(ctx, req.TenantID, since, until, "")
	if err != nil {
		return nil, err
	}

	devices, osList, browsers, err := s.repo.GetDeviceAnalytics(ctx, req.TenantID, since, until)
	if err != nil {
		return nil, err
	}

	funnel, err := s.repo.GetFunnel(ctx, req.TenantID, since, until)
	if err != nil {
		return nil, err
	}

	topAssets, err := s.repo.GetTopAssets(ctx, req.TenantID, since, until, 20)
	if err != nil {
		return nil, err
	}

	return &shortlink.DashboardAnalytics{
		TotalScans:    overview.TotalScans,
		UniqueIPs:     overview.UniqueIPs,
		TopAssets:     topAssets,
		TimeSeries:    timeSeries,
		GeoHeatmap:    geoHeatmap,
		Devices:       devices,
		OS:            osList,
		Browsers:      browsers,
		Funnel:        *funnel,
		TimeRange:     req.TimeRange,
		CustomStart:   req.CustomStart,
		CustomEnd:     req.CustomEnd,
	}, nil
}

func (s *Service) GetTimeSeries(ctx context.Context, req shortlink.TimeSeriesRequest) ([]shortlink.TimeSeriesPoint, error) {
	since, until := s.parseTimeRange(req.TimeRange, req.CustomStart, req.CustomEnd)
	return s.repo.GetTimeSeries(ctx, req.TenantID, since, until, req.Interval)
}

func (s *Service) GetGeoHeatmap(ctx context.Context, req shortlink.GeoHeatmapRequest) ([]shortlink.GeoHeatmapPoint, error) {
	since, until := s.parseTimeRange(req.TimeRange, req.CustomStart, req.CustomEnd)
	return s.repo.GetGeoHeatmap(ctx, req.TenantID, since, until, req.Country)
}

func (s *Service) GetDeviceAnalytics(ctx context.Context, req shortlink.DeviceAnalyticsRequest) ([]shortlink.DeviceBreakdown, []shortlink.OSBreakdown, []shortlink.BrowserBreakdown, error) {
	since, until := s.parseTimeRange(req.TimeRange, req.CustomStart, req.CustomEnd)
	return s.repo.GetDeviceAnalytics(ctx, req.TenantID, since, until)
}

func (s *Service) GetFunnel(ctx context.Context, req shortlink.FunnelRequest) (*shortlink.FunnelData, error) {
	since, until := s.parseTimeRange(req.TimeRange, req.CustomStart, req.CustomEnd)
	return s.repo.GetFunnel(ctx, req.TenantID, since, until)
}

func (s *Service) GetTopAssets(ctx context.Context, req shortlink.TopAssetsRequest) ([]shortlink.TopAsset, error) {
	since, until := s.parseTimeRange(req.TimeRange, req.CustomStart, req.CustomEnd)
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetTopAssets(ctx, req.TenantID, since, until, limit)
}

func (s *Service) parseTimeRange(tr shortlink.TimeRange, customStart, customEnd *time.Time) (*time.Time, *time.Time) {
	now := time.Now()
	var since, until *time.Time
	
	switch tr {
	case shortlink.TimeRange1H:
		s := now.Add(-1 * time.Hour)
		since = &s
		until = &now
	case shortlink.TimeRange24H:
		s := now.Add(-24 * time.Hour)
		since = &s
		until = &now
	case shortlink.TimeRange7D:
		s := now.Add(-7 * 24 * time.Hour)
		since = &s
		until = &now
	case shortlink.TimeRange30D:
		s := now.Add(-30 * 24 * time.Hour)
		since = &s
		until = &now
	case shortlink.TimeRangeCustom:
		since = customStart
		until = customEnd
	default:
		s := now.Add(-24 * time.Hour)
		since = &s
		until = &now
	}
	
	return since, until
}
