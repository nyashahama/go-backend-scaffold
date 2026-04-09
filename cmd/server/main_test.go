package main

import (
	"context"
	"testing"

	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/notification"
)

type fakeSender struct{}

func (f *fakeSender) SendPasswordReset(_ context.Context, _, _ string) error {
	return nil
}

func TestValidateRuntimeConfig_RejectsNoopSenderInProduction(t *testing.T) {
	cfg := &config.Config{
		Env:        "production",
		AppBaseURL: "https://app.example.com",
	}

	err := validateRuntimeConfig(cfg, &notification.NoopSender{})
	if err == nil {
		t.Fatal("expected error for noop sender in production, got nil")
	}
}

func TestValidateRuntimeConfig_AllowsNoopSenderInDevelopment(t *testing.T) {
	cfg := &config.Config{
		Env: "development",
	}

	if err := validateRuntimeConfig(cfg, &notification.NoopSender{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeConfig_RejectsMissingAppBaseURLInProduction(t *testing.T) {
	cfg := &config.Config{
		Env: "production",
	}

	err := validateRuntimeConfig(cfg, &fakeSender{})
	if err == nil {
		t.Fatal("expected error for missing APP_BASE_URL in production, got nil")
	}
}

func TestValidateRuntimeConfig_RejectsMissingResendConfigInProduction(t *testing.T) {
	cfg := &config.Config{
		Env:           "production",
		AppBaseURL:    "https://app.example.com",
		ResendAPIKey:  "",
		EmailFrom:     "no-reply@example.com",
		EmailFromName: "Example App",
	}

	err := validateRuntimeConfig(cfg, &fakeSender{})
	if err == nil {
		t.Fatal("expected error for missing resend config in production, got nil")
	}
}

func TestBuildNotificationSender_ReturnsNoopWithoutResendConfig(t *testing.T) {
	cfg := &config.Config{Env: "development"}

	sender, err := buildNotificationSender(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sender.(*notification.NoopSender); !ok {
		t.Fatalf("sender type = %T, want *notification.NoopSender", sender)
	}
}

func TestBuildNotificationSender_RejectsPartialResendConfig(t *testing.T) {
	cfg := &config.Config{
		ResendAPIKey:  "re_test_123",
		EmailFrom:     "no-reply@example.com",
		EmailFromName: "",
	}

	_, err := buildNotificationSender(cfg)
	if err == nil {
		t.Fatal("expected error for partial resend config, got nil")
	}
}

func TestBuildNotificationSender_ReturnsResendSenderWhenConfigured(t *testing.T) {
	cfg := &config.Config{
		ResendAPIKey:  "re_test_123",
		EmailFrom:     "no-reply@example.com",
		EmailFromName: "Example App",
	}

	sender, err := buildNotificationSender(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := sender.(*notification.ResendSender); !ok {
		t.Fatalf("sender type = %T, want *notification.ResendSender", sender)
	}
}
