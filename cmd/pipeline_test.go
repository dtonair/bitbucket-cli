package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPipelineListPathAndQuery(t *testing.T) {
	var gotURL string
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return jsonResp(200, `{"values":[
			{"build_number":1,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main"},"trigger":{"name":"push"},"duration_in_seconds":120}
		]}`), nil
	}, "pipeline", "list", "--limit", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotURL, "/repositories/team/repo/pipelines") {
		t.Fatalf("unexpected path: %s", gotURL)
	}
	if !strings.Contains(gotURL, "sort=-created_on") || !strings.Contains(gotURL, "pagelen=50") {
		t.Fatalf("query missing params: %s", gotURL)
	}
}

func TestPipelineListWithStateFilter(t *testing.T) {
	var gotURL string
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return jsonResp(200, `{"values":[]}`), nil
	}, "pipeline", "list", "--state", "IN_PROGRESS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotURL, "state=IN_PROGRESS") {
		t.Fatalf("expected state=IN_PROGRESS in url: %s", gotURL)
	}
}

func TestPipelineListStateLowercaseNormalized(t *testing.T) {
	var gotURL string
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return jsonResp(200, `{"values":[]}`), nil
	}, "pipeline", "list", "--state", "in_progress")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotURL, "state=IN_PROGRESS") {
		t.Fatalf("expected state to be uppercase, got: %s", gotURL)
	}
}

func TestPipelineListPretty(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"values":[
			{"build_number":42,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main"},"trigger":{"name":"manual"},"duration_in_seconds":150}
		]}`), nil
	}, "pipeline", "list", "--pretty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "#42 COMPLETED SUCCESSFUL main manual 2m30s"
	if strings.TrimSpace(out) != expected {
		t.Fatalf("unexpected pretty output:\ngot:  %q\nwant: %q", out, expected)
	}
}

func TestPipelineListPrettyRunning(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"values":[
			{"build_number":5,"state":{"name":"IN_PROGRESS"},"target":{"ref_name":"feature/x"},"trigger":{"name":"push"}}
		]}`), nil
	}, "pipeline", "list", "--pretty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Running pipeline has no result, no duration.
	expected := "#5 IN_PROGRESS feature/x push -"
	if strings.TrimSpace(out) != expected {
		t.Fatalf("unexpected pretty output:\ngot:  %q\nwant: %q", out, expected)
	}
}

func TestPipelineListEmpty(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"values":[]}`), nil
	}, "pipeline", "list", "--pretty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "No pipelines found." {
		t.Fatalf("expected empty message, got: %q", out)
	}
}

func TestPipelineListJSON(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		return jsonResp(200, `{"values":[
			{"build_number":1,"state":{"name":"COMPLETED"}}
		]}`), nil
	}, "pipeline", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("not json array: %v\n%s", err, out)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(arr))
	}
}

func TestPipelineListHTTPError(t *testing.T) {
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		return jsonResp(400, `{"error":{"message":"bad request"}}`), nil
	}, "pipeline", "list", "--state", "BOGUS")
	if err == nil {
		t.Fatal("expected error for bad state")
	}
}

func TestPipelineGetJSON(t *testing.T) {
	var pipelineCalled, stepsCalled bool
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.Contains(url, "/steps") {
			stepsCalled = true
			return jsonResp(200, `{"values":[
				{"name":"Build","state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}}},
				{"name":"Test","state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}}}
			]}`), nil
		}
		pipelineCalled = true
		return jsonResp(200, `{"uuid":"abc-123","build_number":7,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main"},"trigger":{"name":"push"},"created_on":"2025-01-01T00:00:00Z","duration_in_seconds":60}`), nil
	}, "pipeline", "get", "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pipelineCalled || !stepsCalled {
		t.Fatalf("pipeline=%v steps=%v", pipelineCalled, stepsCalled)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("not json: %v\n%s", err, out)
	}
	steps := m["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
}

func TestPipelineGetPretty(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.Contains(url, "/steps") {
			return jsonResp(200, `{"values":[
				{"name":"Build","state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}}},
				{"name":"Deploy","state":{"name":"IN_PROGRESS"}}
			]}`), nil
		}
		return jsonResp(200, `{"uuid":"abc-123","build_number":7,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main","commit":{"hash":"abc123def456789"}},"trigger":{"name":"push"},"created_on":"2025-01-01T00:00:00Z","duration_in_seconds":65}`), nil
	}, "pipeline", "get", "abc-123", "--pretty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify key sections are present.
	if !strings.Contains(out, "Pipeline #7") {
		t.Fatalf("missing pipeline header: %s", out)
	}
	if !strings.Contains(out, "State: COMPLETED (SUCCESSFUL)") {
		t.Fatalf("missing state: %s", out)
	}
	if !strings.Contains(out, "Branch: main") {
		t.Fatalf("missing branch: %s", out)
	}
	if !strings.Contains(out, "Commit: abc123def456") {
		t.Fatalf("missing commit: %s", out)
	}
	if !strings.Contains(out, "Duration: 1m5s") {
		t.Fatalf("missing duration: %s", out)
	}
	if !strings.Contains(out, "✓ Build") {
		t.Fatalf("missing successful step: %s", out)
	}
	if !strings.Contains(out, "● Deploy") {
		t.Fatalf("missing in-progress step: %s", out)
	}
}

func TestPipelineGetNoSteps(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.Contains(url, "/steps") {
			return jsonResp(200, `{"values":[]}`), nil
		}
		return jsonResp(200, `{"uuid":"abc-123","build_number":1,"state":{"name":"PENDING"},"target":{"ref_name":"main"},"trigger":{"name":"manual"}}`), nil
	}, "pipeline", "get", "abc-123", "--pretty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Steps (0):") && !strings.Contains(out, "(none)") {
		t.Fatalf("expected empty steps indication: %s", out)
	}
}

func TestPipelineGetStepsErrorFallback(t *testing.T) {
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		url := r.URL.String()
		if strings.Contains(url, "/steps") {
			return jsonResp(500, `{"error":{"message":"internal"}}`), nil
		}
		return jsonResp(200, `{"uuid":"abc-123","build_number":3,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main"},"trigger":{"name":"push"}}`), nil
	}, "pipeline", "get", "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still return pipeline JSON with empty steps array.
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("not json: %v\n%s", err, out)
	}
	steps := m["steps"]
	if steps == nil {
		t.Fatal("steps should be present (empty array) even on error")
	}
}

func TestPipelineGetNotFound(t *testing.T) {
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		return jsonResp(404, `{"error":{"message":"not found"}}`), nil
	}, "pipeline", "get", "nonexistent-uuid")
	if err == nil {
		t.Fatal("expected 404 error")
	}
}

func TestPipelineGetEmptyUUID(t *testing.T) {
	_, err := run(t, nil, "pipeline", "get", "   ")
	if err == nil {
		t.Fatal("expected error for empty uuid")
	}
}
