package parser

import (
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aweist/schedule-watcher/models"
)

type CSVParser struct {
	teamName string
	year     int
}

func NewCSVParser(teamName string) *CSVParser {
	return &CSVParser{
		teamName: teamName,
		year:     time.Now().Year(),
	}
}

func (p *CSVParser) ParseSchedule(csvData string) ([]models.Game, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("insufficient data in CSV")
	}

	var games []models.Game
	headers := records[0]

	dateColumns := p.findDateColumns(headers)

	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 7 {
			continue
		}

		teamCaptain := strings.TrimSpace(row[0])
		if teamCaptain == "" || strings.Contains(teamCaptain, "Fall Schedule") {
			continue
		}

		if !p.isTeamOfInterest(teamCaptain) {
			continue
		}

		teamNum, _ := strconv.Atoi(row[1])
		division := strings.TrimSpace(row[3])

		for _, colIdx := range dateColumns {
			if colIdx > 0 && colIdx < len(row) {
				// We need to account for multiple games per night.
				// Normally this is in the format of 8/9pm,ct 7/7
				gameTimes := gameTimeStrToGameTimes(row[colIdx-1]) // Time is in column before date
				courts := courtStrToCourts(row[colIdx])            // Court is in the date column

				for i, gameTime := range gameTimes {
					court := courts[0]
					if i < len(courts) {
						court = courts[i]
					}
					game := p.createGame(teamCaptain, teamNum, division, gameTime, court, headers[colIdx])
					games = append(games, game)
				}
			}
		}
	}

	return games, nil
}

func courtStrToCourts(s string) []string {
	courts := []string{}
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ct", "")
	s = strings.ReplaceAll(s, "court", "")

	parts := strings.Split(s, "/")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		courts = append(courts, part)
	}

	return courts
}

func gameTimeStrToGameTimes(s string) []string {
	gameTimes := []string{}
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "pm", "")
	s = strings.ReplaceAll(s, ":00", "")
	parts := strings.Split(s, "/")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part += ":00 pm"
		gameTimes = append(gameTimes, part)
	}
	return gameTimes
}

func (p *CSVParser) findDateColumns(headers []string) map[int]int {
	dateColumns := make(map[int]int)
	dateIndex := 0

	for i, header := range headers {
		header = strings.TrimSpace(header)
		if strings.Contains(header, "/") {
			// Found a date column
			dateColumns[dateIndex] = i
			dateIndex++
		}
	}

	return dateColumns
}

func (p *CSVParser) isTeamOfInterest(captain string) bool {
	captainLower := strings.ToLower(captain)
	teamLower := strings.ToLower(p.teamName)
	return strings.Contains(captainLower, teamLower)
}

func (p *CSVParser) createGame(captain string, teamNum int, division string, gameTime, court, dateStr string) models.Game {
	gameDate := p.parseDate(dateStr)

	gameID := p.generateGameID(captain, gameDate, gameTime, court)

	return models.Game{
		ID:          gameID,
		TeamCaptain: captain,
		TeamNumber:  teamNum,
		Division:    division,
		Date:        gameDate,
		Time:        gameTime,
		Court:       court,
		Raw:         fmt.Sprintf("%s|%s|%s", dateStr, gameTime, court),
	}
}

func (p *CSVParser) parseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	dateStr = strings.ReplaceAll(dateStr, "time,", "")
	dateStr = strings.ReplaceAll(dateStr, "Time,", "")
	dateStr = strings.TrimSpace(dateStr)

	parts := strings.Split(dateStr, "/")
	if len(parts) >= 2 {
		month, _ := strconv.Atoi(parts[0])
		day, _ := strconv.Atoi(parts[1])

		year := p.year
		if len(parts) == 3 {
			yearPart, _ := strconv.Atoi(parts[2])
			if yearPart > 0 {
				if yearPart < 100 {
					year = 2000 + yearPart
				} else {
					year = yearPart
				}
			}
		}

		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	}

	return time.Time{}
}

func (p *CSVParser) generateGameID(captain string, date time.Time, gameTime, court string) string {
	data := fmt.Sprintf("%s-%s-%s-%s", captain, date.Format("2006-01-02"), gameTime, court)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)[:12]
}
