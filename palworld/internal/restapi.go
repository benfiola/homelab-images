package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RestAPIClient talks to Palworld's own dedicated-server REST API
// (https://tech.palworldgame.com/category/rest-api). Port and admin
// password come from the same protected-fields source written into
// PalWorldSettings.ini, so client and game can't disagree on credentials.
type RestAPIClient struct {
	Port          int
	AdminPassword string
	HTTPClient    *http.Client
}

func NewRestAPIClient(port int, adminPassword string) *RestAPIClient {
	return &RestAPIClient{
		Port:          port,
		AdminPassword: adminPassword,
		HTTPClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *RestAPIClient) post(ctx context.Context, path string, body any) error {
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}

	url := fmt.Sprintf("http://localhost:%d/v1/api/%s", c.Port, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", c.AdminPassword)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("rest api %s: unexpected status %d", path, resp.StatusCode)
	}
	return nil
}

// Save triggers an immediate world save.
func (c *RestAPIClient) Save(ctx context.Context) error {
	return c.post(ctx, "save", nil)
}

// Shutdown asks the server to shut itself down gracefully after waitSeconds,
// broadcasting message to any connected players.
func (c *RestAPIClient) Shutdown(ctx context.Context, waitSeconds int, message string) error {
	return c.post(ctx, "shutdown", map[string]any{
		"waittime": waitSeconds,
		"message":  message,
	})
}
