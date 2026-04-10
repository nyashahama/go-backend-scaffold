package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	minJWTSecretLength   = 32
	placeholderJWTSecret = "changeme-use-openssl-rand-base64-32"
)

type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	ResendAPIKey       string
	EmailFrom          string
	EmailFromName      string
	MetricsBearerToken string
	AppBaseURL         string
	AllowedOrigins     []string
	TrustProxyHeaders  bool
	TrustedProxyCIDRs  []*net.IPNet
	JWTExpiry          time.Duration
	RefreshExpiry      time.Duration
}

func Load() (*Config, error) {
	// Load .env in development (no-op if file doesn't exist or vars already set)
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RedisURL:           os.Getenv("REDIS_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
		EmailFrom:          os.Getenv("EMAIL_FROM"),
		EmailFromName:      os.Getenv("EMAIL_FROM_NAME"),
		MetricsBearerToken: os.Getenv("METRICS_BEARER_TOKEN"),
		AppBaseURL:         os.Getenv("APP_BASE_URL"),
	}

	var err error

	if cfg.TrustProxyHeaders, err = parseBool("TRUST_PROXY_HEADERS", false); err != nil {
		return nil, err
	}
	cfg.TrustedProxyCIDRs, err = parseCIDRs("TRUSTED_PROXY_CIDRS")
	if err != nil {
		return nil, err
	}
	cfg.JWTExpiry, err = parseDuration("JWT_EXPIRY", 15*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.RefreshExpiry, err = parseDuration("REFRESH_EXPIRY", 168*time.Hour)
	if err != nil {
		return nil, err
	}

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		cfg.AllowedOrigins = strings.Split(origins, ",")
		for i := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL": c.DatabaseURL,
		"REDIS_URL":    c.RedisURL,
		"JWT_SECRET":   c.JWTSecret,
	}

	var missing []string
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if c.JWTSecret == placeholderJWTSecret {
		return fmt.Errorf("JWT_SECRET must be changed from the example placeholder value")
	}

	if len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}

func parseBool(key string, fallback bool) (bool, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("invalid boolean for %s: %w", key, err)
	}
	return parsed, nil
}

func parseCIDRs(key string) ([]*net.IPNet, error) {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return nil, nil
	}

	var cidrs []*net.IPNet
	for _, part := range strings.Split(val, ",") {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		_, network, err := net.ParseCIDR(candidate)
		if err != nil {
			return nil, fmt.Errorf("invalid cidr for %s: %q", key, candidate)
		}
		cidrs = append(cidrs, network)
	}

	return cidrs, nil
}
