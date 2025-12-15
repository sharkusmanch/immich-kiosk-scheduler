// Package scheduler handles date-based album selection.
package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sharkusmanch/immich-kiosk-scheduler/internal/config"
)

// dateRange represents a parsed schedule entry with month/day values.
type dateRange struct {
	name       string
	album      string
	startMonth int
	startDay   int
	endMonth   int
	endDay     int
	wrapsYear  bool // true if the range crosses year boundary (e.g., Nov-Jan)
}

// Scheduler determines which album to display based on the current date.
type Scheduler struct {
	defaultAlbum string
	ranges       []dateRange
}

// New creates a new Scheduler from the given configuration.
func New(cfg *config.Config) (*Scheduler, error) {
	s := &Scheduler{
		defaultAlbum: cfg.DefaultAlbum,
		ranges:       make([]dateRange, 0, len(cfg.Schedule)),
	}

	for _, entry := range cfg.Schedule {
		startMonth, startDay, err := ParseMonthDay(entry.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start date for %q: %w", entry.Name, err)
		}

		endMonth, endDay, err := ParseMonthDay(entry.End)
		if err != nil {
			return nil, fmt.Errorf("invalid end date for %q: %w", entry.Name, err)
		}

		dr := dateRange{
			name:       entry.Name,
			album:      entry.Album,
			startMonth: startMonth,
			startDay:   startDay,
			endMonth:   endMonth,
			endDay:     endDay,
			wrapsYear:  isYearWrap(startMonth, startDay, endMonth, endDay),
		}

		s.ranges = append(s.ranges, dr)
	}

	return s, nil
}

// ParseMonthDay parses a MM-DD string into month and day integers.
func ParseMonthDay(s string) (month, day int, err error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format: expected MM-DD, got %q", s)
	}

	month, err = strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("invalid month: %s", parts[0])
	}

	day, err = strconv.Atoi(parts[1])
	if err != nil || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("invalid day: %s", parts[1])
	}

	return month, day, nil
}

// isYearWrap returns true if the date range crosses a year boundary.
// For example, Nov 15 to Jan 1 wraps the year.
func isYearWrap(startMonth, startDay, endMonth, endDay int) bool {
	startDOY := monthDayToDOY(startMonth, startDay)
	endDOY := monthDayToDOY(endMonth, endDay)
	return endDOY < startDOY
}

// monthDayToDOY converts a month/day to a day-of-year number (1-366).
// This is used for date comparisons without worrying about the actual year.
func monthDayToDOY(month, day int) int {
	// Days in each month (using non-leap year, but allowing 29 for Feb)
	daysInMonth := []int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	doy := 0
	for m := 1; m < month; m++ {
		doy += daysInMonth[m]
	}
	return doy + day
}

// GetCurrentAlbum returns the album ID for the current date.
func (s *Scheduler) GetCurrentAlbum() string {
	return s.GetAlbumForDate(time.Now())
}

// GetAlbumForDate returns the album ID for the given date.
// It evaluates schedules in order and returns the first match.
// If no schedule matches, it returns the default album.
func (s *Scheduler) GetAlbumForDate(t time.Time) string {
	month := int(t.Month())
	day := t.Day()
	currentDOY := monthDayToDOY(month, day)

	for _, r := range s.ranges {
		if s.dateInRange(currentDOY, r) {
			return r.album
		}
	}

	return s.defaultAlbum
}

// GetCurrentScheduleName returns the name of the current schedule (or "default").
func (s *Scheduler) GetCurrentScheduleName() string {
	return s.GetScheduleNameForDate(time.Now())
}

// GetScheduleNameForDate returns the name of the matching schedule for the given date.
// Returns "default" if no schedule matches.
func (s *Scheduler) GetScheduleNameForDate(t time.Time) string {
	month := int(t.Month())
	day := t.Day()
	currentDOY := monthDayToDOY(month, day)

	for _, r := range s.ranges {
		if s.dateInRange(currentDOY, r) {
			return r.name
		}
	}

	return "default"
}

// dateInRange checks if a day-of-year falls within the given date range.
func (s *Scheduler) dateInRange(currentDOY int, r dateRange) bool {
	startDOY := monthDayToDOY(r.startMonth, r.startDay)
	endDOY := monthDayToDOY(r.endMonth, r.endDay)

	if r.wrapsYear {
		// Range wraps year (e.g., Nov 15 to Jan 1)
		// Date is in range if it's >= start OR <= end
		return currentDOY >= startDOY || currentDOY <= endDOY
	}

	// Normal range within same year
	return currentDOY >= startDOY && currentDOY <= endDOY
}

// GetDefaultAlbum returns the default album ID.
func (s *Scheduler) GetDefaultAlbum() string {
	return s.defaultAlbum
}

// GetScheduleCount returns the number of configured schedules.
func (s *Scheduler) GetScheduleCount() int {
	return len(s.ranges)
}
