package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/serbia-gov/platform/internal/coordination"
)

// Service is the notification service
type Service struct {
	// Providers
	pushProvider  PushProvider
	smsProvider   SMSProvider
	emailProvider EmailProvider

	// State
	mu       sync.RWMutex
	pending  map[string]*Notification
	stats    *NotificationStats
	prefs    map[string]*UserPreferences

	// Processing
	notifCh chan *Notification
	workers int

	// Lifecycle
	started bool
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// Configuration
	config ServiceConfig
}

// PushProvider interface for push notification providers
type PushProvider interface {
	Send(ctx context.Context, notification *Notification) error
	GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error)
}

// SMSProvider interface for SMS providers
type SMSProvider interface {
	Send(ctx context.Context, notification *Notification) error
	GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error)
}

// EmailProvider interface for email providers
type EmailProvider interface {
	Send(ctx context.Context, notification *Notification) error
	GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error)
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Workers         int
	BufferSize      int
	RetryAttempts   int
	RetryDelay      time.Duration
	ExpirationTime  time.Duration
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Workers:        4,
		BufferSize:     1000,
		RetryAttempts:  3,
		RetryDelay:     30 * time.Second,
		ExpirationTime: 24 * time.Hour,
	}
}

// NewService creates a new notification service
func NewService(
	pushProvider PushProvider,
	smsProvider SMSProvider,
	emailProvider EmailProvider,
	config ServiceConfig,
) *Service {
	return &Service{
		pushProvider:  pushProvider,
		smsProvider:   smsProvider,
		emailProvider: emailProvider,
		pending:       make(map[string]*Notification),
		stats:         &NotificationStats{},
		prefs:         make(map[string]*UserPreferences),
		notifCh:       make(chan *Notification, config.BufferSize),
		workers:       config.Workers,
		stopCh:        make(chan struct{}),
		config:        config,
	}
}

// Start starts the notification service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("service already started")
	}
	s.started = true
	s.mu.Unlock()

	// Start workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	return nil
}

// Stop stops the notification service
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return fmt.Errorf("service not started")
	}
	s.mu.Unlock()

	close(s.stopCh)
	s.wg.Wait()

	return nil
}

// Send implements coordination.NotificationSender
func (s *Service) Send(ctx context.Context, coordNotif coordination.Notification) error {
	notif := s.convertFromCoordination(coordNotif)
	return s.SendNotification(ctx, notif)
}

// SendNotification sends a notification
func (s *Service) SendNotification(ctx context.Context, notification *Notification) error {
	if notification.ID == "" {
		notification.ID = generateNotificationID()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	notification.UpdatedAt = time.Now()
	notification.Status = StatusPending

	// Apply user preferences
	if err := s.applyPreferences(notification); err != nil {
		return err
	}

	// Store in pending
	s.mu.Lock()
	s.pending[notification.ID] = notification
	s.mu.Unlock()

	// Submit for processing
	select {
	case s.notifCh <- notification:
		return nil
	default:
		return fmt.Errorf("notification buffer full")
	}
}

// worker processes notifications from the channel
func (s *Service) worker(ctx context.Context, id int) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case notif := <-s.notifCh:
			s.processNotification(ctx, notif)
		}
	}
}

// processNotification processes a single notification
func (s *Service) processNotification(ctx context.Context, notif *Notification) {
	var err error

	switch notif.Type {
	case NotificationTypePush:
		if s.pushProvider != nil {
			err = s.pushProvider.Send(ctx, notif)
		} else {
			err = fmt.Errorf("push provider not configured")
		}
	case NotificationTypeSMS:
		if s.smsProvider != nil {
			err = s.smsProvider.Send(ctx, notif)
		} else {
			err = fmt.Errorf("sms provider not configured")
		}
	case NotificationTypeEmail:
		if s.emailProvider != nil {
			err = s.emailProvider.Send(ctx, notif)
		} else {
			err = fmt.Errorf("email provider not configured")
		}
	case NotificationTypeInApp:
		// In-app notifications are just stored
		err = nil
	default:
		err = fmt.Errorf("unknown notification type: %s", notif.Type)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		notif.ErrorMessage = err.Error()
		notif.RetryCount++
		now := time.Now()
		notif.LastRetryAt = &now

		if notif.RetryCount >= s.config.RetryAttempts {
			notif.Status = StatusFailed
			s.updateStats(notif, false)
		} else {
			// Re-queue for retry
			go func() {
				time.Sleep(s.config.RetryDelay)
				select {
				case s.notifCh <- notif:
				default:
				}
			}()
		}
	} else {
		now := time.Now()
		notif.SentAt = &now
		notif.Status = StatusSent
		s.updateStats(notif, true)
	}

	notif.UpdatedAt = time.Now()
}

// updateStats updates notification statistics
func (s *Service) updateStats(notif *Notification, success bool) {
	if s.stats.ByType == nil {
		s.stats.ByType = make(map[NotificationType]int64)
	}
	if s.stats.ByPriority == nil {
		s.stats.ByPriority = make(map[NotificationPriority]int64)
	}
	if s.stats.ByStatus == nil {
		s.stats.ByStatus = make(map[NotificationStatus]int64)
	}

	s.stats.TotalSent++
	s.stats.ByType[notif.Type]++
	s.stats.ByPriority[notif.Priority]++

	if success {
		s.stats.TotalDelivered++
		s.stats.ByStatus[StatusDelivered]++
	} else {
		s.stats.TotalFailed++
		s.stats.ByStatus[StatusFailed]++
	}

	// Update delivery rate
	if s.stats.TotalSent > 0 {
		s.stats.DeliveryRate = float64(s.stats.TotalDelivered) / float64(s.stats.TotalSent)
	}
}

