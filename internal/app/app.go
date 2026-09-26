// SPDX-License-Identifier: AGPL-3.0-only

// Package app is the command line of the identity binary.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime"

	"github.com/aleogr/identity/internal/buildinfo"
	"github.com/aleogr/identity/internal/config"
	"github.com/aleogr/identity/internal/logging"
	"github.com/aleogr/identity/internal/server"
)

// Exit statuses.
const (
	ExitOK      = 0
	ExitFailure = 1 // a run-time failure
	ExitUsage   = 2 // a usage or configuration error
)

const usage = `Usage: identity <command>

Commands:
  serve     Start the HTTP server.
  version   Print the build identifier.
  help      Print this help.

The server reads its configuration from IDENTITY_* environment variables and
refuses to start on an unknown variable or an invalid value.
`

// Run executes the command in args (without the program name) and returns
// the process exit status.
func Run(ctx context.Context, args, environ []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return ExitUsage
	}
	switch cmd := args[0]; cmd {
	case "serve", "version":
		if len(args) > 1 {
			_, _ = fmt.Fprintf(stderr, "identity %s: unexpected argument %q\n\n%s", cmd, args[1], usage)
			return ExitUsage
		}
		if cmd == "version" {
			_, _ = fmt.Fprintln(stdout, buildinfo.ID())
			return ExitOK
		}
		return serve(ctx, environ, stdout, stderr)
	case "help", "-h", "-help", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return ExitOK
	default:
		_, _ = fmt.Fprintf(stderr, "identity: unknown command %q\n\n%s", cmd, usage)
		return ExitUsage
	}
}

func serve(ctx context.Context, environ []string, stdout, stderr io.Writer) int {
	cfg, err := config.Load(environ)
	if err != nil {
		reportConfig(logging.New(stderr, slog.LevelInfo, false), err)
		return ExitUsage
	}
	logger := logging.New(stdout, cfg.LogLevel, cfg.LogSource)

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		logger.Error("cannot listen", "addr", cfg.HTTPAddr, "error", err.Error())
		return ExitFailure
	}
	logger.Info("serving",
		"build_id", buildinfo.ID(),
		"go_version", runtime.Version(),
		"addr", ln.Addr().String())

	if err := server.Run(ctx, server.New(server.NewHandler(), logger), ln, cfg.ShutdownTimeout); err != nil {
		logger.Error("server stopped with an error", "error", err.Error())
		return ExitFailure
	}
	logger.Info("stopped")
	return ExitOK
}

func reportConfig(logger *slog.Logger, err error) {
	var cerr *config.Error
	if !errors.As(err, &cerr) {
		logger.Error("invalid configuration; refusing to start", "error", err.Error())
		return
	}
	problems := make([]string, len(cerr.Problems))
	for i, p := range cerr.Problems {
		problems[i] = p.String()
	}
	logger.Error("invalid configuration; refusing to start", "problems", problems)
}
