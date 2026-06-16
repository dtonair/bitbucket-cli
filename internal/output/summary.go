package output

import "fmt"

// These helpers reproduce the exact text summaries emitted by the original Pi
// extension's tools, operating on decoded JSON objects (map[string]any).

func str(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// PullRequestSummary renders "#<id> <title> [<state>]".
func PullRequestSummary(pr map[string]any) string {
	id := "?"
	if v, ok := pr["id"]; ok {
		id = fmt.Sprintf("%v", numberish(v))
	}
	title, ok := str(pr, "title")
	if !ok || title == "" {
		title = "(untitled)"
	}
	state, ok := str(pr, "state")
	if !ok || state == "" {
		state = "unknown"
	}
	return fmt.Sprintf("#%s %s [%s]", id, title, state)
}

// CommentSummary renders "#<id>: <raw>".
func CommentSummary(comment map[string]any) string {
	id := "?"
	if v, ok := comment["id"]; ok {
		id = fmt.Sprintf("%v", numberish(v))
	}
	raw := ""
	if content, ok := comment["content"].(map[string]any); ok {
		raw, _ = str(content, "raw")
	}
	return fmt.Sprintf("#%s: %s", id, raw)
}

// CommitSummary renders "<hash[:12]> <message>" trimmed.
func CommitSummary(commit map[string]any) string {
	hash := "unknown"
	if h, ok := str(commit, "hash"); ok && h != "" {
		if len(h) > 12 {
			h = h[:12]
		}
		hash = h
	}
	msg, _ := str(commit, "message")
	out := hash + " " + msg
	// Trim a trailing space when message is empty, matching the source.
	if msg == "" {
		return hash
	}
	return out
}

// BranchSummary renders the branch name, falling back to target hash.
func BranchSummary(branch map[string]any) string {
	if name, ok := str(branch, "name"); ok && name != "" {
		return name
	}
	if target, ok := branch["target"].(map[string]any); ok {
		if hash, ok := str(target, "hash"); ok && hash != "" {
			return hash
		}
	}
	return "unknown"
}

// RepoSummary renders "Repository: <full_name|name|unknown>".
func RepoSummary(repo map[string]any) string {
	if full, ok := str(repo, "full_name"); ok && full != "" {
		return "Repository: " + full
	}
	if name, ok := str(repo, "name"); ok && name != "" {
		return "Repository: " + name
	}
	return "Repository: unknown"
}

// numberish normalizes JSON numbers (float64) to integers for display when
// they are whole, so PR ids render as "12" not "12".
func numberish(v any) any {
	if f, ok := v.(float64); ok && f == float64(int64(f)) {
		return int64(f)
	}
	return v
}
