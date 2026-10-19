package sources

import (
	"bytes"
	"context"
	"errors"
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
	RequestDelay     time.Duration
	MaxRetryAfter    time.Duration
	Sleep            func(time.Duration)
	// Optional replay seams. All requests, including redirects, use Transport.
	Transport     http.RoundTripper
	CheckRedirect func(*http.Request, []*http.Request) error
	Now           func() time.Time
	// Wait receives retry=true for backoff and false for per-attempt delay.
	Wait func(context.Context, time.Duration, bool) error
}

const maxSourceResponseBytes int64 = 10 << 20

type sourceHTTPStatusError struct{ status int }

func (e sourceHTTPStatusError) Error() string { return fmt.Sprintf("source HTTP status %d", e.status) }
func isSourceHTTPStatus(err error, status int) bool {
	var e sourceHTTPStatusError
	return errors.As(err, &e) && e.status == status
}

// HTTPClient makes source HTTP requests through an injected base URL.
type HTTPClient struct {
	baseURL          string
	userAgent        string
	client           *http.Client
	maxRetries       int
	maxResponseBytes int64
	headers          map[string]string
	requestDelay     time.Duration
	maxRetryAfter    time.Duration
	sleep            func(time.Duration)
	now              func() time.Time
	waitContext      func(context.Context, time.Duration, bool) error
}

func NewHTTPClient(o HTTPClientOptions) HTTPClient {
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.Sleep == nil {
		o.Sleep = time.Sleep
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.MaxResponseBytes == 0 {
		o.MaxResponseBytes = maxSourceResponseBytes
	}
	headers := map[string]string{}
	for k, v := range o.Headers {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" && v != "" {
			headers[k] = v
		}
	}
	return HTTPClient{
		baseURL: strings.TrimRight(o.BaseURL, "/"), userAgent: o.UserAgent,
		client:     &http.Client{Timeout: o.Timeout, Transport: o.Transport, CheckRedirect: o.CheckRedirect},
		maxRetries: o.MaxRetries, maxResponseBytes: o.MaxResponseBytes, headers: headers,
		requestDelay: o.RequestDelay, maxRetryAfter: o.MaxRetryAfter, sleep: o.Sleep, now: o.Now, waitContext: o.Wait,
	}
}
func (c HTTPClient) Get(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, query, nil, "")
}
func (c HTTPClient) Post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.postWithContentType(ctx, path, body, "application/json")
}
func (c HTTPClient) PostText(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.postWithContentType(ctx, path, body, "text/plain")
}
func (c HTTPClient) postWithContentType(ctx context.Context, path string, body []byte, contentType string) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, nil, body, contentType)
}
func (c HTTPClient) do(ctx context.Context, method, path string, query map[string]string, body []byte, contentType string) ([]byte, error) {
	if path == "" || strings.Contains(path, "://") || strings.HasPrefix(path, "//") {
		return nil, fmt.Errorf("source request path must be relative")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("source base URL is required")
	}
	endpoint, e := url.Parse(c.baseURL + "/" + strings.TrimLeft(path, "/"))
	if e != nil {
		return nil, e
	}
	values := endpoint.Query()
	for k, v := range query {
		values.Set(k, v)
	}
	endpoint.RawQuery = values.Encode()
	var lastStatus int
	var retryLater time.Duration
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if e := c.wait(ctx, c.requestDelay, false); e != nil {
			return nil, e
		}
		request, e := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
		if e != nil {
			return nil, e
		}
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		if c.userAgent != "" {
			request.Header.Set("User-Agent", c.userAgent)
		}
		for k, v := range c.headers {
			request.Header.Set(k, v)
		}
		response, e := c.client.Do(request)
		if e != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if attempt < c.maxRetries {
				continue
			}
			return nil, e
		}
		lastStatus = response.StatusCode
		if lastStatus >= 200 && lastStatus < 300 {
			defer response.Body.Close()
			return readBoundedResponse(response, c.maxResponseBytes)
		}
		delay := retryDelayAt(response, c.now())
		_ = response.Body.Close()
		if attempt < c.maxRetries && retryableStatus(lastStatus) {
			if delay == 0 && lastStatus == http.StatusTooManyRequests {
				delay = time.Duration(1<<uint(attempt)) * time.Second
			}
			if c.maxRetryAfter > 0 && delay > c.maxRetryAfter {
				retryLater = delay
				break
			}
			if e := c.wait(ctx, delay, true); e != nil {
				return nil, e
			}
			continue
		}
		break
	}
	if lastStatus == http.StatusTooManyRequests {
		return nil, rateLimitError(retryLater)
	}
	return nil, sourceHTTPStatusError{status: lastStatus}
}
func readBoundedResponse(response *http.Response, maxBytes int64) ([]byte, error) {
	if response.ContentLength > maxBytes {
		return nil, fmt.Errorf("source response too large: %d bytes exceeds %d", response.ContentLength, maxBytes)
	}
	b, e := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("source response too large: exceeds %d", maxBytes)
	}
	return b, nil
}
func retryableStatus(status int) bool                  { return status == http.StatusTooManyRequests || status >= 500 }
func retryDelay(response *http.Response) time.Duration { return retryDelayAt(response, time.Now()) }
func retryDelayAt(response *http.Response, now time.Time) time.Duration {
	value := strings.TrimSpace(response.Header.Get("Retry-After"))
	if value == "" {
		return 0
	}
	seconds, e := strconv.Atoi(value)
	if e == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	when, e := http.ParseTime(value)
	if e != nil {
		return 0
	}
	if d := when.Sub(now); d > 0 {
		return d
	}
	return 0
}
func rateLimitError(retryAfter time.Duration) error {
	if retryAfter > 0 {
		return fmt.Errorf("source HTTP status 429: rate limited; retry after %s; try again later or configure an API key (check connector env vars)", retryAfter.Round(time.Second))
	}
	return fmt.Errorf("source HTTP status 429: rate limited; reduce request frequency, try again later, or configure an API key (check connector env vars)")
}
func (c HTTPClient) wait(ctx context.Context, d time.Duration, retry bool) error {
	if c.waitContext != nil {
		return c.waitContext(ctx, d, retry)
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if d > 0 {
		c.sleep(d)
	}
	return ctx.Err()
}
