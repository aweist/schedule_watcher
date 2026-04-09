package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCSVParser_ParseSchedule(t *testing.T) {
	// Sample CSV data based on the actual API response
	csvData := `Team Captain ,Team #,Win %,Division ,Wins,Loss,time,8/21/2025,,time,08/28,,Time,09/04,,Time,09/11,,Time,09/18,,Time,09/25,,Time,10/02,,Time,10/09,,Time,10/16,,Time,10/23
Jeff,1,66.67%,Comp Div 1 AG,4,2,7:00 PM,ct 7,,7:00 PM,ct 7,,7:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,
Cory G,2,100.00%,Comp Div 1 AG,3,0,8:00 PM,ct 7,,7:00 PM,ct 7,,8:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,
Erica G,3,66.67%,Comp Div 1 AG,4,2,9:00 PM,ct 7,,8:00 PM,ct 7,,9:00 PM,ct 6,,,,,,,,,,,,,,,,,,,,,
Rachel Wise,6,100.00%,Fun 4s AG,6,0,6:00 PM,ct 4,,8:00 PM,ct 5,,9:00 PM,CT 4,,,,,,,,,,,,,,,,,,,,,
David Morgan,16,100.00%,Fun 4s AG,6,0,7:00 PM,ct 3,,8:00 PM,ct 3,,9:00 PM,CT 3,,,,,,,,,,,,,,,,,,,,,
Fall Schedule,,#DIV/0!,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,
Alec Van Wormer,1,100.00%,Fun 6s AG,6,0,6:00 PM,ct 9,,9:00 PM,ct 9,,7:00 PM,ct 9,,,,,,,,,,,,,,,,,,,,,
Daghera Hewlett @,17,0.00%,Comp Div 1 AG,0,6,7:00 PM,ct 6,,6:00 PM,ct 6,,8/9pm,ct 7/7,,,,,,,,,,,,,,,,,,,,,`

	tests := []struct {
		name     string
		teamName string
		expected int // expected number of games found
		wantErr  bool
	}{
		{
			name:     "Parse games for Jeff",
			teamName: "Jeff",
			expected: 3, // 3 scheduled games
			wantErr:  false,
		},
		{
			name:     "Parse games for Rachel Wise",
			teamName: "Rachel Wise",
			expected: 3,
			wantErr:  false,
		},
		{
			name:     "Parse games for David",
			teamName: "David",
			expected: 3, // Should match "David Morgan"
			wantErr:  false,
		},
		{
			name:     "No games for non-existent team",
			teamName: "NonExistent",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Case insensitive matching",
			teamName: "jeff",
			expected: 3,
			wantErr:  false,
		},
		{
			name:     "Parse games for Daghera Hewlett",
			teamName: "Daghera Hewlett @",
			expected: 4,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewCSVParser(tt.teamName)
			games, err := parser.ParseSchedule(csvData)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(games) != tt.expected {
				t.Errorf("ParseSchedule() got %d games, want %d", len(games), tt.expected)
				return
			}

			// Verify game structure for first game if any exist
			if len(games) > 0 {
				game := games[0]
				if game.ID == "" {
					t.Error("Game ID should not be empty")
				}
				if game.TeamCaptain == "" {
					t.Error("Team captain should not be empty")
				}
				if game.Time == "" {
					t.Error("Game time should not be empty")
				}
				if game.Court == "" {
					t.Error("Court should not be empty")
				}
			}
		})
	}
}

