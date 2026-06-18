package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPClientOptions configures a local-testable scholarly source HTTP client.
type HTTPClientOptions struct {
	BaseURL          string
	UserAgent        string
	Timeout          time.Duration
	MaxRetries       int
	MaxResponseBytes int64
	Headers          map[string]string
	Sleep            func(time.Duration)
}

const maxSourceResponseBytes int64 = 10 << 20

// HTTPClient makes source HTTP requests through an injected base URL.
type HTTPClient struct {
	baseURL          string
	userAgent        string
	client           *http.Client
	maxRetries       int
	maxResponseBytes int64
	headers          map[string]string
	sleep            func(time.Duration)
}

// NewHTTPClient creates an HTTP client suitable for real adapters and httptest servers.
func NewHTTPClient(options HTTPClientOptions) HTTPClient {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	sleep := options.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	maxBytes := options.MaxResponseBytes
	if maxBytes == 0 {
		maxBytes = maxSourceResponseBytes
	}
	headers := map[string]string{}
	for key, value := range options.Headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			headers[key] = value
		}
	}
	return HTTPClient{
		baseURL:          strings.TrimRight(options.BaseURL, "/"),
		userAgent:        options.UserAgent,
		client:           &http.Client{Timeout: timeout},
		maxRetries:       options.MaxRetries,
		maxResponseBytes: maxBytes,
		headers:          headers,
		sleep:            sleep,
	}
}

// Get fetches a relative source path with query parameters and returns the response body.
func (c HTTPClient) Get(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	if path == "" || strings.Contains(path, "://") || strings.HasPrefix(path, "//") {
		return nil, fmt.Errorf("source request path must be relative")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("source base URL is required")
	}
	endpoint, err := url.Parse(c.baseURL + "/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return nil, err
	}
	values := endpoint.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	endpoint.RawQuery = values.Encode()

	var lastStatus int
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, err
		}
		if c.userAgent != "" {
			request.Header.Set("User-Agent", c.userAgent)
		}
		for key, value := range c.headers {
			request.Header.Set(key, value)
		}
		response, err := c.client.Do(request)
		if err != nil {
			if attempt < c.maxRetries {
				continue
			}
			return nil, err
		}
		lastStatus = response.StatusCode
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			defer response.Body.Close()
			return readBoundedResponse(response, c.maxResponseBytes)
		}
		delay := retryDelay(response)
		_ = response.Body.Close()
		if attempt < c.maxRetries && retryableStatus(lastStatus) {
			if delay == 0 && lastStatus == http.StatusTooManyRequests {
				delay = time.Duration(1<<uint(attempt)) * time.Second
			}
			if delay > 0 {
				c.sleep(delay)
			}
			continue
		}
		break
	}
	return nil, fmt.Errorf("source HTTP status %d", lastStatus)
}

func readBoundedResponse(response *http.Response, maxBytes int64) ([]byte, error) {
	if response.ContentLength > maxBytes {
		return nil, fmt.Errorf("source response too large: %d bytes exceeds %d", response.ContentLength, maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("source response too large: exceeds %d", maxBytes)
	}
	return data, nil
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout || status >= 500
}

func retryDelay(response *http.Response) time.Duration {
	value := strings.TrimSpace(response.Header.Get("Retry-After"))
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
