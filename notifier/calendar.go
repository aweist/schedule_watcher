package notifier

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/aweist/schedule-watcher/models"
)

// GenerateICS creates an ICS (iCalendar) file content for a game
func GenerateICS(game models.Game) string {
	uid := fmt.Sprintf("%x@schedule-watcher", md5.Sum([]byte(game.ID)))

	startTime, endTime := parseGameTime(game.Date, game.Time)

	dtStart := startTime.UTC().Format("20060102T150405Z")
	dtEnd := endTime.UTC().Format("20060102T150405Z")
	dtStamp := time.Now().UTC().Format("20060102T150405Z")

	leagueName := strings.ToUpper(game.League)
	if leagueName == "" {
		leagueName = "Volleyball"
	}

	var ics strings.Builder
	ics.WriteString("BEGIN:VCALENDAR\r\n")
	ics.WriteString("VERSION:2.0\r\n")
	ics.WriteString(fmt.Sprintf("PRODID:-//%s Schedule Watcher//EN\r\n", leagueName))
	ics.WriteString("CALSCALE:GREGORIAN\r\n")
	ics.WriteString("METHOD:REQUEST\r\n")
	ics.WriteString("BEGIN:VEVENT\r\n")
	ics.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
	ics.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", dtStamp))
	ics.WriteString(fmt.Sprintf("DTSTART:%s\r\n", dtStart))
	ics.WriteString(fmt.Sprintf("DTEND:%s\r\n", dtEnd))

	summary := fmt.Sprintf("%s Volleyball Game", leagueName)
	if game.Division != "" {
		summary += " - " + game.Division
	}
	ics.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", summary))

	description := fmt.Sprintf("League: %s\\nTeam: %s", leagueName, game.TeamCaptain)
	if game.TeamNumber > 0 {
		description += fmt.Sprintf(" (#%d)", game.TeamNumber)
	}
	if game.Division != "" {
		description += fmt.Sprintf("\\nDivision: %s", game.Division)
	}
	description += fmt.Sprintf("\\nTime: %s\\nCourt: %s", game.Time, game.Court)
	if game.Opponent != "" {
		description += fmt.Sprintf("\\nOpponent: %s", game.Opponent)
	}
	ics.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(description)))

	if game.Court != "" {
		location := fmt.Sprintf("Court %s", game.Court)
		ics.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICS(location)))
	}

	ics.WriteString("BEGIN:VALARM\r\n")
	ics.WriteString("TRIGGER:-PT1H\r\n")
	ics.WriteString("ACTION:DISPLAY\r\n")
	ics.WriteString("DESCRIPTION:Volleyball game in 1 hour!\r\n")
	ics.WriteString("END:VALARM\r\n")

	ics.WriteString("END:VEVENT\r\n")
	ics.WriteString("END:VCALENDAR\r\n")

	return ics.String()
}

// parseGameTime converts the game date and time string into start and end times
func parseGameTime(gameDate time.Time, timeStr string) (time.Time, time.Time) {
	duration := time.Hour

	timeStr = strings.TrimSpace(timeStr)
	timeStr = strings.ToUpper(timeStr)
	timeStr = strings.ReplaceAll(timeStr, ".", "")

	// Extract the first time if multiple times are listed (e.g., "8/9pm" -> "8")
	if strings.Contains(timeStr, "/") {
		parts := strings.Split(timeStr, "/")
		if len(parts) > 0 {
			firstTime := parts[0]
			if strings.Contains(strings.ToUpper(timeStr), "PM") && !strings.Contains(firstTime, "PM") {
				timeStr = firstTime + "PM"
			} else if strings.Contains(strings.ToUpper(timeStr), "AM") && !strings.Contains(firstTime, "AM") {
				timeStr = firstTime + "AM"
			} else {
				timeStr = firstTime
			}
		}
	}

	var hour, minute int
	var isPM bool

	if strings.Contains(timeStr, ":") {
		fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	} else {
		fmt.Sscanf(timeStr, "%d", &hour)
	}

	if strings.Contains(timeStr, "PM") {
		isPM = true
	} else if strings.Contains(timeStr, "AM") {
		isPM = false
	} else {
		if hour < 12 && hour != 0 {
			isPM = true
		}
	}

	if isPM && hour != 12 {
		hour += 12
	} else if !isPM && hour == 12 {
		hour = 0
	}

	startTime := time.Date(
		gameDate.Year(), gameDate.Month(), gameDate.Day(),
		hour, minute, 0, 0, gameDate.Location(),
	)

	endTime := startTime.Add(duration)

	return startTime, endTime
}

// escapeICS escapes special characters for ICS format
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	return s
}
