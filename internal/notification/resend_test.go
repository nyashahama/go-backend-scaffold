package notification

import (
	"context"
	"errors"
	"testing"
)

type capturedEmail struct {
	From    string
	To      []string
	Subject string
	HTML    string
	ReplyTo []string
}

func TestNewResendSender_RejectsMissingConfig(t *testing.T) {
	_, err := NewResendSender("", "no-reply@example.com", "Example App")
	if err == nil {
		t.Fatal("expected error for missing api key, got nil")
	}
}

func TestResendSender_SendPasswordResetFormatsMessage(t *testing.T) {
	var captured capturedEmail
	sender := &ResendSender{
		fromEmail: "no-reply@example.com",
		fromName:  "Example App",
		sendEmail: func(_ context.Context, msg resendEmail) error {
			captured = capturedEmail{
				From:    msg.From,
				To:      msg.To,
				Subject: msg.Subject,
				HTML:    msg.HTML,
				ReplyTo: msg.ReplyTo,
			}
			return nil
		},
	}

	err := sender.SendPasswordReset(context.Background(), "user@example.com", "https://app.example.com/reset?token=abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.From != "Example App <no-reply@example.com>" {
		t.Fatalf("From=%q, want %q", captured.From, "Example App <no-reply@example.com>")
	}
	if len(captured.To) != 1 || captured.To[0] != "user@example.com" {
		t.Fatalf("To=%v, want [user@example.com]", captured.To)
	}
	if captured.Subject != "Reset your password" {
		t.Fatalf("Subject=%q, want %q", captured.Subject, "Reset your password")
	}
	if captured.HTML == "" {
		t.Fatal("expected HTML body to be populated")
	}
	if len(captured.ReplyTo) != 1 || captured.ReplyTo[0] != "no-reply@example.com" {
		t.Fatalf("ReplyTo=%v, want [no-reply@example.com]", captured.ReplyTo)
	}
}

func TestResendSender_SendPasswordResetReturnsProviderError(t *testing.T) {
	sender := &ResendSender{
		fromEmail: "no-reply@example.com",
		fromName:  "Example App",
		sendEmail: func(_ context.Context, msg resendEmail) error {
			return errors.New("provider down")
		},
	}

	err := sender.SendPasswordReset(context.Background(), "user@example.com", "https://app.example.com/reset?token=abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
