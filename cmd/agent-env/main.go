package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/mahcialet/agent-env/internal/cli"
)

func main() {
	if handled, code := cli.RunPodmanBridge(os.Args[1:]); handled {
		os.Exit(code)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := cli.New(os.Stdout, os.Stderr).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
