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
	// Generate a unique UID for the event
	uid := fmt.Sprintf("%x@schedule-watcher", md5.Sum([]byte(game.ID)))
	
	// Parse the game time to determine start and end times
	startTime, endTime := parseGameTime(game.Date, game.Time)
	
	// Format times in ICS format (YYYYMMDDTHHMMSSZ)
	dtStart := startTime.UTC().Format("20060102T150405Z")
	dtEnd := endTime.UTC().Format("20060102T150405Z")
	dtStamp := time.Now().UTC().Format("20060102T150405Z")
	
	// Build the ICS content
	var ics strings.Builder
	ics.WriteString("BEGIN:VCALENDAR\r\n")
	ics.WriteString("VERSION:2.0\r\n")
	ics.WriteString("PRODID:-//IVP Schedule Watcher//EN\r\n")
	ics.WriteString("CALSCALE:GREGORIAN\r\n")
	ics.WriteString("METHOD:REQUEST\r\n")
	ics.WriteString("BEGIN:VEVENT\r\n")
	ics.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
	ics.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", dtStamp))
	ics.WriteString(fmt.Sprintf("DTSTART:%s\r\n", dtStart))
	ics.WriteString(fmt.Sprintf("DTEND:%s\r\n", dtEnd))
	ics.WriteString(fmt.Sprintf("SUMMARY:Volleyball Game - %s\r\n", game.Division))
	
	// Create a detailed description
	description := fmt.Sprintf("Team: %s (#%d)\\nDivision: %s\\nTime: %s\\nCourt: %s", 
		game.TeamCaptain, game.TeamNumber, game.Division, game.Time, game.Court)
	ics.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(description)))
	
	// Add location if court information is available
	if game.Court != "" {
		location := fmt.Sprintf("Court %s", game.Court)
		ics.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICS(location)))
	}
	
	// Add a reminder 1 hour before the game
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
	// Default to 1 hour duration
	duration := time.Hour
	
	// Clean up the time string
	timeStr = strings.TrimSpace(timeStr)
	timeStr = strings.ToUpper(timeStr)
	timeStr = strings.ReplaceAll(timeStr, ".", "")
	
	// Extract the first time if multiple times are listed (e.g., "8/9pm" -> "8")
	if strings.Contains(timeStr, "/") {
		parts := strings.Split(timeStr, "/")
		if len(parts) > 0 {
			// Take the first time and add PM if the original had it
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
	
	// Parse the time
	var hour, minute int
	var isPM bool
	
	// Try to parse different time formats
	if strings.Contains(timeStr, ":") {
		// Format like "7:00 PM" or "7:30PM"
		fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	} else {
		// Format like "8PM" or "8"
		fmt.Sscanf(timeStr, "%d", &hour)
	}
	
	// Check for PM/AM
	if strings.Contains(timeStr, "PM") {
		isPM = true
	} else if strings.Contains(timeStr, "AM") {
		isPM = false
	} else {
		// Assume PM for evening games (typical volleyball times)
		if hour < 12 && hour != 0 {
			isPM = true
		}
	}
	
	// Adjust hour for PM
	if isPM && hour != 12 {
		hour += 12
	} else if !isPM && hour == 12 {
		hour = 0
	}
	
	// Create the start time
	startTime := time.Date(
		gameDate.Year(), gameDate.Month(), gameDate.Day(),
		hour, minute, 0, 0, gameDate.Location(),
	)
	
	// End time is start time plus duration
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