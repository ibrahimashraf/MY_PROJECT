package notifications

type Category string

const (
	CategoryInspectionAssigned  Category = "INSPECTION_ASSIGNED"
	CategoryInspectionSubmitted Category = "INSPECTION_SUBMITTED"
	CategoryInspectionReturned  Category = "INSPECTION_RETURNED"
	CategoryInspectionApproved  Category = "INSPECTION_APPROVED"
	CategoryCertificateIssued   Category = "CERTIFICATE_ISSUED"
	CategoryCertificateExpiring Category = "CERTIFICATE_EXPIRING"
	CategoryCertificateRevoked  Category = "CERTIFICATE_REVOKED"
	CategoryFindingRaised       Category = "FINDING_RAISED"
	CategoryCorrectiveActionDue Category = "CORRECTIVE_ACTION_DUE"
	CategoryCalibrationExpired  Category = "CALIBRATION_EXPIRED"
	CategorySecurityEvent       Category = "SECURITY_EVENT"
	CategorySyncHeld            Category = "SYNC_HELD"
	CategorySystemAlert         Category = "SYSTEM_ALERT"
)

func (c Category) Mandatory() bool { return c == CategorySecurityEvent || c == CategorySystemAlert }

type Channel string

const (
	ChannelInApp Channel = "IN_APP"
	ChannelEmail Channel = "EMAIL"
)

type Preference struct {
	Category Category `json:"category"`
	Channel  Channel  `json:"channel"`
	Enabled  bool     `json:"enabled"`
}
type DeliveryState string

const (
	DeliveryQueued    DeliveryState = "QUEUED"
	DeliverySending   DeliveryState = "SENDING"
	DeliverySent      DeliveryState = "SENT"
	DeliveryDelivered DeliveryState = "DELIVERED"
	DeliveryBounced   DeliveryState = "BOUNCED"
	DeliveryFailed    DeliveryState = "FAILED"
	DeliveryRetry     DeliveryState = "RETRY"
	DeliveryGivenUp   DeliveryState = "GIVEN_UP"
)
