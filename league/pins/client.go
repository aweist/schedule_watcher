package pins

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type PINSClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *PINSClient {
	return &PINSClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchSchedulesPage fetches the main schedules page to discover available seasons.
func (c *PINSClient) FetchSchedulesPage() (string, error) {
	return c.fetch(fmt.Sprintf("%s/schedules.cgi", c.baseURL))
}

// FetchTeamsPage fetches the schedule page with a selected season to discover teams.
func (c *PINSClient) FetchTeamsPage(scheduleID string) (string, error) {
	return c.fetch(fmt.Sprintf("%s/schedules.cgi?SCHEDULE_ID=%s", c.baseURL, scheduleID))
}

// FetchTeamSchedule fetches the full schedule page for a specific team.
func (c *PINSClient) FetchTeamSchedule(scheduleID, teamID string) (string, error) {
	return c.fetch(fmt.Sprintf("%s/schedules.cgi?SCHEDULE_ID=%s&TEAM_ID=%s", c.baseURL, scheduleID, teamID))
}

func (c *PINSClient) fetch(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "ScheduleWatcher/1.0")
	req.Header.Set("Accept", "text/html")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	return string(body), nil
}