// CSV format appears to have changed as of Spring 2026 season, so added test with new format to ensure parsing still works correctly
func TestCSVParser_ParseScheduleSpring2026(t *testing.T) {
	// Sample CSV data based on the actual API response
	csvData := " ,Team #,Division,Win %,Wins,Loss,time,4/2/2026,,time,04/09,,Time,04/16,,Time,04/23,,Time,04/30,,Time,05/07,,Time,05/14,,Time,05/21,,Time,05/28,,Time,06/04\r\nCory G,1,Comp 4s AG,100.00%,3,0,7:00 PM,ct 5,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nErika G,2,Comp 4s AG,100.00%,3,0,8:00 PM,ct 5,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nTaylor Sisneros,3,Comp 4s AG,66.67%,2,1,8:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nAlex Lugo #1,4,Comp 4s AG,33.33%,1,2,6:00 PM,ct 5,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nAlex Lugo # 2,5,Comp 4s AG,0.00%,0,3,7:00 PM,ct 5,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nJeff Hoover,6,Comp 4s AG,66.67%,2,1,7:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nBrandon Hall,7,Comp 4s AG,33.33%,1,2,7:00 PM,ct 6,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nGabby Cuomo,8,Comp 4s AG,0.00%,0,3,8:00 PM,ct 5,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nAshleigh/Patrick,9,Comp 4s AG,66.67%,2,1,6:00 PM,ct 5,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nKris Adkins,10,Comp 4s AG,100.00%,3,0,8:00 PM,ct 6,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nTyler Adkins,11,Comp 4s AG,100.00%,3,0,6:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nKalin Hannon,12,Comp 4s AG,33.33%,1,2,8:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nDerek Valdez,13,Comp 4s AG,0.00%,0,3,8:00 PM,ct 6,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nTravis Hannon,14,Comp 4s AG,33.33%,1,2,7:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nKyle Smith,15,Comp 4s AG,0.00%,0,3,6:00 PM,ct 7,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nNick Hancock,16,Comp 4s AG,66.67%,2,1,7:00 PM,ct 6,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nTrent Thompson,17,Comp 4s AG,#DIV/0!,,,9:00 PM,ct 6,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nJesus Vasquez,1,Fun 4s  AG,0.00%,0,3,6:00 PM,ct 4,,8:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nJesse Mittino,2,Fun 4s  AG,100.00%,3,0,6:00 PM,ct 4,,9:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nJeff Hoover,3,Fun 4s  AG,100.00%,3,0,8:00 PM,ct 4,,8:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nCicada Zehner  #1,4,Fun 4s  AG,33.33%,1,2,7:00 PM,ct 4,,6:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nCicada Zehner   #2,5,Fun 4s  AG,0.00%,0,3,8:00 PM,ct 4,,7:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nDaniel Castaneda  #1,6,Fun 4s  AG,100.00%,3,0,6:00 PM,ct 1,,6:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nDaniel Castaneda  #2,7,Fun 4s  AG,66.67%,2,1,7:00 PM,ct 1,,7:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nDaniel Castaneda  #3,8,Fun 4s  AG,33.33%,1,2,8:00 PM,ct 1,,8:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nDave Morgan,9,Fun 4s  AG,66.67%,2,1,8:00 PM,ct 1,,9:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nZak Hannon,10,Fun 4s  AG,33.33%,1,2,7:00 PM,ct 13,,7:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nMichael Dugan,11,Fun 4s  AG,100.00%,3,0,9:00 PM,ct 3,,8:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nKevin Knuth,12,Fun 4s  AG,33.33%,1,2,7:00 PM,ct 1,,9:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nAlexa Cordray,13,Fun 4s  AG,0.00%,0,3,9:00 PM,ct 4,,9:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nPeri Duncan,14,Fun 4s  AG,100.00%,3,0,9:00 PM,ct 4,,9:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nDaniel Mendoza,15,Fun 4s  AG,66.67%,2,1,7:00 PM,ct 3,,9:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nMatt Boggs,16,Fun 4s  AG,0.00%,0,3,6:00 PM,ct 3,,7:00 PM,ct 2,,,,,,,,,,,,,,,,,,,,,,,,\r\nTim W,17,Fun 4s  AG,0.00%,0,3,9:00 PM,ct 3,,6:00 PM,ct 3,,,,,,,,,,,,,,,,,,,,,,,,\r\nVin W #1,18,Fun 4s  AG,33.33%,1,2,7:00 PM,ct 3,,6:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nVin W  #2,19,Fun 4s  AG,0.00%,0,3,8:00 PM,ct 3,,7:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nMark K,20,Fun 4s  AG,100.00%,3,0,6:00 PM,ct 3,,6:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nBlake A,21,Fun 4s  AG,0.00%,0,3,6:00 PM,ct 1,,7:00 PM,ct 2,,,,,,,,,,,,,,,,,,,,,,,,\r\nWally W,22,Fun 4s  AG,100.00%,3,0,8:00 PM,ct 3,,8:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nHappy McThicc,23,Fun 4s  AG,66.67%,2,1,7:00 PM,ct 13,,6:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\nErik A,24,Fun 4s  AG,66.67%,2,1,7:00 PM,ct 4,,7:00 PM,ct 4,,,,,,,,,,,,,,,,,,,,,,,,\r\ntalia  grier,25,Fun 4s  AG,0.00%,0,3,8:00 PM,ct 2,,7:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nTrey Sanders,26,Fun 4s  AG,100.00%,3,0,8:00 PM,ct 2,,8:00 PM,ct 1,,,,,,,,,,,,,,,,,,,,,,,,\r\nJesse Mittino #2,27,Fun 4s  AG,#DIV/0!,,,,,,8:00 PM,ct 2,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\n,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,\r\nGabriel H.,1,Fun 6s,66.67%,2,1,6:00 PM,ct 13,,8:00 PM,ct 13,,,,,,,,,,,,,,,,,,,,,,,,\r\nHunter E.,2,Fun 6s,33.33%,1,2,6:00 PM,ct 13,,7:00 PM,ct 13,,,,,,,,,,,,,,,,,,,,,,,,\r\nEmma R.,3,Fun 6s,100.00%,3,0,7:00 PM,ct 13,,6:00 PM,ct 13,,,,,,,,,,,,,,,,,,,,,,,,\r\nMeghan O.,4,Fun 6s,0.00%,0,3,7:00 PM,ct 13,,7:00 PM,ct 13,,,,,,,,,,,,,,,,,,,,,,,,\r\nGreg T,5,Fun 6s,0.00%,0,3,8:00 PM,ct 13,,8:00 PM,ct 13,,,,,,,,,,,,,,,,,,,,,,,,\r\nBelle Michaud,6,Fun 6s,100.00%,3,0,8:00 PM,ct 13,,6:00 PM,ct 13,,,,,,,,,,,,,,,,,,,,,,,,"

	tests := []struct {
		name     string
		teamName string
		expected int // expected number of games found
		wantErr  bool
	}{
		{
			name:     "Parse games for Taylor Sisneros",
			teamName: "Taylor Sisneros",
			expected: 1, // 1 scheduled games
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewCSVParser(tt.teamName)
			games, err := parser.ParseSchedule(csvData)

			if tt.wantErr {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}

			assert.Equal(t, tt.expected, len(games), "Expected %d games, got %d", tt.expected, len(games))

			// Verify game structure for first game if any exist
			if len(games) > 0 {
				game := games[0]
				if game.ID == "" {
					t.Error("Game ID should not be empty")
				}
				if game.TeamCaptain == "" {
					t.Error("Team captain should not be empty")
				}
				if game.Time == "" {
					t.Error("Game time should not be empty")
				}
				if game.Court == "" {
					t.Error("Court should not be empty")
				}
			}
		})
	}
}

