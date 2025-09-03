package client

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aweist/schedule-watcher/models"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *APIClient) FetchSchedule(instance, compID string) (*models.Schedule, error) {
	url := fmt.Sprintf("%s/api/file?instance=%s&compId=%s", c.baseURL, instance, compID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Build referer URL based on the pattern from the log file
	refererURL := fmt.Sprintf("%s/index?pageId=ul3cy&compId=%s&viewerCompId=%s&siteRevision=4641&viewMode=site&deviceType=desktop&locale=en&tz=America%%2FDenver&regionalLanguage=en&width=985&height=2155&instance=%s&currency=USD&currentCurrency=USD&commonConfig=%%7B%%22brand%%22:%%22wix%%22,%%22host%%22:%%22VIEWER%%22,%%22bsi%%22:%%221e7dafce-7b68-498a-89d4-f105b8e7eddb%%7C6%%22,%%22siteRevision%%22:%%224641%%22,%%22BSI%%22:%%221e7dafce-7b68-498a-89d4-f105b8e7eddb%%7C6%%22%%7D&currentRoute=.%%2Fthursdayleagues&vsi=5cef7406-d83a-41b7-846f-e6fbc85c11dd", 
		c.baseURL, compID, compID, instance)
	
	req.Header.Set("User-Agent", "PostmanRuntime/7.45.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Referer", refererURL)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Log the full URL for debugging (but mask sensitive parts)
		maskedURL := fmt.Sprintf("%s/api/file?instance=[MASKED]&compId=%s", c.baseURL, compID)
		return nil, fmt.Errorf("API request failed - URL: %s, Status: %d, Body: %s", maskedURL, resp.StatusCode, string(body))
	}
	
	// Handle gzip compression
	var reader io.Reader = resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}
	
	var schedule models.Schedule
	if err := json.NewDecoder(reader).Decode(&schedule); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	return &schedule, nil
}