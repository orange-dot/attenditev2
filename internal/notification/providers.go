package notification

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockPushProvider is a mock push notification provider for testing
type MockPushProvider struct {
	mu           sync.RWMutex
	sent         map[string]*Notification
	failOnSend   bool
	sendDelay    time.Duration
}

// NewMockPushProvider creates a new mock push provider
func NewMockPushProvider() *MockPushProvider {
	return &MockPushProvider{
		sent: make(map[string]*Notification),
	}
}

// Send sends a push notification (mock implementation)
func (p *MockPushProvider) Send(ctx context.Context, notification *Notification) error {
	if p.sendDelay > 0 {
		time.Sleep(p.sendDelay)
	}

	if p.failOnSend {
		return fmt.Errorf("mock send failure")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.sent[notification.ID] = notification
	fmt.Printf("[MOCK PUSH] To: %s, Subject: %s\n", notification.RecipientID, notification.Subject)

	return nil
}

// GetDeliveryStatus returns delivery status (mock)
func (p *MockPushProvider) GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, ok := p.sent[notificationID]; ok {
		return &DeliveryReceipt{
			NotificationID: notificationID,
			Status:         StatusDelivered,
			Timestamp:      time.Now(),
			Provider:       "mock_push",
		}, nil
	}

	return nil, fmt.Errorf("notification not found")
}

// SetFailOnSend sets whether Send should fail
func (p *MockPushProvider) SetFailOnSend(fail bool) {
	p.failOnSend = fail
}

// SetSendDelay sets artificial delay for Send
func (p *MockPushProvider) SetSendDelay(delay time.Duration) {
	p.sendDelay = delay
}

// GetSentNotifications returns all sent notifications
func (p *MockPushProvider) GetSentNotifications() []*Notification {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Notification, 0, len(p.sent))
	for _, n := range p.sent {
		result = append(result, n)
	}
	return result
}

// MockSMSProvider is a mock SMS provider for testing
type MockSMSProvider struct {
	mu         sync.RWMutex
	sent       map[string]*Notification
	failOnSend bool
	sendDelay  time.Duration
}

// NewMockSMSProvider creates a new mock SMS provider
func NewMockSMSProvider() *MockSMSProvider {
	return &MockSMSProvider{
		sent: make(map[string]*Notification),
	}
}

// Send sends an SMS (mock implementation)
func (p *MockSMSProvider) Send(ctx context.Context, notification *Notification) error {
	if p.sendDelay > 0 {
		time.Sleep(p.sendDelay)
	}

	if p.failOnSend {
		return fmt.Errorf("mock send failure")
	}

	if notification.Phone == "" {
		return fmt.Errorf("no phone number provided")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.sent[notification.ID] = notification
	fmt.Printf("[MOCK SMS] To: %s, Message: %s\n", notification.Phone, notification.Body[:min(50, len(notification.Body))])

	return nil
}

// GetDeliveryStatus returns delivery status (mock)
func (p *MockSMSProvider) GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, ok := p.sent[notificationID]; ok {
		return &DeliveryReceipt{
			NotificationID: notificationID,
			Status:         StatusDelivered,
			Timestamp:      time.Now(),
			Provider:       "mock_sms",
		}, nil
	}

	return nil, fmt.Errorf("notification not found")
}

// SetFailOnSend sets whether Send should fail
func (p *MockSMSProvider) SetFailOnSend(fail bool) {
	p.failOnSend = fail
}

// MockEmailProvider is a mock email provider for testing
type MockEmailProvider struct {
	mu         sync.RWMutex
	sent       map[string]*Notification
	failOnSend bool
	sendDelay  time.Duration
}

// NewMockEmailProvider creates a new mock email provider
func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		sent: make(map[string]*Notification),
	}
}

// Send sends an email (mock implementation)
func (p *MockEmailProvider) Send(ctx context.Context, notification *Notification) error {
	if p.sendDelay > 0 {
		time.Sleep(p.sendDelay)
	}

	if p.failOnSend {
		return fmt.Errorf("mock send failure")
	}

	if notification.Email == "" {
		return fmt.Errorf("no email address provided")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.sent[notification.ID] = notification
	fmt.Printf("[MOCK EMAIL] To: %s, Subject: %s\n", notification.Email, notification.Subject)

	return nil
}

// GetDeliveryStatus returns delivery status (mock)
func (p *MockEmailProvider) GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if _, ok := p.sent[notificationID]; ok {
		return &DeliveryReceipt{
			NotificationID: notificationID,
			Status:         StatusDelivered,
			Timestamp:      time.Now(),
			Provider:       "mock_email",
		}, nil
	}

	return nil, fmt.Errorf("notification not found")
}

// SetFailOnSend sets whether Send should fail
func (p *MockEmailProvider) SetFailOnSend(fail bool) {
	p.failOnSend = fail
}

// ConsoleProvider logs notifications to console (for development)
type ConsoleProvider struct {
	prefix string
}

// NewConsoleProvider creates a console logging provider
func NewConsoleProvider(prefix string) *ConsoleProvider {
	return &ConsoleProvider{prefix: prefix}
}

// Send logs the notification to console
func (p *ConsoleProvider) Send(ctx context.Context, notification *Notification) error {
	fmt.Printf("\n[%s NOTIFICATION]\n", p.prefix)
	fmt.Printf("  ID:        %s\n", notification.ID)
	fmt.Printf("  Type:      %s\n", notification.Type)
	fmt.Printf("  Priority:  %s\n", notification.Priority)
	fmt.Printf("  Recipient: %s (%s)\n", notification.RecipientID, notification.RecipientType)
	fmt.Printf("  Subject:   %s\n", notification.Subject)
	fmt.Printf("  Body:\n%s\n", notification.Body)
	if notification.EventID != "" {
		fmt.Printf("  Event:     %s\n", notification.EventID)
	}
	fmt.Println()
	return nil
}

// GetDeliveryStatus returns delivery status
func (p *ConsoleProvider) GetDeliveryStatus(ctx context.Context, notificationID string) (*DeliveryReceipt, error) {
	return &DeliveryReceipt{
		NotificationID: notificationID,
		Status:         StatusDelivered,
		Timestamp:      time.Now(),
		Provider:       "console",
	}, nil
}

// Helper
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
