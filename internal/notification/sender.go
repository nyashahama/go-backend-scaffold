package notification

import "context"

// Sender is implemented by any service that can send transactional emails.
// The scaffold ships with NoopSender. Wire in a real implementation per project.
type Sender interface {
	SendPasswordReset(ctx context.Context, to, resetURL string) error
}

// NoopSender discards all notifications. Used in tests and as the default
// in cmd/server/main.go until a real email provider is configured.
type NoopSender struct{}

func (n *NoopSender) SendPasswordReset(_ context.Context, _, _ string) error {
	return nil
}
