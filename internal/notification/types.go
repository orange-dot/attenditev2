package notification

import (
	"time"
)

// NotificationType represents the type of notification channel
type NotificationType string

const (
	NotificationTypePush  NotificationType = "push"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeInApp NotificationType = "in_app"
)

// NotificationPriority represents notification priority
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityUrgent   NotificationPriority = "urgent"
	PriorityCritical NotificationPriority = "critical"
)

// NotificationStatus represents notification delivery status
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusRead      NotificationStatus = "read"
	StatusFailed    NotificationStatus = "failed"
	StatusExpired   NotificationStatus = "expired"
)

// Notification represents a notification to be sent
type Notification struct {
	ID          string               `json:"id"`
	Type        NotificationType     `json:"type"`
	Priority    NotificationPriority `json:"priority"`
	Status      NotificationStatus   `json:"status"`

	// Recipient info
	RecipientID   string `json:"recipient_id"`
	RecipientType string `json:"recipient_type"` // user, role, agency, group
	RecipientName string `json:"recipient_name,omitempty"`

	// Contact details (resolved from recipient)
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	DeviceToken string `json:"device_token,omitempty"`

	// Content
	Subject string         `json:"subject"`
	Body    string         `json:"body"`
	Data    map[string]any `json:"data,omitempty"`

	// Metadata
	EventID       string `json:"event_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`

	// Timing
	ScheduledAt time.Time  `json:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Retry info
	RetryCount   int       `json:"retry_count"`
	LastRetryAt  *time.Time `json:"last_retry_at,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`

	// Audit
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Template represents a notification template
type Template struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Type        NotificationType `json:"type"`
	Subject     string           `json:"subject"`
	Body        string           `json:"body"`
	Variables   []string         `json:"variables,omitempty"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// DeliveryReceipt represents a delivery confirmation
type DeliveryReceipt struct {
	NotificationID string             `json:"notification_id"`
	Status         NotificationStatus `json:"status"`
	Timestamp      time.Time          `json:"timestamp"`
	Provider       string             `json:"provider"`
	ProviderID     string             `json:"provider_id,omitempty"`
	ErrorCode      string             `json:"error_code,omitempty"`
	ErrorMessage   string             `json:"error_message,omitempty"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	Period          string                       `json:"period"`
	TotalSent       int64                        `json:"total_sent"`
	TotalDelivered  int64                        `json:"total_delivered"`
	TotalFailed     int64                        `json:"total_failed"`
	TotalRead       int64                        `json:"total_read"`
	ByType          map[NotificationType]int64   `json:"by_type"`
	ByPriority      map[NotificationPriority]int64 `json:"by_priority"`
	ByStatus        map[NotificationStatus]int64 `json:"by_status"`
	AverageDeliveryTime time.Duration            `json:"average_delivery_time"`
	DeliveryRate    float64                      `json:"delivery_rate"`
}

// UserPreferences represents user notification preferences
type UserPreferences struct {
	UserID    string `json:"user_id"`

	// Channel preferences
	EnablePush  bool `json:"enable_push"`
	EnableSMS   bool `json:"enable_sms"`
	EnableEmail bool `json:"enable_email"`
	EnableInApp bool `json:"enable_in_app"`

	// Priority thresholds
	PushMinPriority   NotificationPriority `json:"push_min_priority"`
	SMSMinPriority    NotificationPriority `json:"sms_min_priority"`
	EmailMinPriority  NotificationPriority `json:"email_min_priority"`

	// Quiet hours
	QuietHoursEnabled bool   `json:"quiet_hours_enabled"`
	QuietHoursStart   string `json:"quiet_hours_start,omitempty"` // HH:MM format
	QuietHoursEnd     string `json:"quiet_hours_end,omitempty"`

	// Override for critical
	AlwaysAllowCritical bool `json:"always_allow_critical"`

	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultUserPreferences returns default preferences
func DefaultUserPreferences(userID string) *UserPreferences {
	return &UserPreferences{
		UserID:              userID,
		EnablePush:          true,
		EnableSMS:           true,
		EnableEmail:         true,
		EnableInApp:         true,
		PushMinPriority:     PriorityNormal,
		SMSMinPriority:      PriorityHigh,
		EmailMinPriority:    PriorityNormal,
		QuietHoursEnabled:   false,
		AlwaysAllowCritical: true,
		UpdatedAt:           time.Now(),
	}
}
