package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadBody(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "body.md")
	if err := os.WriteFile(file, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("flag only", func(t *testing.T) {
		v, ok, err := readBody("hello", "", strings.NewReader(""))
		if err != nil || !ok || v != "hello" {
			t.Fatalf("got %q ok=%v err=%v", v, ok, err)
		}
	})
	t.Run("empty flag not provided", func(t *testing.T) {
		_, ok, err := readBody("", "", strings.NewReader(""))
		if err != nil || ok {
			t.Fatalf("expected not provided, ok=%v err=%v", ok, err)
		}
	})
	t.Run("file only", func(t *testing.T) {
		v, ok, err := readBody("", file, strings.NewReader(""))
		if err != nil || !ok || v != "from file" {
			t.Fatalf("got %q ok=%v err=%v", v, ok, err)
		}
	})
	t.Run("stdin", func(t *testing.T) {
		v, ok, err := readBody("", "-", strings.NewReader("piped"))
		if err != nil || !ok || v != "piped" {
			t.Fatalf("got %q ok=%v err=%v", v, ok, err)
		}
	})
	t.Run("both is error", func(t *testing.T) {
		if _, _, err := readBody("a", file, strings.NewReader("")); err == nil {
			t.Fatal("expected mutual-exclusion error")
		}
	})
	t.Run("missing file is error", func(t *testing.T) {
		if _, _, err := readBody("", filepath.Join(dir, "nope.md"), strings.NewReader("")); err == nil {
			t.Fatal("expected read error")
		}
	})
}

func TestPRUpdateReadModifyWrite(t *testing.T) {
	var methods []string
	var putBody []byte
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		switch r.Method {
		case http.MethodGet:
			return jsonResp(200, `{"id":42,"title":"Old title","description":"old","reviewers":[{"uuid":"{abc}"}],"state":"OPEN"}`), nil
		case http.MethodPut:
			putBody, _ = io.ReadAll(r.Body)
			return jsonResp(200, `{"id":42,"title":"Old title","state":"OPEN"}`), nil
		default:
			t.Fatalf("unexpected method %s", r.Method)
			return nil, nil
		}
	}, "pr", "update", "42", "--description-file", "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 2 || methods[0] != http.MethodGet || methods[1] != http.MethodPut {
		t.Fatalf("expected GET then PUT, got %v", methods)
	}
	var sent map[string]any
	if err := json.Unmarshal(putBody, &sent); err != nil {
		t.Fatalf("put body not json: %v", err)
	}
	if sent["title"] != "Old title" {
		t.Fatalf("expected carried-over title, got %v", sent["title"])
	}
	if _, hasDesc := sent["description"]; !hasDesc {
		t.Fatalf("expected description in put body: %s", putBody)
	}
	if _, hasRev := sent["reviewers"]; !hasRev {
		t.Fatalf("expected reviewers carried over: %s", putBody)
	}
	if !strings.Contains(out, `"id": 42`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPRUpdateTitleOnlyOmitsDescription(t *testing.T) {
	var putBody []byte
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet {
			return jsonResp(200, `{"id":7,"title":"Old","description":"keep me","state":"OPEN"}`), nil
		}
		putBody, _ = io.ReadAll(r.Body)
		return jsonResp(200, `{"id":7}`), nil
	}, "pr", "update", "7", "--title", "New title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal(putBody, &sent)
	if sent["title"] != "New title" {
		t.Fatalf("expected new title, got %v", sent["title"])
	}
	if _, hasDesc := sent["description"]; hasDesc {
		t.Fatalf("title-only update should not send description: %s", putBody)
	}
}

func TestPRUpdateNoFieldsFails(t *testing.T) {
	called := false
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		called = true
		return jsonResp(200, `{}`), nil
	}, "pr", "update", "5")
	if err == nil {
		t.Fatal("expected error when no fields provided")
	}
	if called {
		t.Fatal("should not call API when nothing to update")
	}
}

func TestPRCreatePostsBody(t *testing.T) {
	var gotURL, gotMethod string
	var body []byte
	out, err := run(t, func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		gotMethod = r.Method
		body, _ = io.ReadAll(r.Body)
		return jsonResp(201, `{"id":101,"title":"Add feature","state":"OPEN"}`), nil
	}, "pr", "create", "--source", "feature/x", "--destination", "main", "--title", "Add feature", "--description", "body text", "--close-source-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if !strings.HasSuffix(gotURL, "/repositories/team/repo/pullrequests") {
		t.Fatalf("unexpected url: %s", gotURL)
	}
	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	src := sent["source"].(map[string]any)["branch"].(map[string]any)["name"]
	if src != "feature/x" {
		t.Fatalf("unexpected source branch: %v", src)
	}
	dst := sent["destination"].(map[string]any)["branch"].(map[string]any)["name"]
	if dst != "main" {
		t.Fatalf("unexpected destination: %v", dst)
	}
	if sent["title"] != "Add feature" || sent["description"] != "body text" {
		t.Fatalf("unexpected body: %s", body)
	}
	if _, has := sent["content"]; has {
		t.Fatalf("create body must not send content; Bitbucket rejects content.markup: %s", body)
	}
	if sent["close_source_branch"] != true {
		t.Fatalf("expected close_source_branch true: %s", body)
	}
	if !strings.Contains(out, `"id": 101`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestPRCreateOmitsDestinationWhenUnset(t *testing.T) {
	var body []byte
	_, err := run(t, func(r *http.Request) (*http.Response, error) {
		body, _ = io.ReadAll(r.Body)
		return jsonResp(201, `{"id":1}`), nil
	}, "pr", "create", "--source", "feature/x", "--title", "T")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal(body, &sent)
	if _, has := sent["destination"]; has {
		t.Fatalf("destination should be omitted when unset: %s", body)
	}
	if _, has := sent["description"]; has {
		t.Fatalf("description should be omitted when unset: %s", body)
	}
}

func TestPRCreateRequiresSourceAndTitle(t *testing.T) {
	// MarkFlagRequired enforces these before RunE; expect an error.
	if _, err := run(t, nil, "pr", "create", "--title", "T"); err == nil {
		t.Fatal("expected error when --source missing")
	}
	if _, err := run(t, nil, "pr", "create", "--source", "feature/x"); err == nil {
		t.Fatal("expected error when --title missing")
	}
}
