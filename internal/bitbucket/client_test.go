package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"bitbucket-cli/internal/config"
)

// roundTripFunc lets a test stand in for an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func testClient(rt roundTripFunc) *Client {
	cfg := config.Config{Email: "dev@example.com", APIToken: "token"}
	return NewClient(cfg, WithHTTPClient(&http.Client{Transport: rt}))
}

func TestEncodePathSegment(t *testing.T) {
	if got := EncodePathSegment("team/repo name"); got != "team%2Frepo%20name" {
		t.Fatalf("got %q", got)
	}
}

func TestRequestSendsBasicAuthAndAccept(t *testing.T) {
	var captured *http.Request
	c := testClient(func(r *http.Request) (*http.Response, error) {
		captured = r
		return jsonResponse(200, `{"type":"repository","name":"repo"}`), nil
	})

	var out map[string]any
	if err := c.Request(context.Background(), "/repositories/team/repo", RequestOptions{}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.URL.String() != "https://api.bitbucket.org/2.0/repositories/team/repo" {
		t.Fatalf("unexpected url: %s", captured.URL)
	}
	if captured.Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", captured.Method)
	}
	if got := captured.Header.Get("Authorization"); got != "Basic ZGV2QGV4YW1wbGUuY29tOnRva2Vu" {
		t.Fatalf("unexpected auth header: %s", got)
	}
	if got := captured.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("unexpected accept header: %s", got)
	}
}

func TestRequestPostsJSONBody(t *testing.T) {
	var captured *http.Request
	var body []byte
	c := testClient(func(r *http.Request) (*http.Response, error) {
		captured = r
		body, _ = io.ReadAll(r.Body)
		return jsonResponse(201, `{"id":42}`), nil
	})

	payload := map[string]any{"content": map[string]any{"raw": "hi"}}
	var out map[string]any
	if err := c.Request(context.Background(), "/x", RequestOptions{Method: http.MethodPost, Body: payload}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", captured.Method)
	}
	if got := captured.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content-type: %s", got)
	}
	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if sent["content"].(map[string]any)["raw"] != "hi" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestPaginateFollowsNext(t *testing.T) {
	calls := 0
	c := testClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return jsonResponse(200, `{"values":[{"id":1}],"next":"https://api.bitbucket.org/2.0/next-page"}`), nil
		}
		return jsonResponse(200, `{"values":[{"id":2}]}`), nil
	})

	values, err := c.Paginate(context.Background(), "/first", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestPaginateRespectsLimit(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"values":[{"id":1},{"id":2},{"id":3}]}`), nil
	})
	values, err := c.Paginate(context.Background(), "/first", 2, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected limit of 2, got %d", len(values))
	}
}

func TestRequestMapsHTTPErrors(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{401, "authentication failed"},
		{403, "authorization failed"},
		{404, "resource not found"},
		{429, "rate limit reached"},
		{500, "request failed with 500"},
	}
	for _, tc := range cases {
		c := testClient(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(tc.status, `{"error":{"message":"denied"}}`), nil
		})
		err := c.Request(context.Background(), "/private", RequestOptions{}, nil)
		if err == nil {
			t.Fatalf("status %d: expected error", tc.status)
		}
		var he *HTTPError
		if !errors.As(err, &he) {
			t.Fatalf("status %d: expected *HTTPError, got %T", tc.status, err)
		}
		if he.Status != tc.status {
			t.Fatalf("status mismatch: got %d want %d", he.Status, tc.status)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("status %d: message %q missing %q", tc.status, err.Error(), tc.want)
		}
	}
}

func TestExcerptTruncates(t *testing.T) {
	long := strings.Repeat("x", 600)
	c := testClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(500, long), nil
	})
	err := c.Request(context.Background(), "/x", RequestOptions{}, nil)
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *HTTPError")
	}
	if len(he.Excerpt) != excerptLimit+len("...") {
		t.Fatalf("excerpt not truncated: len=%d", len(he.Excerpt))
	}
}
