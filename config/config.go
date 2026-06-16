// Package config resolves Bitbucket credentials and repo defaults from
// environment variables, with optional fallback to the local git remote for
// the default workspace/repo. No config file is read.
package config

import (
	"fmt"
	"os/exec"
	"strings"
)

// Config holds resolved Bitbucket credentials and optional repo defaults.
type Config struct {
	Email            string
	APIToken         string
	DefaultWorkspace string
	DefaultRepo      string
}

// RepoRef is an unresolved workspace/repo reference, typically from CLI flags.
type RepoRef struct {
	Workspace string
	RepoSlug  string
}

// ResolvedRepoRef is a fully resolved workspace/repo pair.
type ResolvedRepoRef struct {
	Workspace string
	RepoSlug  string
}

func trimmed(env map[string]string, key string) string {
	return strings.TrimSpace(env[key])
}

// LoadConfig builds a Config from the given environment map. gitCwd is the
// directory used for git remote auto-detection of the default workspace/repo
// (pass "" for the current process directory). Email and API token are
// required; everything else is optional. Git detection only fills defaults
// that the environment did not provide.
func LoadConfig(env map[string]string, gitCwd string) (Config, error) {
	email := trimmed(env, "BITBUCKET_EMAIL")
	apiToken := trimmed(env, "BITBUCKET_API_TOKEN")

	if email == "" || apiToken == "" {
		return Config{}, fmt.Errorf("Set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN before using bitbucket-cli.")
	}

	workspace := trimmed(env, "BITBUCKET_DEFAULT_WORKSPACE")
	repo := trimmed(env, "BITBUCKET_DEFAULT_REPO")

	if workspace == "" || repo == "" {
		if ref, ok := GitRepoRefFrom(gitCwd); ok {
			if workspace == "" {
				workspace = ref.Workspace
			}
			if repo == "" {
				repo = ref.RepoSlug
			}
		}
	}

	return Config{
		Email:            email,
		APIToken:         apiToken,
		DefaultWorkspace: workspace,
		DefaultRepo:      repo,
	}, nil
}

// GitRepoRefFrom inspects the `origin` remote in gitCwd and returns the
// workspace/repo if it points at bitbucket.org. All errors (no git, no remote,
// non-Bitbucket remote, parse failures) are swallowed and reported as ok=false.
func GitRepoRefFrom(gitCwd string) (RepoRef, bool) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if gitCwd != "" {
		cmd.Dir = gitCwd
	}
	out, err := cmd.Output()
	if err != nil {
		return RepoRef{}, false
	}

	url := strings.TrimSpace(string(out))
	if url == "" {
		return RepoRef{}, false
	}
	url = strings.TrimSuffix(url, ".git")

	if !strings.Contains(url, "bitbucket.org") {
		return RepoRef{}, false
	}

	var remaining string
	switch {
	case strings.Contains(url, "bitbucket.org:"):
		remaining = strings.SplitN(url, "bitbucket.org:", 2)[1]
	case strings.Contains(url, "bitbucket.org/"):
		remaining = strings.SplitN(url, "bitbucket.org/", 2)[1]
	default:
		return RepoRef{}, false
	}

	parts := strings.Split(remaining, "/")
	if len(parts) < 2 {
		return RepoRef{}, false
	}
	repoSlug := parts[len(parts)-1]
	workspace := parts[len(parts)-2]
	if workspace == "" || repoSlug == "" {
		return RepoRef{}, false
	}
	return RepoRef{Workspace: workspace, RepoSlug: repoSlug}, true
}

// ResolveRepoRef resolves a workspace/repo, preferring explicit input over the
// config defaults. It errors when neither source yields both values.
func ResolveRepoRef(input RepoRef, cfg Config) (ResolvedRepoRef, error) {
	workspace := strings.TrimSpace(input.Workspace)
	if workspace == "" {
		workspace = cfg.DefaultWorkspace
	}
	repoSlug := strings.TrimSpace(input.RepoSlug)
	if repoSlug == "" {
		repoSlug = cfg.DefaultRepo
	}

	if workspace == "" || repoSlug == "" {
		return ResolvedRepoRef{}, fmt.Errorf("Provide workspace and repo via --workspace/--repo, or set BITBUCKET_DEFAULT_WORKSPACE and BITBUCKET_DEFAULT_REPO, or run inside a Bitbucket git repository.")
	}

	return ResolvedRepoRef{Workspace: workspace, RepoSlug: repoSlug}, nil
}
