//go:build !windows

package process

func retryableStateRemoval(error) bool { return false }
