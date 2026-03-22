package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) FetchEncryptedData(uid string, typeIDs []int, dateFrom, dateTo string) ([]EncryptedPackage, error) {
	endpoint, err := c.buildURL("/healthdata/encrypted", func(values url.Values) {
		values.Set("uid", uid)
		for _, typeID := range typeIDs {
			values.Add("type[]", strconv.Itoa(typeID))
		}
		values.Set("dateFrom", dateFrom)
		values.Set("dateTo", dateTo)
	})
	if err != nil {
		return nil, err
	}

	req, err := c.newJSONRequest(http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}

	var packages []EncryptedPackage
	if err := c.doJSON(req, &packages); err != nil {
		return nil, err
	}

	return packages, nil
}

func (c *Client) FetchHealthTypes() (*HealthTypesResponse, error) {
	endpoint, err := c.buildURL("/healthtypes", nil)
	if err != nil {
		return nil, err
	}

	req, err := c.newJSONRequest(http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}

	var response HealthTypesResponse
	if err := c.doJSON(req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) buildURL(endpointPath string, mutateQuery func(url.Values)) (string, error) {
	baseURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url %q: %w", c.BaseURL, err)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + endpointPath

	query := baseURL.Query()
	if mutateQuery != nil {
		mutateQuery(query)
	}
	baseURL.RawQuery = query.Encode()

	return baseURL.String(), nil
}

func (c *Client) newJSONRequest(method, endpoint string) (*http.Request, error) {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for %s: %w", endpoint, err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "healthexport-cli/"+userAgentVersion)

	return req, nil
}

func (c *Client) doJSON(req *http.Request, target any) error {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("send request to %s: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("read error response from %s: %w", req.URL.Path, readErr)
		}

		return &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Endpoint:   req.URL.Path,
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response from %s: %w", req.URL.Path, err)
	}

	return nil
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}

	return &http.Client{
		Timeout: 30 * time.Second,
	}
}
