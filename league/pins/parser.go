package pins

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aweist/schedule-watcher/models"
)

var (
	// Match table rows: <TR ...> content </TR>
	trRe = regexp.MustCompile(`(?is)<TR[^>]*>(.*?)</TR>`)
	// Match table cells: <TD ...> content </TD> or <TH ...> content </TH>
	tdRe = regexp.MustCompile(`(?is)<T[DH][^>]*>(.*?)</T[DH]>`)
	// Match division text: "Team Division:" followed by text
	divisionRe = regexp.MustCompile(`(?i)<B><U>Team Division:</U></B>\s*(.+?)(?:\s*&nbsp;|<)`)
	// Match game time: "MM/DD/YYYY    H:MM" or "MM/DD/YYYY    HH:MM"
	gameTimeRe = regexp.MustCompile(`(\d{2}/\d{2}/\d{4})\s+(\d{1,2}:\d{2})`)
)

// ParseSchedule parses the HTML from a PINS team schedule page into games.
func ParseSchedule(html string, teamKey string, teamName string) ([]models.Game, error) {
	division := parseDivision(html)

	// Find the schedule table - it's the one with "Week" and "Game Time" headers
	rows := trRe.FindAllStringSubmatch(html, -1)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no table rows found in HTML")
	}

	var games []models.Game
	inScheduleTable := false

	for _, row := range rows {
		cells := tdRe.FindAllStringSubmatch(row[1], -1)
		if len(cells) == 0 {
			continue
		}

		// Check if this is the header row of the schedule table
		firstCell := stripHTML(cells[0][1])
		if strings.TrimSpace(firstCell) == "Week" {
			inScheduleTable = true
			continue
		}

		if !inScheduleTable {
			continue
		}

		// Expect 5 columns: Week, Game Time, Court, Other Team Name, Games Won
		if len(cells) < 4 {
			continue
		}

		gameTimeStr := stripHTML(cells[1][1])
		courtStr := stripHTML(cells[2][1])
		opponent := stripHTML(cells[3][1])

		// Parse game time: "03/17/2026    9:40"
		m := gameTimeRe.FindStringSubmatch(gameTimeStr)
		if m == nil {
			continue
		}

		dateStr := m[1]
		timeStr := m[2]

		gameDate, err := time.Parse("01/02/2006", dateStr)
		if err != nil {
			continue
		}

		court := strings.TrimSpace(courtStr)
		court = strings.TrimPrefix(court, "Court ")
		court = strings.TrimPrefix(court, "court ")

		opponent = strings.TrimSpace(opponent)
		// Decode HTML entities
		opponent = strings.ReplaceAll(opponent, "&amp;", "&")

		gameID := generatePINSGameID(teamKey, dateStr, timeStr, court)

		games = append(games, models.Game{
			ID:          gameID,
			League:      "pins",
			TeamKey:     teamKey,
			TeamCaptain: teamName,
			Division:    division,
			Date:        gameDate,
			Time:        timeStr,
			Court:       court,
			Opponent:    opponent,
			Raw:         fmt.Sprintf("%s|%s|%s|%s", dateStr, timeStr, court, opponent),
		})
	}

	return games, nil
}

func parseDivision(html string) string {
	m := divisionRe.FindStringSubmatch(html)
	if m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func stripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	// Collapse &nbsp; to spaces
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	// Collapse multiple spaces
	spaceRe := regexp.MustCompile(`\s+`)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func generatePINSGameID(teamKey, dateStr, timeStr, court string) string {
	data := fmt.Sprintf("pins-%s-%s-%s-%s", teamKey, dateStr, timeStr, court)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("pins-%x", hash)[:16]
}