// applyPreferences applies user preferences to notification
func (s *Service) applyPreferences(notif *Notification) error {
	prefs := s.getUserPreferences(notif.RecipientID)
	if prefs == nil {
		return nil // No preferences, allow all
	}

	// Check if channel is enabled
	switch notif.Type {
	case NotificationTypePush:
		if !prefs.EnablePush {
			return fmt.Errorf("push notifications disabled for user")
		}
		if !s.meetsMinPriority(notif.Priority, prefs.PushMinPriority) {
			if prefs.AlwaysAllowCritical && notif.Priority == PriorityCritical {
				// Allow critical
			} else {
				return fmt.Errorf("notification priority below threshold")
			}
		}
	case NotificationTypeSMS:
		if !prefs.EnableSMS {
			return fmt.Errorf("sms notifications disabled for user")
		}
		if !s.meetsMinPriority(notif.Priority, prefs.SMSMinPriority) {
			if prefs.AlwaysAllowCritical && notif.Priority == PriorityCritical {
				// Allow critical
			} else {
				return fmt.Errorf("notification priority below threshold")
			}
		}
	case NotificationTypeEmail:
		if !prefs.EnableEmail {
			return fmt.Errorf("email notifications disabled for user")
		}
		if !s.meetsMinPriority(notif.Priority, prefs.EmailMinPriority) {
			if prefs.AlwaysAllowCritical && notif.Priority == PriorityCritical {
				// Allow critical
			} else {
				return fmt.Errorf("notification priority below threshold")
			}
		}
	case NotificationTypeInApp:
		if !prefs.EnableInApp {
			return fmt.Errorf("in-app notifications disabled for user")
		}
	}

	// Check quiet hours
	if prefs.QuietHoursEnabled && !s.isAllowedDuringQuietHours(notif, prefs) {
		if prefs.AlwaysAllowCritical && notif.Priority == PriorityCritical {
			// Allow critical during quiet hours
		} else {
			return fmt.Errorf("quiet hours active")
		}
	}

	return nil
}

// meetsMinPriority checks if notification priority meets minimum threshold
func (s *Service) meetsMinPriority(priority, minPriority NotificationPriority) bool {
	order := map[NotificationPriority]int{
		PriorityLow:      1,
		PriorityNormal:   2,
		PriorityHigh:     3,
		PriorityUrgent:   4,
		PriorityCritical: 5,
	}
	return order[priority] >= order[minPriority]
}

// isAllowedDuringQuietHours checks if notification is allowed during quiet hours
func (s *Service) isAllowedDuringQuietHours(notif *Notification, prefs *UserPreferences) bool {
	if !prefs.QuietHoursEnabled {
		return true
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	// Simple comparison (assumes start < end, no midnight crossing for now)
	if currentTime >= prefs.QuietHoursStart && currentTime <= prefs.QuietHoursEnd {
		return false
	}

	return true
}

// getUserPreferences returns user notification preferences
func (s *Service) getUserPreferences(userID string) *UserPreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.prefs[userID]
}

// SetUserPreferences sets user notification preferences
func (s *Service) SetUserPreferences(prefs *UserPreferences) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefs[prefs.UserID] = prefs
}

// GetNotification returns a notification by ID
func (s *Service) GetNotification(id string) (*Notification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.pending[id]
	return n, ok
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(notificationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notif, ok := s.pending[notificationID]
	if !ok {
		return fmt.Errorf("notification not found: %s", notificationID)
	}

	now := time.Now()
	notif.ReadAt = &now
	notif.Status = StatusRead
	notif.UpdatedAt = now

	s.stats.TotalRead++
	s.stats.ByStatus[StatusRead]++

	return nil
}

// GetStats returns notification statistics
func (s *Service) GetStats() NotificationStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.stats
}

// convertFromCoordination converts coordination.Notification to Notification
func (s *Service) convertFromCoordination(cn coordination.Notification) *Notification {
	return &Notification{
		ID:            cn.ID,
		Type:          NotificationType(cn.Type),
		Priority:      convertPriority(cn.Priority),
		Status:        StatusPending,
		RecipientID:   cn.Recipient.ID,
		RecipientType: cn.Recipient.Type,
		RecipientName: cn.Recipient.Name,
		Phone:         cn.Recipient.Phone,
		Email:         cn.Recipient.Email,
		DeviceToken:   cn.Recipient.DeviceToken,
		Subject:       cn.Subject,
		Body:          cn.Body,
		Data:          cn.Data,
		EventID:       cn.EventID,
		ScheduledAt:   cn.ScheduledAt,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// convertPriority converts coordination.Priority to NotificationPriority
func convertPriority(p coordination.Priority) NotificationPriority {
	switch p {
	case coordination.PriorityLow:
		return PriorityLow
	case coordination.PriorityNormal:
		return PriorityNormal
	case coordination.PriorityHigh:
		return PriorityHigh
	case coordination.PriorityUrgent:
		return PriorityUrgent
	case coordination.PriorityCritical:
		return PriorityCritical
	default:
		return PriorityNormal
	}
}

// Helper functions

func generateNotificationID() string {
	return fmt.Sprintf("ntf-%d", time.Now().UnixNano())
}
