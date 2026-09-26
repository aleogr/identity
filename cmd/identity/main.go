// SPDX-License-Identifier: AGPL-3.0-only

// Command identity is the identity platform's server and command line.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/aleogr/identity/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	code := app.Run(ctx, os.Args[1:], os.Environ(), os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