func TestCSVParser_ParseScheduleWithSpecificData(t *testing.T) {
	csvData := `Team Captain ,Team #,Win %,Division ,Wins,Loss,time,8/21/2025,,time,08/28,,Time,09/04
Jeff,1,66.67%,Comp Div 1 AG,4,2,7:00 PM,ct 7,,7:00 PM,ct 7,,7:00 PM,ct 7`

	parser := NewCSVParser("Jeff")
	games, err := parser.ParseSchedule(csvData)

	if err != nil {
		t.Fatalf("ParseSchedule() error = %v", err)
	}

	if len(games) != 3 {
		t.Fatalf("Expected 3 games, got %d", len(games))
	}

	// Test first game
	game := games[0]
	if game.TeamCaptain != "Jeff" {
		t.Errorf("Expected team captain 'Jeff', got '%s'", game.TeamCaptain)
	}
	if game.TeamNumber != 1 {
		t.Errorf("Expected team number 1, got %d", game.TeamNumber)
	}
	if game.Division != "Comp Div 1 AG" {
		t.Errorf("Expected division 'Comp Div 1 AG', got '%s'", game.Division)
	}
	if game.Time != "7:00 pm" {
		t.Errorf("Expected time '7:00 pm', got '%s'", game.Time)
	}
	if game.Court != "7" {
		t.Errorf("Expected court '7', got '%s'", game.Court)
	}

	// Verify date parsing
	expectedDate := time.Date(2025, 8, 21, 0, 0, 0, 0, time.Local)
	if !game.Date.Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, game.Date)
	}
}

