package shortlink

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("short link not found")
	ErrInvalidHMAC  = errors.New("invalid HMAC signature")
	ErrHMACRequired = errors.New("HMAC signature required but not provided")
)

type AnomalyType string

const (
	AnomalyTypeGeo       AnomalyType = "geo"
	AnomalyTypeFrequency AnomalyType = "frequency"
	AnomalyTypePattern   AnomalyType = "pattern"
)

type AlertChannel string

const (
	AlertChannelWebhook   AlertChannel = "webhook"
	AlertChannelEmail     AlertChannel = "email"
	AlertChannelSlack     AlertChannel = "slack"
	AlertChannelPagerDuty AlertChannel = "pagerduty"
)

type AlertStatus string

const (
	AlertStatusFiring      AlertStatus = "firing"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved    AlertStatus = "resolved"
)

type AnomalyRule struct {
	ID          int64
	TenantID    string
	Name        string
	Description string
	Type        AnomalyType
	Config      AlertConfig
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AlertConfig struct {
	Threshold     int64
	Window        time.Duration
	Countries     []string
	Cooldown      time.Duration
	Channels      []AlertChannel
	WebhookURL    *string
	EmailTo       *string
	SlackWebhook  *string
	PagerDutyKey  *string
}

type AnomalyAlert struct {
	ID            int64
	RuleID        int64
	TenantID      string
	ShortLinkCode string
	Type          AnomalyType
	Status        AlertStatus
	Message       string
	Details       map[string]interface{}
	FiredAt       time.Time
	AcknowledgedAt *time.Time
	AcknowledgedBy *string
	ResolvedAt    *time.Time
	ResolvedBy    *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateAnomalyRuleRequest struct {
	TenantID    string
	Name        string
	Description string
	Type        AnomalyType
	Config      AlertConfig
}

type UpdateAnomalyRuleRequest struct {
	Name        *string
	Description *string
	Config      *AlertConfig
	Enabled     *bool
}

type ListAnomalyRulesRequest struct {
	TenantID string
	Limit    int
	Offset   int
	Enabled  *bool
}

type CreateAnomalyAlertRequest struct {
	RuleID        int64
	TenantID      string
	ShortLinkCode string
	Type          AnomalyType
	Message       string
	Details       map[string]interface{}
}

type UpdateAnomalyAlertRequest struct {
	Status      *AlertStatus
	AcknowledgedBy *string
	ResolvedBy    *string
}

type ListAnomalyAlertsRequest struct {
	TenantID      string
	ShortLinkCode *string
	RuleID        *int64
	Status        *AlertStatus
	Type          *AnomalyType
	Since         *time.Time
	Limit         int
	Offset        int
}

type WebhookDeliveryRequest struct {
	URL      string
	Payload  []byte
	Headers  map[string]string
	Timeout  time.Duration
	MaxRetries int
}

type AnomalyCheckResult struct {
	Triggered bool
	Rule      *AnomalyRule
	Alert     *AnomalyAlert
}

type Repository interface {
	Create(ctx context.Context, code, targetURL string, expiresAt *time.Time, webhookURL *string, customDomain *string, hmacSecretRef *string, hmacAlgorithm *string, hmacSignature *string) error
	Get(ctx context.Context, code string) (*ShortLink, error)
	IncrementScanCount(ctx context.Context, code string) error
	RecordScanEvent(ctx context.Context, event ScanEvent) error
	GetScanEvents(ctx context.Context, code string, limit, offset int) ([]ScanEvent, error)
	GetScanStats(ctx context.Context, code string, since *time.Time) (*ScanStats, error)
	Revoke(ctx context.Context, code string) error
	List(ctx context.Context, limit, offset int) ([]ShortLink, error)
	GetStats(ctx context.Context, code string) (*ShortLink, error)
	GetByCodes(ctx context.Context, codes []string) ([]*ShortLink, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*WebhookDelivery, error)

	// Webhook delivery methods
	CreateWebhookDelivery(ctx context.Context, req CreateWebhookDeliveryRequest) (*WebhookDelivery, error)
	GetWebhookDelivery(ctx context.Context, id int64) (*WebhookDelivery, error)
	UpdateWebhookDeliveryStatus(ctx context.Context, id int64, status WebhookDeliveryStatus, attempt int, lastError string, nextRetryAt *time.Time) error
	GetPendingWebhookDeliveries(ctx context.Context, limit int) ([]WebhookDelivery, error)
	GetDLQEntries(ctx context.Context, req ListDLQRequest) ([]DLQEntry, int, error)
	CreateDLQEntry(ctx context.Context, delivery *WebhookDelivery, errorMsg string) error
	RetryDLQEntry(ctx context.Context, req RetryDLQRequest) (*WebhookDelivery, error)
	ResolveDLQEntry(ctx context.Context, id int64, resolvedBy string) error

	// HMAC secret management
	GetHMACSecret(ctx context.Context, tenantID string, version int) (*HMACSecret, error)
	GetActiveHMACSecret(ctx context.Context, tenantID string) (*HMACSecret, error)
	CreateHMACSecret(ctx context.Context, tenantID string, secret string, algorithm string) (*HMACSecret, error)
	RevokeHMACSecret(ctx context.Context, tenantID string, version int) error
	ListHMACSecrets(ctx context.Context, tenantID string) ([]HMACSecret, error)

	// Anomaly detection methods
	CreateAnomalyRule(ctx context.Context, rule *AnomalyRule) error
	GetAnomalyRule(ctx context.Context, id int64) (*AnomalyRule, error)
	UpdateAnomalyRule(ctx context.Context, id int64, req UpdateAnomalyRuleRequest) error
	DeleteAnomalyRule(ctx context.Context, id int64) error
	ListAnomalyRules(ctx context.Context, req ListAnomalyRulesRequest) ([]AnomalyRule, int, error)
	GetActiveAnomalyRules(ctx context.Context, tenantID string) ([]AnomalyRule, error)

	CreateAnomalyAlert(ctx context.Context, alert *AnomalyAlert) error
	GetAnomalyAlert(ctx context.Context, id int64) (*AnomalyAlert, error)
	UpdateAnomalyAlert(ctx context.Context, id int64, req UpdateAnomalyAlertRequest) error
	ListAnomalyAlerts(ctx context.Context, req ListAnomalyAlertsRequest) ([]AnomalyAlert, int, error)
	GetRecentAlertForDedup(ctx context.Context, ruleID int64, shortLinkCode string, since time.Time) (*AnomalyAlert, error)

	DeliverAlertWebhook(ctx context.Context, req WebhookDeliveryRequest) error
	GetPendingAlertWebhooks(ctx context.Context, limit int) ([]AnomalyAlert, error)
	UpdateAlertWebhookStatus(ctx context.Context, alertID int64, status AlertStatus, errorMsg string) error

	// Dashboard Analytics methods
	GetDashboardOverview(ctx context.Context, tenantID string, since, until *time.Time) (*DashboardOverview, error)
	GetTimeSeries(ctx context.Context, tenantID string, since, until *time.Time, interval string) ([]TimeSeriesPoint, error)
	GetGeoHeatmap(ctx context.Context, tenantID string, since, until *time.Time, country string) ([]GeoHeatmapPoint, error)
	GetDeviceAnalytics(ctx context.Context, tenantID string, since, until *time.Time) ([]DeviceBreakdown, []OSBreakdown, []BrowserBreakdown, error)
	GetFunnel(ctx context.Context, tenantID string, since, until *time.Time) (*FunnelData, error)
	GetTopAssets(ctx context.Context, tenantID string, since, until *time.Time, limit int) ([]TopAsset, error)
}

type ShortLink struct {
	Code          string
	TargetURL     string
	CreatedAt     time.Time
	ExpiresAt     *time.Time
	RevokedAt     *time.Time
	ScanCount     int64
	WebhookURL    *string
	CustomDomain  *string
	HMACSecretRef *string
	HMACAlgorithm *string
	HMACSignature *string
}

type HMACSecret struct {
	Version   int
	Secret    string
	Algorithm string
	CreatedAt time.Time
	RevokedAt *time.Time
}

type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending    WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusDelivered  WebhookDeliveryStatus = "delivered"
	WebhookDeliveryStatusFailed     WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusDeadLetter WebhookDeliveryStatus = "dead_letter"
)

type WebhookDelivery struct {
	ID            int64
	ShortLinkCode string
	Payload       []byte
	Status        WebhookDeliveryStatus
	Attempt       int
	MaxAttempts   int
	NextRetryAt   *time.Time
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeliveredAt   *time.Time
}

type DLQEntry struct {
	ID            int64
	DeliveryID    int64
	ShortLinkCode string
	Payload       []byte
	Error         string
	Attempts      int
	CreatedAt     time.Time
	ResolvedAt    *time.Time
	ResolvedBy    string
}

type CreateWebhookDeliveryRequest struct {
	ShortLinkCode string
	Payload       []byte
	MaxAttempts   int
}

type ListDLQRequest struct {
	Limit  int
	Offset int
	Status *WebhookDeliveryStatus
}

type RetryDLQRequest struct {
	DeliveryID int64
}

type ShortLinkStats struct {
	Code      string
	TargetURL string
	CreatedAt time.Time
	ExpiresAt *time.Time
	RevokedAt *time.Time
	ScanCount int64
}

type ScanEvent struct {
	ShortLinkCode string
	Timestamp     time.Time
	IP            string
	Country       string
	Region        string
	City          string
	DeviceType    string
	OS            string
	Browser       string
	Referrer      string
	UTMSource     string
	UTMMedium     string
	UTMCampaign   string
	UTMTerm       string
	UTMContent    string
}

type CreateRequest struct {
	TargetURL     string
	TTL           time.Duration
	WebhookURL    *string
	CustomDomain  *string
	HMACSecretRef *string
	HMACAlgorithm *string
}

type ScanEventRequest struct {
	Code        string
	IP          string
	UserAgent   string
	Referrer    string
	UTMSource   string
	UTMMedium   string
	UTMCampaign string
	UTMTerm     string
	UTMContent  string
}

type ScanStats struct {
	TotalScans   int64
	UniqueIPs    int64
	Countries    map[string]int64
	Devices      map[string]int64
	Browsers     map[string]int64
	Referrers    map[string]int64
	UTMSources   map[string]int64
	UTMMedia     map[string]int64
	UTMCampaigns map[string]int64
}

type TimeRange string

const (
	TimeRange1H  TimeRange = "1h"
	TimeRange24H TimeRange = "24h"
	TimeRange7D  TimeRange = "7d"
	TimeRange30D TimeRange = "30d"
	TimeRangeCustom TimeRange = "custom"
)

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
}

