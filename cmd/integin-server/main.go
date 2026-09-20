// Command integin-server is the universal INTEGIN server binary.
//
// Deployment topology is selected with --mode=cloud|sovereign|airgap
// (default cloud), which presets INTEGIN_DEPLOY_MODE before booting the shared
// server logic in internal/serverboot. INTEGIN_DEPLOY_MODE set directly in the
// environment takes precedence when --mode is left at its default.
package main

import (
	"flag"
	"fmt"
	"os"

	"integin/internal/serverboot"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "integin-server: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("integin-server", flag.ContinueOnError)
	mode := fs.String("mode", "", "deployment topology: cloud|sovereign|airgap (default: INTEGIN_DEPLOY_MODE or cloud)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mode != "" {
		if err := os.Setenv("INTEGIN_DEPLOY_MODE", *mode); err != nil {
			return fmt.Errorf("set INTEGIN_DEPLOY_MODE: %w", err)
		}
	}
	serverboot.Run()
	return nil
}
