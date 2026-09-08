// Command uihelper explicitly builds the optional Android observer APK.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mahcialet/agent-env/internal/runtime/android/uihelper"
	"os"
)

func main() {
	var o uihelper.BuildOptions
	flag.StringVar(&o.SDK, "sdk", "", "installed Android SDK directory")
	flag.StringVar(&o.JDK, "jdk", "", "installed JDK directory")
	flag.StringVar(&o.Platform, "platform", "android-35", "installed Android platform")
	flag.StringVar(&o.BuildTools, "build-tools", "36.0.0", "installed Android build-tools version")
	flag.StringVar(&o.Output, "output", "", "new output directory")
	flag.Parse()
	m, err := uihelper.Build(context.Background(), o)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(m)
}
