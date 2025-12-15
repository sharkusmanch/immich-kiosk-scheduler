package scheduler

import (
	"testing"
	"time"

	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMonthDay(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMonth int
		wantDay   int
		wantErr   bool
	}{
		{"valid date", "11-15", 11, 15, false},
		{"january first", "01-01", 1, 1, false},
		{"december last", "12-31", 12, 31, false},
		{"invalid format", "2024-11-15", 0, 0, true},
		{"invalid month", "13-01", 0, 0, true},
		{"invalid day", "01-32", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			month, day, err := ParseMonthDay(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMonth, month)
				assert.Equal(t, tt.wantDay, day)
			}
		})
	}
}

func TestScheduler_GetAlbum_SimpleRange(t *testing.T) {
	cfg := &config.Config{
		DefaultAlbum: "default-album",
		Schedule: []config.ScheduleEntry{
			{Name: "summer", Album: "summer-album", Start: "06-21", End: "09-21"},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{"before range", time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC), "default-album"},
		{"start of range", time.Date(2024, 6, 21, 0, 0, 0, 0, time.UTC), "summer-album"},
		{"middle of range", time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC), "summer-album"},
		{"end of range", time.Date(2024, 9, 21, 0, 0, 0, 0, time.UTC), "summer-album"},
		{"after range", time.Date(2024, 9, 22, 0, 0, 0, 0, time.UTC), "default-album"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			album := s.GetAlbumForDate(tt.date)
			assert.Equal(t, tt.expected, album)
		})
	}
}

func TestScheduler_GetAlbum_YearWrap(t *testing.T) {
	cfg := &config.Config{
		DefaultAlbum: "default-album",
		Schedule: []config.ScheduleEntry{
			{Name: "christmas", Album: "christmas-album", Start: "11-15", End: "01-01"},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{"before christmas season", time.Date(2024, 11, 14, 0, 0, 0, 0, time.UTC), "default-album"},
		{"start of christmas", time.Date(2024, 11, 15, 0, 0, 0, 0, time.UTC), "christmas-album"},
		{"december", time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC), "christmas-album"},
		{"new years day", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "christmas-album"},
		{"after new years", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), "default-album"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			album := s.GetAlbumForDate(tt.date)
			assert.Equal(t, tt.expected, album)
		})
	}
}

func TestScheduler_GetAlbum_MultipleSchedules(t *testing.T) {
	cfg := &config.Config{
		DefaultAlbum: "favorites",
		Schedule: []config.ScheduleEntry{
			{Name: "christmas", Album: "christmas-album", Start: "11-15", End: "01-01"},
			{Name: "spring", Album: "spring-album", Start: "03-20", End: "06-20"},
			{Name: "summer", Album: "summer-album", Start: "06-21", End: "09-21"},
			{Name: "fall", Album: "fall-album", Start: "09-22", End: "11-14"},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{"winter (default)", time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), "favorites"},
		{"spring", time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC), "spring-album"},
		{"summer", time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC), "summer-album"},
		{"fall", time.Date(2024, 10, 15, 0, 0, 0, 0, time.UTC), "fall-album"},
		{"christmas", time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC), "christmas-album"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			album := s.GetAlbumForDate(tt.date)
			assert.Equal(t, tt.expected, album)
		})
	}
}

func TestScheduler_GetAlbum_FirstMatchWins(t *testing.T) {
	// Overlapping schedules - first match should win
	cfg := &config.Config{
		DefaultAlbum: "default-album",
		Schedule: []config.ScheduleEntry{
			{Name: "special", Album: "special-album", Start: "12-20", End: "12-26"},
			{Name: "christmas", Album: "christmas-album", Start: "11-15", End: "01-01"},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	// Dec 25 matches both, but "special" is first in the list
	album := s.GetAlbumForDate(time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "special-album", album)

	// Nov 20 only matches christmas
	album = s.GetAlbumForDate(time.Date(2024, 11, 20, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "christmas-album", album)
}

func TestScheduler_GetCurrentAlbum(t *testing.T) {
	cfg := &config.Config{
		DefaultAlbum: "default-album",
		Schedule:     []config.ScheduleEntry{},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	// Should return default when no schedules match
	album := s.GetCurrentAlbum()
	assert.Equal(t, "default-album", album)
}

func TestScheduler_GetCurrentScheduleName(t *testing.T) {
	cfg := &config.Config{
		DefaultAlbum: "default-album",
		Schedule: []config.ScheduleEntry{
			{Name: "summer", Album: "summer-album", Start: "06-21", End: "09-21"},
		},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	// Test with a specific date in summer
	name := s.GetScheduleNameForDate(time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "summer", name)

	// Test with a date outside any schedule
	name = s.GetScheduleNameForDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "default", name)
}

func TestScheduler_EmptySchedule(t *testing.T) {
	cfg := &config.Config{
		DefaultAlbum: "default-album",
		Schedule:     []config.ScheduleEntry{},
	}

	s, err := New(cfg)
	require.NoError(t, err)

	album := s.GetAlbumForDate(time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "default-album", album)
}
