package config

import (
	"os/exec"
	"strings"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	cfg, err := LoadConfig(map[string]string{
		"BITBUCKET_EMAIL":             "dev@example.com",
		"BITBUCKET_API_TOKEN":         "token",
		"BITBUCKET_DEFAULT_WORKSPACE": "team",
		"BITBUCKET_DEFAULT_REPO":      "repo",
	}, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Config{Email: "dev@example.com", APIToken: "token", DefaultWorkspace: "team", DefaultRepo: "repo"}
	if cfg != want {
		t.Fatalf("got %+v want %+v", cfg, want)
	}
}

func TestLoadConfigMissingCredentials(t *testing.T) {
	_, err := LoadConfig(map[string]string{}, t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	if !strings.Contains(err.Error(), "Set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestLoadConfigTrimsAndTreatsEmptyAsUnset(t *testing.T) {
	cfg, err := LoadConfig(map[string]string{
		"BITBUCKET_EMAIL":             "  dev@example.com  ",
		"BITBUCKET_API_TOKEN":         "token",
		"BITBUCKET_DEFAULT_WORKSPACE": "   ",
		"BITBUCKET_DEFAULT_REPO":      "",
	}, t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Email != "dev@example.com" {
		t.Fatalf("email not trimmed: %q", cfg.Email)
	}
	if cfg.DefaultWorkspace != "" || cfg.DefaultRepo != "" {
		t.Fatalf("expected empty defaults, got %q/%q", cfg.DefaultWorkspace, cfg.DefaultRepo)
	}
}

func gitInitWithRemote(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"remote", "add", "origin", remote},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, out)
		}
	}
	return dir
}

func TestGitRepoRefFromSSH(t *testing.T) {
	dir := gitInitWithRemote(t, "git@bitbucket.org:my-workspace/my-repo-slug.git")
	ref, ok := GitRepoRefFrom(dir)
	if !ok {
		t.Fatal("expected detection")
	}
	if ref.Workspace != "my-workspace" || ref.RepoSlug != "my-repo-slug" {
		t.Fatalf("got %+v", ref)
	}
}

func TestGitRepoRefFromHTTPS(t *testing.T) {
	dir := gitInitWithRemote(t, "https://user@bitbucket.org/other-workspace/other-repo")
	ref, ok := GitRepoRefFrom(dir)
	if !ok {
		t.Fatal("expected detection")
	}
	if ref.Workspace != "other-workspace" || ref.RepoSlug != "other-repo" {
		t.Fatalf("got %+v", ref)
	}
}

func TestGitRepoRefIgnoresNonBitbucket(t *testing.T) {
	dir := gitInitWithRemote(t, "https://github.com/workspace/repo")
	if _, ok := GitRepoRefFrom(dir); ok {
		t.Fatal("expected non-bitbucket remote to be ignored")
	}
}

func TestLoadConfigFillsDefaultsFromGit(t *testing.T) {
	dir := gitInitWithRemote(t, "git@bitbucket.org:git-workspace/git-repo.git")
	cfg, err := LoadConfig(map[string]string{
		"BITBUCKET_EMAIL":     "dev@example.com",
		"BITBUCKET_API_TOKEN": "token",
	}, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultWorkspace != "git-workspace" || cfg.DefaultRepo != "git-repo" {
		t.Fatalf("git defaults not applied: %+v", cfg)
	}
}

func TestEnvDefaultsOverrideGit(t *testing.T) {
	dir := gitInitWithRemote(t, "git@bitbucket.org:git-workspace/git-repo.git")
	cfg, err := LoadConfig(map[string]string{
		"BITBUCKET_EMAIL":             "dev@example.com",
		"BITBUCKET_API_TOKEN":         "token",
		"BITBUCKET_DEFAULT_WORKSPACE": "env-workspace",
		"BITBUCKET_DEFAULT_REPO":      "env-repo",
	}, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultWorkspace != "env-workspace" || cfg.DefaultRepo != "env-repo" {
		t.Fatalf("env should override git: %+v", cfg)
	}
}

func TestResolveRepoRefPrefersExplicit(t *testing.T) {
	cfg := Config{DefaultWorkspace: "team", DefaultRepo: "repo"}
	ref, err := ResolveRepoRef(RepoRef{Workspace: "explicit-team", RepoSlug: "explicit-repo"}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Workspace != "explicit-team" || ref.RepoSlug != "explicit-repo" {
		t.Fatalf("got %+v", ref)
	}
}

func TestResolveRepoRefFallsBackToDefaults(t *testing.T) {
	cfg := Config{DefaultWorkspace: "team", DefaultRepo: "repo"}
	ref, err := ResolveRepoRef(RepoRef{}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Workspace != "team" || ref.RepoSlug != "repo" {
		t.Fatalf("got %+v", ref)
	}
}

func TestResolveRepoRefErrorsWhenUnresolved(t *testing.T) {
	_, err := ResolveRepoRef(RepoRef{}, Config{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Provide workspace and repo") {
		t.Fatalf("unexpected error: %v", err)
	}
}
