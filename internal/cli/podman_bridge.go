package cli

import "github.com/mahcialet/agent-env/internal/runtime/compose"

// RunPodmanBridge handles only the private provider child-process environment.
// Ordinary CLI invocations return unhandled without probing provider tools.
func RunPodmanBridge(args []string) (bool, int) {
	return compose.RunPodmanBridge(args)
}
