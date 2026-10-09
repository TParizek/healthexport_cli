package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// FetchWorkoutJSON bounds untrusted responses and avoids putting account UID
// query strings or backend response bodies into diagnostics.
func (c *Client) FetchWorkoutJSON(ctx context.Context, path string, query url.Values, target any) error {
	endpoint, err := c.buildURL(path, func(v url.Values) {
		for key, values := range query {
			v[key] = values
		}
	})
	if err != nil {
		return fmt.Errorf("invalid workout API URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("invalid workout request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "healthexport-cli/"+userAgentVersion)
	client := *c.httpClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("workout API request failed (connection or timeout)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body := ""
		if resp.StatusCode == http.StatusConflict {
			body = "workout changed; run workouts list again and use its current revision"
		}
		return &APIError{StatusCode: resp.StatusCode, Endpoint: path, Body: body}
	}
	const maxBytes = 2 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return fmt.Errorf("read workout response failed")
	}
	if len(body) > maxBytes {
		return fmt.Errorf("workout response exceeds size limit")
	}
	if json.Unmarshal(body, target) != nil {
		return fmt.Errorf("invalid workout API response")
	}
	return nil
}
