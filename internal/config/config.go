// Package config handles configuration loading from files, environment variables, and CLI flags.
package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// ScheduleEntry represents a single schedule entry that maps a date range to an album.
type ScheduleEntry struct {
	Name  string `mapstructure:"name"`
	Album string `mapstructure:"album"`
	Start string `mapstructure:"start"` // Format: MM-DD
	End   string `mapstructure:"end"`   // Format: MM-DD
}

// Config holds all application configuration.
type Config struct {
	KioskURL          string          `mapstructure:"kiosk_url"`
	DefaultAlbum      string          `mapstructure:"default_album"`
	Port              int             `mapstructure:"port"`
	LogLevel          string          `mapstructure:"log_level"`
	PassthroughParams []string        `mapstructure:"passthrough_params"`
	Schedule          []ScheduleEntry `mapstructure:"schedule"`
	MetricsUsername   string          `mapstructure:"metrics_username"`
	MetricsPassword   string          `mapstructure:"metrics_password"`
}

// dateRegex validates MM-DD format.
var dateRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)

// paramRegex validates safe parameter names (alphanumeric, underscore, hyphen).
var paramRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Validate checks if the schedule entry is valid.
func (s *ScheduleEntry) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("schedule entry name is required")
	}
	if strings.TrimSpace(s.Album) == "" {
		return fmt.Errorf("schedule entry album is required")
	}
	if !dateRegex.MatchString(s.Start) {
		return fmt.Errorf("invalid start date format %q, expected MM-DD", s.Start)
	}
	if !dateRegex.MatchString(s.End) {
		return fmt.Errorf("invalid end date format %q, expected MM-DD", s.End)
	}

	// Validate month/day values
	if err := validateDate(s.Start); err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}
	if err := validateDate(s.End); err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	return nil
}

// validateDate checks if the MM-DD string represents a valid date.
func validateDate(date string) error {
	parts := strings.Split(date, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid date format")
	}

	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return fmt.Errorf("invalid month: %s", parts[0])
	}

	day, err := strconv.Atoi(parts[1])
	if err != nil || day < 1 || day > 31 {
		return fmt.Errorf("invalid day: %s", parts[1])
	}

	// Check days per month (simplified, doesn't account for leap years)
	maxDays := []int{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day > maxDays[month-1] {
		return fmt.Errorf("day %d is invalid for month %d", day, month)
	}

	return nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.KioskURL) == "" {
		return fmt.Errorf("kiosk_url is required")
	}

	// Validate kiosk_url scheme
	parsedURL, err := url.Parse(c.KioskURL)
	if err != nil {
		return fmt.Errorf("invalid kiosk_url: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("kiosk_url must use http or https scheme, got %q", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("kiosk_url must include a host")
	}

	if strings.TrimSpace(c.DefaultAlbum) == "" {
		return fmt.Errorf("default_album is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	for i, entry := range c.Schedule {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("schedule entry %d (%s): %w", i, entry.Name, err)
		}
	}

	return nil
}

// SanitizeParam validates and sanitizes a parameter name.
// Returns the sanitized parameter and whether it's valid.
func SanitizeParam(param string) (string, bool) {
	param = strings.TrimSpace(param)
	if param == "" {
		return "", false
	}
	if !paramRegex.MatchString(param) {
		return "", false
	}
	return param, true
}

// Load reads configuration from file and environment variables.
// Environment variables take precedence over file values.
// Environment variable prefix is IKS_ (e.g., IKS_KIOSK_URL).
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("port", 8080)
	v.SetDefault("log_level", "info")
	v.SetDefault("passthrough_params", []string{})
	v.SetDefault("schedule", []ScheduleEntry{})

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Bind environment variables
	v.SetEnvPrefix("IKS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Manually bind specific env vars for proper override behavior
	_ = v.BindEnv("kiosk_url", "IKS_KIOSK_URL")
	_ = v.BindEnv("default_album", "IKS_DEFAULT_ALBUM")
	_ = v.BindEnv("port", "IKS_PORT")
	_ = v.BindEnv("log_level", "IKS_LOG_LEVEL")
	_ = v.BindEnv("metrics_username", "IKS_METRICS_USERNAME")
	_ = v.BindEnv("metrics_password", "IKS_METRICS_PASSWORD")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}