type GeoHeatmapPoint struct {
	Country  string `json:"country"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	Count    int64  `json:"count"`
}

type DeviceBreakdown struct {
	DeviceType string `json:"device_type"`
	Count      int64  `json:"count"`
}

type OSBreakdown struct {
	OS    string `json:"os"`
	Count int64  `json:"count"`
}

type BrowserBreakdown struct {
	Browser string `json:"browser"`
	Count   int64  `json:"count"`
}

type FunnelData struct {
	Scans       int64   `json:"scans"`
	Redirects   int64   `json:"redirects"`
	Conversions int64   `json:"conversions"`
	ScanToRedirectRate   float64 `json:"scan_to_redirect_rate"`
	RedirectToConversionRate float64 `json:"redirect_to_conversion_rate"`
}

type TopAsset struct {
	Code       string `json:"code"`
	TargetURL  string `json:"target_url"`
	ScanCount  int64  `json:"scan_count"`
	UniqueIPs  int64  `json:"unique_ips"`
}

type DashboardOverview struct {
	TotalScans int64   `json:"total_scans"`
	UniqueIPs  int64   `json:"unique_ips"`
	TopAssets  []TopAsset `json:"top_assets"`
}

type DashboardAnalytics struct {
	TotalScans     int64              `json:"total_scans"`
	UniqueIPs      int64              `json:"unique_ips"`
	TopAssets      []TopAsset         `json:"top_assets"`
	TimeSeries     []TimeSeriesPoint  `json:"time_series"`
	GeoHeatmap     []GeoHeatmapPoint  `json:"geo_heatmap"`
	Devices        []DeviceBreakdown  `json:"devices"`
	OS             []OSBreakdown      `json:"os"`
	Browsers       []BrowserBreakdown `json:"browsers"`
	Funnel         FunnelData         `json:"funnel"`
	TimeRange      TimeRange          `json:"time_range"`
	CustomStart    *time.Time         `json:"custom_start,omitempty"`
	CustomEnd      *time.Time         `json:"custom_end,omitempty"`
}

type AnalyticsRequest struct {
	TenantID   string     `json:"tenant_id"`
	TimeRange  TimeRange  `json:"time_range"`
	CustomStart *time.Time `json:"custom_start,omitempty"`
	CustomEnd   *time.Time `json:"custom_end,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}

type TopAssetsRequest struct {
	TenantID  string    `json:"tenant_id"`
	TimeRange TimeRange `json:"time_range"`
	CustomStart *time.Time `json:"custom_start,omitempty"`
	CustomEnd   *time.Time `json:"custom_end,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

type TimeSeriesRequest struct {
	TenantID    string    `json:"tenant_id"`
	TimeRange   TimeRange `json:"time_range"`
	CustomStart *time.Time `json:"custom_start,omitempty"`
	CustomEnd   *time.Time `json:"custom_end,omitempty"`
	Interval    string    `json:"interval,omitempty"` // e.g., "1h", "1d"
}

type GeoHeatmapRequest struct {
	TenantID    string    `json:"tenant_id"`
	TimeRange   TimeRange `json:"time_range"`
	CustomStart *time.Time `json:"custom_start,omitempty"`
	CustomEnd   *time.Time `json:"custom_end,omitempty"`
	Country     string    `json:"country,omitempty"` // Optional filter
}

type DeviceAnalyticsRequest struct {
	TenantID    string    `json:"tenant_id"`
	TimeRange   TimeRange `json:"time_range"`
	CustomStart *time.Time `json:"custom_start,omitempty"`
	CustomEnd   *time.Time `json:"custom_end,omitempty"`
}

type FunnelRequest struct {
	TenantID    string    `json:"tenant_id"`
	TimeRange   TimeRange `json:"time_range"`
	CustomStart *time.Time `json:"custom_start,omitempty"`
	CustomEnd   *time.Time `json:"custom_end,omitempty"`
}

type Service interface {
	CreateShortLink(ctx context.Context, req CreateRequest) (string, error)
	ResolveShortLink(ctx context.Context, code string) (string, error)
	RecordScan(ctx context.Context, req ScanEventRequest) error
	RevokeShortLink(ctx context.Context, code string) error
	GetStats(ctx context.Context, code string) (*ShortLink, error)
	GetScanStats(ctx context.Context, code string, since *time.Time) (*ScanStats, error)
	GetScanEvents(ctx context.Context, code string, limit, offset int) ([]ScanEvent, error)
	List(ctx context.Context, limit, offset int) ([]ShortLink, error)
	DeliverWebhook(ctx context.Context, code string, event ScanEvent) error

	// Webhook delivery retry methods
	ProcessWebhookRetries(ctx context.Context) error
	GetDLQEntries(ctx context.Context, req ListDLQRequest) ([]DLQEntry, int, error)
	RetryDLQEntry(ctx context.Context, req RetryDLQRequest) error
	ResolveDLQEntry(ctx context.Context, id int64, resolvedBy string) error

	// HMAC secret management
	GetHMACSecret(ctx context.Context, tenantID string, version int) (*HMACSecret, error)
	GetActiveHMACSecret(ctx context.Context, tenantID string) (*HMACSecret, error)
	CreateHMACSecret(ctx context.Context, tenantID string, secret string, algorithm string) (*HMACSecret, error)
	RevokeHMACSecret(ctx context.Context, tenantID string, version int) error
	ListHMACSecrets(ctx context.Context, tenantID string) ([]HMACSecret, error)

	// Anomaly detection methods
	CreateAnomalyRule(ctx context.Context, req CreateAnomalyRuleRequest) (*AnomalyRule, error)
	GetAnomalyRule(ctx context.Context, id int64) (*AnomalyRule, error)
	UpdateAnomalyRule(ctx context.Context, id int64, req UpdateAnomalyRuleRequest) (*AnomalyRule, error)
	DeleteAnomalyRule(ctx context.Context, id int64) error
	ListAnomalyRules(ctx context.Context, req ListAnomalyRulesRequest) ([]AnomalyRule, int, error)

	CreateAnomalyAlert(ctx context.Context, req CreateAnomalyAlertRequest) (*AnomalyAlert, error)
	GetAnomalyAlert(ctx context.Context, id int64) (*AnomalyAlert, error)
	UpdateAnomalyAlert(ctx context.Context, id int64, req UpdateAnomalyAlertRequest) (*AnomalyAlert, error)
	ListAnomalyAlerts(ctx context.Context, req ListAnomalyAlertsRequest) ([]AnomalyAlert, int, error)
	CheckAnomalies(ctx context.Context, event ScanEvent) ([]AnomalyCheckResult, error)
	ProcessAlertWebhooks(ctx context.Context) error

	// Dashboard Analytics methods
	GetDashboardAnalytics(ctx context.Context, req AnalyticsRequest) (*DashboardAnalytics, error)
	GetTimeSeries(ctx context.Context, req TimeSeriesRequest) ([]TimeSeriesPoint, error)
	GetGeoHeatmap(ctx context.Context, req GeoHeatmapRequest) ([]GeoHeatmapPoint, error)
	GetDeviceAnalytics(ctx context.Context, req DeviceAnalyticsRequest) ([]DeviceBreakdown, []OSBreakdown, []BrowserBreakdown, error)
	GetFunnel(ctx context.Context, req FunnelRequest) (*FunnelData, error)
	GetTopAssets(ctx context.Context, req TopAssetsRequest) ([]TopAsset, error)
}
