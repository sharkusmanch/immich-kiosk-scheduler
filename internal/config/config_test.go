package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduleEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		entry   ScheduleEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: ScheduleEntry{
				Name:  "christmas",
				Album: "abc-123",
				Start: "11-15",
				End:   "01-01",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			entry: ScheduleEntry{
				Album: "abc-123",
				Start: "11-15",
				End:   "01-01",
			},
			wantErr: true,
		},
		{
			name: "missing album",
			entry: ScheduleEntry{
				Name:  "christmas",
				Start: "11-15",
				End:   "01-01",
			},
			wantErr: true,
		},
		{
			name: "invalid start date format",
			entry: ScheduleEntry{
				Name:  "christmas",
				Album: "abc-123",
				Start: "2024-11-15",
				End:   "01-01",
			},
			wantErr: true,
		},
		{
			name: "invalid end date format",
			entry: ScheduleEntry{
				Name:  "christmas",
				Album: "abc-123",
				Start: "11-15",
				End:   "january-01",
			},
			wantErr: true,
		},
		{
			name: "invalid month",
			entry: ScheduleEntry{
				Name:  "test",
				Album: "abc-123",
				Start: "13-15",
				End:   "01-01",
			},
			wantErr: true,
		},
		{
			name: "invalid day",
			entry: ScheduleEntry{
				Name:  "test",
				Album: "abc-123",
				Start: "11-32",
				End:   "01-01",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				KioskURL:     "https://kiosk.example.com",
				DefaultAlbum: "default-album-id",
				Port:         8080,
				Schedule: []ScheduleEntry{
					{Name: "test", Album: "abc", Start: "01-01", End: "12-31"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing kiosk url",
			config: Config{
				DefaultAlbum: "default-album-id",
				Port:         8080,
			},
			wantErr: true,
		},
		{
			name: "missing default album",
			config: Config{
				KioskURL: "https://kiosk.example.com",
				Port:     8080,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: Config{
				KioskURL:     "https://kiosk.example.com",
				DefaultAlbum: "default-album-id",
				Port:         0,
			},
			wantErr: true,
		},
		{
			name: "port too high",
			config: Config{
				KioskURL:     "https://kiosk.example.com",
				DefaultAlbum: "default-album-id",
				Port:         70000,
			},
			wantErr: true,
		},
		{
			name: "invalid schedule entry",
			config: Config{
				KioskURL:     "https://kiosk.example.com",
				DefaultAlbum: "default-album-id",
				Port:         8080,
				Schedule: []ScheduleEntry{
					{Name: "", Album: "abc", Start: "01-01", End: "12-31"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
kiosk_url: "https://kiosk.example.com"
default_album: "default-123"
port: 9090
log_level: "debug"
passthrough_params:
  - transition
  - duration
schedule:
  - name: christmas
    album: "christmas-456"
    start: "11-15"
    end: "01-01"
  - name: summer
    album: "summer-789"
    start: "06-21"
    end: "09-21"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "https://kiosk.example.com", cfg.KioskURL)
	assert.Equal(t, "default-123", cfg.DefaultAlbum)
	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, []string{"transition", "duration"}, cfg.PassthroughParams)
	assert.Len(t, cfg.Schedule, 2)
	assert.Equal(t, "christmas", cfg.Schedule[0].Name)
	assert.Equal(t, "christmas-456", cfg.Schedule[0].Album)
}

func TestLoadFromEnvVars(t *testing.T) {
	// Create minimal config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
kiosk_url: "https://default.example.com"
default_album: "default-123"
port: 8080
schedule: []
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set env vars (should override file)
	t.Setenv("IKS_KIOSK_URL", "https://env.example.com")
	t.Setenv("IKS_PORT", "3000")
	t.Setenv("IKS_LOG_LEVEL", "warn")

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "https://env.example.com", cfg.KioskURL)
	assert.Equal(t, 3000, cfg.Port)
	assert.Equal(t, "warn", cfg.LogLevel)
	// File value should be preserved for non-overridden fields
	assert.Equal(t, "default-123", cfg.DefaultAlbum)
}

func TestDefaultValues(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Minimal config with only required fields
	configContent := `
kiosk_url: "https://kiosk.example.com"
default_album: "default-123"
schedule: []
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Empty(t, cfg.PassthroughParams)
}

func TestPassthroughParamsSanitization(t *testing.T) {
	tests := []struct {
		name     string
		param    string
		expected string
		valid    bool
	}{
		{"valid alphanumeric", "transition", "transition", true},
		{"valid with underscore", "show_time", "show_time", true},
		{"valid with hyphen", "image-fit", "image-fit", true},
		{"invalid with spaces", "my param", "", false},
		{"invalid with special chars", "param<script>", "", false},
		{"invalid empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeParam(tt.param)
			assert.Equal(t, tt.valid, valid)
			if valid {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
