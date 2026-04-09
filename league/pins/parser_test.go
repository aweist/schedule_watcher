package pins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleSchedulePageHTML = `
<TABLE CELLPADDING=0 CELLSPACING=0 ALIGN=CENTER WIDTH=400>
   <TR>
      <TD><B><U>Team Division:</U></B> Coed 4's A-1
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;      <B><U>Current Position:</U></B><FONT SIZE=4><I> 2<sup>nd</sup></I></FONT></TD>
   </TR>
</TABLE>
<BR><TABLE CELLPADDING=3 CELLSPACING=0 WIDTH=700>
   <TR>
      <TH WIDTH=30px>Week</TH>
      <TH>Game Time</TH>
      <TH>Court</TH>
      <TH>Other Team Name</TH>
      <TH>Games Won</TH>
   </TR>
   <TR BGCOLOR=white>
      <TD ALIGN=CENTER>0</TD>
      <TD ALIGN=CENTER>03/17/2026 &nbsp;&nbsp; 9:40</TD>
      <TD ALIGN=CENTER>Court 2</TD>
      <TD ALIGN=LEFT>2 - Pat's Team</TD>
      <TD ALIGN=CENTER>3</TD>
   </TR>
   <TR BGCOLOR=gainsboro>
      <TD ALIGN=CENTER>0</TD>
      <TD ALIGN=CENTER>03/24/2026 &nbsp;&nbsp; 5:10</TD>
      <TD ALIGN=CENTER>Court 4</TD>
      <TD ALIGN=LEFT>6 - Volleybirdies</TD>
      <TD ALIGN=CENTER>2</TD>
   </TR>
   <TR BGCOLOR=white>
      <TD ALIGN=CENTER>0</TD>
      <TD ALIGN=CENTER>04/07/2026 &nbsp;&nbsp; 8:50</TD>
      <TD ALIGN=CENTER>Court 3</TD>
      <TD ALIGN=LEFT>5 - Papa &amp; Family</TD>
      <TD ALIGN=CENTER>1</TD>
   </TR>
   <TR BGCOLOR=lightgreen>
      <TD ALIGN=CENTER>0</TD>
      <TD ALIGN=CENTER>04/14/2026 &nbsp;&nbsp; 7:55</TD>
      <TD ALIGN=CENTER>Court 2</TD>
      <TD ALIGN=LEFT>19 - Goose Bumps</TD>
      <TD ALIGN=CENTER>0</TD>
   </TR>
   <TR BGCOLOR=white>
      <TD ALIGN=CENTER>0</TD>
      <TD ALIGN=CENTER>04/14/2026 &nbsp;&nbsp; 8:50</TD>
      <TD ALIGN=CENTER>Court 1</TD>
      <TD ALIGN=LEFT>36 - Setsy &amp; we know it</TD>
      <TD ALIGN=CENTER>0</TD>
   </TR>
</TABLE>
`

func TestParseSchedule(t *testing.T) {
	games, err := ParseSchedule(sampleSchedulePageHTML, "tue-sets-is-great", "1 - The Sets is Great")
	require.NoError(t, err)
	assert.Len(t, games, 5)

	// First game
	g := games[0]
	assert.Equal(t, "pins", g.League)
	assert.Equal(t, "tue-sets-is-great", g.TeamKey)
	assert.Equal(t, "1 - The Sets is Great", g.TeamCaptain)
	assert.Equal(t, "Coed 4's A-1", g.Division)
	assert.Equal(t, 2026, g.Date.Year())
	assert.Equal(t, 3, int(g.Date.Month()))
	assert.Equal(t, 17, g.Date.Day())
	assert.Equal(t, "9:40", g.Time)
	assert.Equal(t, "2", g.Court)
	assert.Equal(t, "2 - Pat's Team", g.Opponent)
}

func TestParseSchedule_HTMLEntitiesInOpponent(t *testing.T) {
	games, err := ParseSchedule(sampleSchedulePageHTML, "test", "Test Team")
	require.NoError(t, err)

	// Third game has "Papa & Family" (HTML entity)
	assert.Equal(t, "5 - Papa & Family", games[2].Opponent)
}

func TestParseSchedule_MultipleGamesSameDay(t *testing.T) {
	games, err := ParseSchedule(sampleSchedulePageHTML, "test", "Test Team")
	require.NoError(t, err)

	// Games 3 and 4 are both on 04/14/2026
	assert.Equal(t, 14, games[3].Date.Day())
	assert.Equal(t, 14, games[4].Date.Day())
	assert.Equal(t, "7:55", games[3].Time)
	assert.Equal(t, "8:50", games[4].Time)
}

func TestParseSchedule_Division(t *testing.T) {
	games, err := ParseSchedule(sampleSchedulePageHTML, "test", "Test Team")
	require.NoError(t, err)

	for _, g := range games {
		assert.Equal(t, "Coed 4's A-1", g.Division)
	}
}

func TestParseSchedule_GameIDs(t *testing.T) {
	games, err := ParseSchedule(sampleSchedulePageHTML, "test", "Test Team")
	require.NoError(t, err)

	// All IDs should be unique and start with "pins-"
	ids := make(map[string]bool)
	for _, g := range games {
		assert.True(t, len(g.ID) > 0)
		assert.Contains(t, g.ID, "pins-")
		assert.False(t, ids[g.ID], "duplicate game ID: %s", g.ID)
		ids[g.ID] = true
	}
}

func TestParseSchedule_EmptyHTML(t *testing.T) {
	_, err := ParseSchedule("<html></html>", "test", "Test")
	assert.Error(t, err)
}

func TestParseDivision(t *testing.T) {
	div := parseDivision(sampleSchedulePageHTML)
	assert.Equal(t, "Coed 4's A-1", div)
}
