// SPDX-License-Identifier: AGPL-3.0-only

// Package config loads the deployment configuration from IDENTITY_*
// environment variables. It refuses anything it does not understand, naming
// the variable and quoting the value (docs/design.md, section 7; threat X-05).
package config

import (
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Prefix marks the variables this package owns. An unknown variable with this
// prefix is refused, so a typo cannot silently leave a default in place.
const Prefix = "IDENTITY_"

const (
	varHTTPAddr        = "IDENTITY_HTTP_ADDR"
	varLogLevel        = "IDENTITY_LOG_LEVEL"
	varLogSource       = "IDENTITY_LOG_SOURCE"
	varShutdownTimeout = "IDENTITY_SHUTDOWN_TIMEOUT"
	// varPort is set by Cloud Run; it is the only variable read without the prefix.
	varPort = "PORT"
)

var known = map[string]bool{varHTTPAddr: true, varLogLevel: true, varLogSource: true, varShutdownTimeout: true}

var levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

const (
	minShutdownTimeout = time.Second
	maxShutdownTimeout = time.Minute
	// maxQuoted is the most bytes of a value an error message repeats.
	maxQuoted = 64
)

// Config is the validated deployment configuration.
type Config struct {
	HTTPAddr        string
	LogLevel        slog.Level
	LogSource       bool
	ShutdownTimeout time.Duration
}

// Default returns the configuration used when no variable is set.
func Default() Config {
	return Config{HTTPAddr: ":8080", LogLevel: slog.LevelInfo, ShutdownTimeout: 8 * time.Second}
}

// Problem is one refused variable.
type Problem struct {
	Variable string
	Value    string
	Reason   string
}

// String renders the problem as NAME="value": reason, with the value quoted
// (control characters escaped) and truncated. A name that is not plain
// printable text (possible for an unknown variable) is quoted the same way.
func (p Problem) String() string { return name(p.Variable) + "=" + quote(p.Value) + ": " + p.Reason }

// name returns v unchanged when quoting would only add the quotes.
func name(v string) string {
	if q := strconv.Quote(v); len(v) <= maxQuoted && q[1:len(q)-1] == v {
		return v
	}
	return quote(v)
}

// Error lists every refused variable.
type Error struct {
	Problems []Problem
}

func (e *Error) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.String()
	}
	return "invalid configuration: " + strings.Join(parts, "; ")
}

// Load reads the configuration from environ, "KEY=value" entries as
// os.Environ returns them. It validates every variable before returning and
// reports every problem at once.
func Load(environ []string) (Config, error) {
	env := map[string]string{}
	for _, entry := range environ {
		k, v, ok := strings.Cut(entry, "=")
		if ok && (strings.HasPrefix(k, Prefix) || k == varPort) {
			env[k] = v
		}
	}
	cfg := Default()
	var problems []Problem
	refuse := func(name, reason string) {
		problems = append(problems, Problem{Variable: name, Value: env[name], Reason: reason})
	}

	var unknown []string
	for k := range env {
		if strings.HasPrefix(k, Prefix) && !known[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		refuse(k, "unknown variable")
	}

	addr, hasAddr := env[varHTTPAddr]
	port, hasPort := env[varPort]
	switch {
	case hasAddr && hasPort:
		refuse(varHTTPAddr, "conflicts with PORT; set only one of them")
	case hasAddr:
		if reason := checkAddr(addr); reason != "" {
			refuse(varHTTPAddr, reason)
		} else {
			cfg.HTTPAddr = addr
		}
	case hasPort:
		if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
			refuse(varPort, "must be a number from 1 to 65535")
		} else {
			cfg.HTTPAddr = ":" + port
		}
	}

	if v, ok := env[varLogLevel]; ok {
		if level, ok := levels[v]; ok {
			cfg.LogLevel = level
		} else {
			refuse(varLogLevel, "must be one of debug, info, warn, error")
		}
	}

	if v, ok := env[varLogSource]; ok {
		switch v {
		case "true":
			cfg.LogSource = true
		case "false":
			cfg.LogSource = false
		default:
			refuse(varLogSource, "must be true or false")
		}
	}

	if v, ok := env[varShutdownTimeout]; ok {
		d, err := time.ParseDuration(v)
		switch {
		case err != nil:
			refuse(varShutdownTimeout, "must be a duration such as 8s")
		case d < minShutdownTimeout || d > maxShutdownTimeout:
			refuse(varShutdownTimeout, fmt.Sprintf("must be between %s and %s", minShutdownTimeout, maxShutdownTimeout))
		default:
			cfg.ShutdownTimeout = d
		}
	}

	if len(problems) > 0 {
		return Config{}, &Error{Problems: problems}
	}
	return cfg, nil
}

// checkAddr returns why addr is not an acceptable listening address, or "".
func checkAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "must be host:port, such as :8080"
	}
	if host != "" && host != "localhost" && net.ParseIP(host) == nil {
		return "the host must be empty, localhost or an IP address"
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "the port must be a number from 0 to 65535"
	}
	return ""
}

// quote renders v for an error message: quoted with control characters
// escaped, and cut at maxQuoted bytes on a rune boundary.
func quote(v string) string {
	if len(v) <= maxQuoted {
		return strconv.Quote(v)
	}
	cut := maxQuoted
	for cut > 0 && !utf8.RuneStart(v[cut]) {
		cut--
	}
	return strconv.Quote(v[:cut]) + "..."
}
