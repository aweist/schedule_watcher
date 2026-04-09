package pins

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleScheduleHTML = `
<SELECT NAME=SCHEDULE_ID onChange="location.href='/schedules.cgi?SCHEDULE_ID=' + this.value">
   <OPTION VALUE="0">Select A Schedule</OPTION>
   <OPTION VALUE="169" >Tue Night Mar-May 2026 Season</OPTION>
   <OPTION VALUE="168" >Mon Night Mar-May 2026 Season</OPTION>
   <OPTION VALUE="163" >Wed Night Feb-Apr 2026 Season</OPTION>
   <OPTION VALUE="162" >Sun Night Feb-Apr 2026 Season</OPTION>
   <OPTION VALUE="161" >Tue Night Jan-Mar 2026 Season</OPTION>
   <OPTION VALUE="152" >Tue Night Oct-Dec 2025 Season</OPTION>
   <OPTION VALUE="151" >Mon Night Oct-Dec 2025 Season</OPTION>
   <OPTION VALUE="100" >Tue Night Nov-Jan 2023 Season</OPTION>
</SELECT>
`

const sampleTeamsHTML = `
<SELECT NAME=TEAM_ID onChange="location.href='/schedules.cgi?SCHEDULE_ID=169&TEAM_ID=' + this.value">
   <OPTION VALUE="0">Select A Team</OPTION>
   <OPTION VALUE="6762" >1 - The Sets is Great</OPTION>
   <OPTION VALUE="6751" >2 - Pat's Team</OPTION>
   <OPTION VALUE="6770" >3 - Chaddies Baddies</OPTION>
   <OPTION VALUE="6753" >5 - Papa &amp; Family</OPTION>
</SELECT>
`

func TestDiscoverCurrentScheduleID_TuesdayCurrent(t *testing.T) {
	// Simulate "now" being in April 2026 - should match "Tue Night Mar-May 2026"
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.Local)
	options := parseScheduleOptions(sampleScheduleHTML)

	id, err := findBestSchedule(options, "Tue", now)
	require.NoError(t, err)
	assert.Equal(t, "169", id)
}

func TestDiscoverCurrentScheduleID_MondayCurrent(t *testing.T) {
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.Local)
	options := parseScheduleOptions(sampleScheduleHTML)

	id, err := findBestSchedule(options, "Mon", now)
	require.NoError(t, err)
	assert.Equal(t, "168", id)
}

func TestDiscoverCurrentScheduleID_PrefersCurrent(t *testing.T) {
	// In Feb 2026, "Tue Night Jan-Mar 2026" is current, "Tue Night Mar-May 2026" is future
	now := time.Date(2026, 2, 15, 0, 0, 0, 0, time.Local)
	options := parseScheduleOptions(sampleScheduleHTML)

	id, err := findBestSchedule(options, "Tue", now)
	require.NoError(t, err)
	assert.Equal(t, "161", id) // Jan-Mar 2026
}

func TestDiscoverCurrentScheduleID_NoMatch(t *testing.T) {
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.Local)
	options := parseScheduleOptions(sampleScheduleHTML)

	_, err := findBestSchedule(options, "Fri", now)
	assert.Error(t, err)
}

func TestDiscoverTeamID(t *testing.T) {
	id, fullName, err := DiscoverTeamID(sampleTeamsHTML, "Sets is Great")
	require.NoError(t, err)
	assert.Equal(t, "6762", id)
	assert.Contains(t, fullName, "The Sets is Great")
}

func TestDiscoverTeamID_CaseInsensitive(t *testing.T) {
	id, _, err := DiscoverTeamID(sampleTeamsHTML, "pat's team")
	require.NoError(t, err)
	assert.Equal(t, "6751", id)
}

func TestDiscoverTeamID_NotFound(t *testing.T) {
	_, _, err := DiscoverTeamID(sampleTeamsHTML, "Nonexistent Team")
	assert.Error(t, err)
}

func TestDiscoverTeamID_HTMLEntities(t *testing.T) {
	id, fullName, err := DiscoverTeamID(sampleTeamsHTML, "Papa & Family")
	require.NoError(t, err)
	assert.Equal(t, "6753", id)
	assert.Contains(t, fullName, "Papa & Family")
}