func TestCSVParser_ParseDate(t *testing.T) {
	parser := NewCSVParser("test")
	currentYear := time.Now().Year()

	tests := []struct {
		input    string
		expected time.Time
	}{
		{
			input:    "8/21/2025",
			expected: time.Date(2025, 8, 21, 0, 0, 0, 0, time.Local),
		},
		{
			input:    "12/31/24",
			expected: time.Date(2024, 12, 31, 0, 0, 0, 0, time.Local),
		},
		{
			input:    "1/15", // Should use current year
			expected: time.Date(currentYear, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			input:    "time,8/21/2025", // Should handle prefix
			expected: time.Date(2025, 8, 21, 0, 0, 0, 0, time.Local),
		},
		{
			input:    "Time,08/28", // Should handle prefix and use current year
			expected: time.Date(currentYear, 8, 28, 0, 0, 0, 0, time.Local),
		},
		{
			input:    "invalid",
			expected: time.Time{}, // Zero time for invalid input
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.parseDate(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("parseDate(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCSVParser_GenerateGameID(t *testing.T) {
	parser := NewCSVParser("test")
	date := time.Date(2025, 8, 21, 0, 0, 0, 0, time.Local)

	id1 := parser.generateGameID("Jeff", date, "7:00 PM", "ct 7")
	id2 := parser.generateGameID("Jeff", date, "7:00 PM", "ct 7")
	id3 := parser.generateGameID("Cory", date, "7:00 PM", "ct 7")

	// Same inputs should generate same ID
	if id1 != id2 {
		t.Error("Same inputs should generate same game ID")
	}

	// Different inputs should generate different IDs
	if id1 == id3 {
		t.Error("Different inputs should generate different game IDs")
	}

	// ID should be 12 characters (truncated MD5)
	if len(id1) != 12 {
		t.Errorf("Game ID should be 12 characters, got %d", len(id1))
	}
}

func TestCSVParser_IsTeamOfInterest(t *testing.T) {
	tests := []struct {
		teamName    string
		captain     string
		shouldMatch bool
	}{
		{
			teamName:    "Jeff",
			captain:     "Jeff",
			shouldMatch: true,
		},
		{
			teamName:    "jeff",
			captain:     "Jeff",
			shouldMatch: true,
		},
		{
			teamName:    "Jeff",
			captain:     "jeff",
			shouldMatch: true,
		},
		{
			teamName:    "Rachel",
			captain:     "Rachel Wise",
			shouldMatch: true,
		},
		{
			teamName:    "Wise",
			captain:     "Rachel Wise",
			shouldMatch: true,
		},
		{
			teamName:    "Jeff",
			captain:     "Cory G",
			shouldMatch: false,
		},
		{
			teamName:    "David",
			captain:     "David Morgan",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.teamName+"_"+tt.captain, func(t *testing.T) {
			parser := NewCSVParser(tt.teamName)
			result := parser.isTeamOfInterest(tt.captain)
			if result != tt.shouldMatch {
				t.Errorf("isTeamOfInterest(%s, %s) = %v, want %v",
					tt.teamName, tt.captain, result, tt.shouldMatch)
			}
		})
	}
}

func TestCSVParser_EmptyOrInvalidCSV(t *testing.T) {
	parser := NewCSVParser("Jeff")

	tests := []struct {
		name    string
		csvData string
		wantErr bool
	}{
		{
			name:    "Empty CSV",
			csvData: "",
			wantErr: true,
		},
		{
			name:    "Only headers",
			csvData: "Team Captain ,Team #,Win %,Division",
			wantErr: true,
		},
		{
			name:    "Invalid CSV format",
			csvData: "Invalid\nCSV\"Data",
			wantErr: true, // CSV reader will fail on malformed quotes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			games, err := parser.ParseSchedule(tt.csvData)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && games == nil {
				t.Error("ParseSchedule() should return empty slice, not nil")
			}
		})
	}
}

func TestCSVParser_FindDateColumns(t *testing.T) {
	parser := NewCSVParser("test")

	headers := []string{
		"Team Captain", "Team #", "Win %", "Division", "Wins", "Loss",
		"time", "8/21/2025", "", "time", "08/28", "", "Time", "09/04", "",
	}

	dateColumns := parser.findDateColumns(headers)

	// Should find date columns
	expectedColumns := map[int]int{
		0: 7,  // First date column at index 7 (8/21/2025)
		1: 10, // Second date column at index 10 (08/28)
		2: 13, // Third date column at index 13 (09/04)
	}

	if len(dateColumns) != len(expectedColumns) {
		t.Errorf("Expected %d date columns, got %d", len(expectedColumns), len(dateColumns))
	}

	for dateIdx, colIdx := range expectedColumns {
		if dateColumns[dateIdx] != colIdx {
			t.Errorf("Expected date index %d at column %d, got %d",
				dateIdx, colIdx, dateColumns[dateIdx])
		}
	}
}

func TestCSVParser_SkipFallScheduleRows(t *testing.T) {
	csvData := `Team Captain ,Team #,Win %,Division ,Wins,Loss,time,8/21/2025
Jeff,1,66.67%,Comp Div 1 AG,4,2,7:00 PM,ct 7
Fall Schedule,,#DIV/0!,,,,,
Rachel Wise,6,100.00%,Fun 4s AG,6,0,6:00 PM,ct 4`

	parser := NewCSVParser("Rachel")
	games, err := parser.ParseSchedule(csvData)

	if err != nil {
		t.Fatalf("ParseSchedule() error = %v", err)
	}

	// Should find Rachel's game but skip Fall Schedule row
	if len(games) != 1 {
		t.Errorf("Expected 1 game, got %d", len(games))
	}

	if len(games) > 0 && games[0].TeamCaptain != "Rachel Wise" {
		t.Errorf("Expected Rachel Wise, got %s", games[0].TeamCaptain)
	}
}
func TestGameTimeStrToGameTimes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single time with PM",
			input:    "7:00 PM",
			expected: []string{"7:00 pm"},
		},
		{
			name:     "Multiple times with PM",
			input:    "8:00 PM/9:00 PM",
			expected: []string{"8:00 pm", "9:00 pm"},
		},
		{
			name:     "Multiple times without PM",
			input:    "8/9",
			expected: []string{"8:00 pm", "9:00 pm"},
		},
		{
			name:     "Mixed case PM",
			input:    "7:00 pm/8:00 PM",
			expected: []string{"7:00 pm", "8:00 pm"},
		},
		{
			name:     "Times with spaces",
			input:    "7 / 8 / 9",
			expected: []string{"7:00 pm", "8:00 pm", "9:00 pm"},
		},
		{
			name:     "Time with extra formatting",
			input:    "8/9pm",
			expected: []string{"8:00 pm", "9:00 pm"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "String with only separators",
			input:    "///",
			expected: []string{},
		},
		{
			name:     "Single time without formatting",
			input:    "7",
			expected: []string{"7:00 pm"},
		},
		{
			name:     "Time with only :00 to remove",
			input:    "7:00/8:00",
			expected: []string{"7:00 pm", "8:00 pm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gameTimeStrToGameTimes(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("gameTimeStrToGameTimes(%q) returned %d items, want %d",
					tt.input, len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("gameTimeStrToGameTimes(%q)[%d] = %q, want %q",
						tt.input, i, result[i], expected)
				}
			}
		})
	}
}
func TestCourtStrToCourts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single court with ct prefix",
			input:    "ct 7",
			expected: []string{"7"},
		},
		{
			name:     "Single court with court prefix",
			input:    "court 7",
			expected: []string{"7"},
		},
		{
			name:     "Multiple courts with ct prefix",
			input:    "ct 7/7",
			expected: []string{"7", "7"},
		},
		{
			name:     "Multiple courts with different numbers",
			input:    "ct 7/8",
			expected: []string{"7", "8"},
		},
		{
			name:     "Mixed case CT",
			input:    "CT 4",
			expected: []string{"4"},
		},
		{
			name:     "Mixed case Court",
			input:    "Court 3",
			expected: []string{"3"},
		},
		{
			name:     "Multiple courts with CT prefix",
			input:    "CT 4/5",
			expected: []string{"4", "5"},
		},
		{
			name:     "Courts with spaces",
			input:    "ct 7 / 8 / 9",
			expected: []string{"7", "8", "9"},
		},
		{
			name:     "Courts without prefix",
			input:    "7/8",
			expected: []string{"7", "8"},
		},
		{
			name:     "Single court without prefix",
			input:    "7",
			expected: []string{"7"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "String with only separators",
			input:    "///",
			expected: []string{},
		},
		{
			name:     "String with only prefixes",
			input:    "ct/court",
			expected: []string{},
		},
		{
			name:     "Courts with extra whitespace",
			input:    "  ct 7  /  ct 8  ",
			expected: []string{"7", "8"},
		},
		{
			name:     "Mixed prefixes",
			input:    "ct 7/court 8",
			expected: []string{"7", "8"},
		},
		{
			name:     "Court numbers with letters",
			input:    "ct 7A/ct 8B",
			expected: []string{"7a", "8b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := courtStrToCourts(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("courtStrToCourts(%q) returned %d items, want %d",
					tt.input, len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("courtStrToCourts(%q)[%d] = %q, want %q",
						tt.input, i, result[i], expected)
				}
			}
		})
	}
}
