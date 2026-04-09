package pins

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// scheduleOption represents a parsed option from the schedule dropdown.
type scheduleOption struct {
	Value string // SCHEDULE_ID value
	Text  string // Display text, e.g., "Tue Night Mar-May 2026 Season"
}

// teamOption represents a parsed option from the team dropdown.
type teamOption struct {
	Value string // TEAM_ID value
	Text  string // Display text, e.g., "1 - The Sets is Great"
}

var (
	scheduleOptionRe = regexp.MustCompile(`<OPTION\s+VALUE="(\d+)"[^>]*>\s*(.+?)\s*</OPTION>`)
	teamOptionRe     = regexp.MustCompile(`<OPTION\s+VALUE="(\d+)"[^>]*>\s*(.+?)\s*</OPTION>`)
	// Matches patterns like "Tue Night Mar-May 2026 Season"
	seasonPatternRe = regexp.MustCompile(`(?i)^(\w+)\s+Night\s+(\w+)-(\w+)\s+(\d{4})\s+Season$`)
)

// monthIndex maps month abbreviations to month numbers.
var monthIndex = map[string]time.Month{
	"jan": time.January, "feb": time.February, "mar": time.March,
	"apr": time.April, "may": time.May, "jun": time.June,
	"jul": time.July, "aug": time.August, "sep": time.September,
	"oct": time.October, "nov": time.November, "dec": time.December,
}

// DiscoverCurrentScheduleID finds the current season's SCHEDULE_ID for a given day of week.
// It parses the schedule dropdown HTML and finds the best match.
func DiscoverCurrentScheduleID(html string, dayOfWeek string) (string, error) {
	// Find the first SELECT (schedule selector)
	selectStart := strings.Index(html, `<SELECT NAME=SCHEDULE_ID`)
	if selectStart == -1 {
		return "", fmt.Errorf("schedule select not found in HTML")
	}
	selectEnd := strings.Index(html[selectStart:], `</SELECT>`)
	if selectEnd == -1 {
		return "", fmt.Errorf("schedule select end not found")
	}
	selectHTML := html[selectStart : selectStart+selectEnd]

	options := parseScheduleOptions(selectHTML)
	return findBestSchedule(options, dayOfWeek, time.Now())
}

func parseScheduleOptions(html string) []scheduleOption {
	matches := scheduleOptionRe.FindAllStringSubmatch(html, -1)
	var options []scheduleOption
	for _, m := range matches {
		if m[1] == "0" {
			continue // skip "Select A Schedule" placeholder
		}
		options = append(options, scheduleOption{
			Value: m[1],
			Text:  strings.TrimSpace(m[2]),
		})
	}
	return options
}

// findBestSchedule finds the schedule that matches the day of week and
// whose date range contains or is closest to the current date.
func findBestSchedule(options []scheduleOption, dayOfWeek string, now time.Time) (string, error) {
	dayLower := strings.ToLower(dayOfWeek)

	type scored struct {
		id    string
		score int // higher is better: 2 = contains now, 1 = future, 0 = past
		year  int
		start time.Month
	}

	var candidates []scored

	for _, opt := range options {
		m := seasonPatternRe.FindStringSubmatch(opt.Text)
		if m == nil {
			continue
		}

		optDay := strings.ToLower(m[1])
		if !strings.HasPrefix(optDay, dayLower[:3]) {
			continue
		}

		startMonth, ok1 := monthIndex[strings.ToLower(m[2])]
		endMonth, ok2 := monthIndex[strings.ToLower(m[3])]
		year, err := strconv.Atoi(m[4])
		if !ok1 || !ok2 || err != nil {
			continue
		}

		// Build approximate date range for this season
		seasonStart := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.Local)
		// End month: use last day
		seasonEnd := time.Date(year, endMonth+1, 0, 23, 59, 59, 0, time.Local)

		// Handle wrap-around seasons (e.g., "Dec-Feb")
		if endMonth < startMonth {
			seasonEnd = time.Date(year+1, endMonth+1, 0, 23, 59, 59, 0, time.Local)
		}

		score := 0
		if !now.Before(seasonStart) && !now.After(seasonEnd) {
			score = 2 // current season
		} else if now.Before(seasonStart) {
			score = 1 // future season
		}

		candidates = append(candidates, scored{
			id:    opt.Value,
			score: score,
			year:  year,
			start: startMonth,
		})
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no schedule found matching day %q", dayOfWeek)
	}

	// Pick best: highest score, then most recent year, then latest start month
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		} else if c.score == best.score {
			if c.year > best.year {
				best = c
			} else if c.year == best.year && c.start > best.start {
				best = c
			}
		}
	}

	return best.id, nil
}

// DiscoverTeamID finds the TEAM_ID for a team by matching team name in the HTML.
// Uses case-insensitive substring matching.
func DiscoverTeamID(html string, teamName string) (string, string, error) {
	// Find the TEAM_ID SELECT
	selectStart := strings.Index(html, `<SELECT NAME=TEAM_ID`)
	if selectStart == -1 {
		return "", "", fmt.Errorf("team select not found in HTML")
	}
	selectEnd := strings.Index(html[selectStart:], `</SELECT>`)
	if selectEnd == -1 {
		return "", "", fmt.Errorf("team select end not found")
	}
	selectHTML := html[selectStart : selectStart+selectEnd]

	matches := teamOptionRe.FindAllStringSubmatch(selectHTML, -1)
	teamNameLower := strings.ToLower(teamName)

	for _, m := range matches {
		if m[1] == "0" {
			continue
		}
		optText := strings.TrimSpace(m[2])
		// HTML entities decode
		optText = strings.ReplaceAll(optText, "&amp;", "&")
		if strings.Contains(strings.ToLower(optText), teamNameLower) {
			return m[1], optText, nil
		}
	}

	return "", "", fmt.Errorf("team %q not found in schedule", teamName)
}
