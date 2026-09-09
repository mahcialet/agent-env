package main

import (
	"fmt"
	"strings"
)

// Validate a branch name without consulting checkout state or expanding Git
// revision syntax (including the checkout-history shorthand @{-1}).
func validPlanBaseBranch(branch string) bool {
	if branch == "" || branch == "HEAD" || strings.HasPrefix(branch, "-") || strings.HasSuffix(branch, ".") || strings.Contains(branch, "..") || strings.Contains(branch, "@{") || strings.ContainsAny(branch, "~^:?*[\\") {
		return false
	}
	for _, r := range branch {
		if r <= 0x20 || r == 0x7f {
			return false
		}
	}
	for _, component := range strings.Split(branch, "/") {
		if component == "" || strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".lock") {
			return false
		}
	}
	return true
}

func planBaseRevision(root, branch string) (string, error) {
	if !validPlanBaseBranch(branch) {
		return "", fmt.Errorf("invalid base_branch %q: expected a Git branch name", branch)
	}
	sha, err := planGit(root, "rev-parse", "--verify", "--end-of-options", "refs/heads/"+branch+"^{commit}")
	if err == nil {
		return sha, nil
	}
	return planGit(root, "rev-parse", "--verify", "--end-of-options", "refs/remotes/origin/"+branch+"^{commit}")
}
