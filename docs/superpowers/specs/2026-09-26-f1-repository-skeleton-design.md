# F1 — Repository skeleton, modules, pipeline and build identifier: Design

> **Status:** spec for delivery F1 of `docs/roadmap.md`, written through the brainstorming process on
> 2026-09-26. Each section below was approved by the owner in the session; the written spec awaits the
> owner's review.
>
> **Sources:** `docs/requirements.md` (§N), `docs/design.md` ("design §N"), `docs/threat-model.md`
> (threat identifiers), `docs/roadmap.md` (F1 and its appendix), `CLAUDE.md`.

---

## 1. Goal

Every later pull request is checked by a pipeline that runs the full command list of `CLAUDE.md`; the
module boundaries are enforced from the first line of code; and the binary starts, announces which
build it is, and answers a health check.

## 2. Decisions taken in the brainstorming session

| # | Decision | Alternative rejected |
|---|---|---|
| 1 | **One required status check per CI job**, named exactly as in section 6 | A single aggregating job: the ruleset would never change, but what blocks would be hidden in YAML and a mistake in the aggregator could let a failed job pass |
| 2 | **The `integration` job exists from F1**, with a PostgreSQL 16 service container, even though it has no tests until F4 | Adding it in F4, which would need another manual change to the ruleset |
| 3 | **Deployment configuration from `IDENTITY_*` environment variables only** in F1; file-based values (`IDENTITY_X_FILE`) arrive with the first secret (F4) | Environment plus a YAML file now: another parser (X-03) with no use yet |
| 4 | **GitHub code scanning in F1**: a CodeQL workflow, and SARIF from `gosec` and Trivy uploaded to the Security tab | Blocking only through job logs, deferring code scanning |
| 5 | **Trivy blocks on HIGH and CRITICAL, with or without a fix available**; MEDIUM and LOW are reported through SARIF without blocking | Blocking every severity (noise); ignoring unfixed findings (hides the cases without a way out) |
| 6 | **Individual Trivy exceptions are allowed only under strict rules** (section 6.4) | No exceptions at all, which could stop every pull request indefinitely |
| 7 | **Images built with `ko`**, reproducible by default, base image pinned by digest | A Dockerfile with `docker buildx` (cannot build in a session, reproducibility by hand); building only in F3 |
| 8 | **Tools in a separate `tools/` module outside the workspace** | Tools in the workspace, which would raise shared dependency versions (for example `golang.org/x/crypto`) for `core` during workspace builds |
| 9 | **`math/rand` forbidden in all non-test code**, not only in "security packages" | A per-package list, fragile in a product where almost every package is security code |
| 10 | **`zizmor` added to `make check`** alongside `actionlint`, as the workflow security linter B7-02 asks for | `actionlint` alone, which checks syntax, not security |

## 3. Environment facts this design depends on

Verified in the session on 2026-09-26; `docs/roadmap.md`'s appendix is updated with them.

| Fact | Value | Consequence |
|---|---|---|
| Go on the path | `go1.24.7`, `GOTOOLCHAIN=auto` | `go.work` and `go.mod` name `toolchain go1.27.1`, fetched from `proxy.golang.org` |
| Latest Go release | `go1.27.1` (from `proxy.golang.org/golang.org/toolchain/@v/list`) | Pinned; Dependabot does not bump the `toolchain` line, so a toolchain bump is a manual pull request when `govulncheck` reports a standard-library fix |
| Chromium | Revision 1194 (Chromium 141.0.7390.37) in `/opt/pw-browsers` | Playwright for Python **1.56.0**, whose `browsers.json` names revision 1194 (1.57.0 names 1200) |
| PostgreSQL | 16.13 in `/usr/lib/postgresql/16` | Used by F4's harness; F1's `make integration` needs none |
| Docker | Installed, daemon not running | Images are built by `ko` to a tarball; Trivy and CodeQL run only in CI |
| Network | `go.dev` and `github.com` release downloads refused by the session's proxy; `proxy.golang.org`, `pypi.org`, `gcr.io` and `ghcr.io` reachable | Every tool the session runs comes through the Go proxy or PyPI; Trivy is CI-only |

## 4. Modules, files and tools

### 4.1 The workspace

`go.work` at the root declares `go 1.27.0` and `toolchain go1.27.1` and uses seven modules. Every module
declares `go 1.27.0` and carries its own `LICENSE`.

| Module | Path | Licence | Contents in F1 |
|---|---|---|---|
| `github.com/aleogr/identity` | `/` | AGPL-3.0-only | `cmd/identity` (`main`), `internal/config`, `internal/logging`, `internal/server`, `internal/buildinfo` |
| `.../identity/spec` | `spec/` | Apache-2.0 | `doc.go` |
| `.../identity/core` | `core/` | Apache-2.0 | `doc.go`; `core/secret` |
| `.../identity/adapters/postgres` | `adapters/postgres/` | Apache-2.0 | `doc.go` |
| `.../identity/adapters/gcp` | `adapters/gcp/` | Apache-2.0 | `doc.go` |
| `.../identity/adapters/mail` | `adapters/mail/` | Apache-2.0 | `doc.go` |
| `.../identity/conformance` | `conformance/` | Apache-2.0 | `doc.go` |

The root module's `LICENSE` is the existing repository `LICENSE`. Each Apache module's `LICENSE` is a
copy of `LICENSES/Apache-2.0.txt`.

### 4.2 `core/secret`

`secret.String` wraps a secret value. Every formatting path redacts it: `String`, `GoString`, `Format`
(every verb), `slog.LogValuer`, `json.Marshaler` and `encoding.TextMarshaler` all produce `[REDACTED]`.
The value is reachable only through an explicitly named method (`Reveal`). It lives in `core` because
the whole domain will carry secrets, and it performs no I/O. It is the typed secret wrapper of B4-04.

### 4.3 The tools module

`tools/` is a module of its own (`github.com/aleogr/identity/tools`, AGPL-3.0-only by the default rule
of `LICENSING.md`, with its `LICENSE`), **not listed in `go.work`**, and always invoked as
`GOWORK=off go tool -modfile=tools/go.mod <tool>`. It holds, as `tool` directives:

- `golangci-lint` v2 (which includes `depguard`), `staticcheck`, `gosec`, `govulncheck`, `gitleaks`,
  `actionlint`, `ko`;
- two checkers written for this project, with their tests: `tools/cmd/spdxcheck` and
  `tools/cmd/trivyignorecheck`.

**Stated departure:** golangci-lint's documentation recommends its released binary over building it
from source. The project's decision (`CLAUDE.md`) is `tool` directives; the separate module contains the
risk that decision carries, since the tools' dependencies never enter `core`'s module graph.

`zizmor` is a Python tool. It runs from a virtual environment the `Makefile` creates on demand in
`tools/.venv` from `tools/requirements.txt`, pinned with hashes and installed with `--require-hashes`.

### 4.4 Licence identifiers

Every `.go`, `.py` and `.sh` file and every `Makefile` starts with an SPDX identifier (after a shebang
or a `//go:build` line, where one exists), and the identifier matches the directory:
`Apache-2.0` under `spec/`, `core/`, `adapters/` and `conformance/`; `AGPL-3.0-only` everywhere else.
Markdown, YAML, licence texts and generated files are exempt. `spdxcheck` enforces this and names every
offending file.

### 4.5 Dependabot

`.github/dependabot.yml` gains the `gomod` ecosystem for every module and for `tools/`, and the `pip`
ecosystem for `e2e/` and `tools/`. `playwright` is ignored, with a comment: its version is bound to the
Chromium revision pre-installed in the session (section 3), and moves only by a deliberate pull request
when that revision changes.

## 5. Lint rules and the direction of dependencies

One `.golangci.yml` at the root, run by the `Makefile` once per module, because golangci-lint analyses
one module at a time.

### 5.1 `depguard`

| Files | May import | Denied |
|---|---|---|
| `spec/**` | The standard library, `spec` | Anything else |
| `core/**` | The standard library, `golang.org/x/crypto`, `spec`, `core` | Anything else: adapters, the root module, cloud SDKs. The allowlist of pure libraries starts empty and grows only through a pull request |
| `adapters/<name>/**` | Anything else, including `core` and `spec` | The root module's packages; any other adapter (so the GCP libraries never reach a PostgreSQL user) |
| `conformance/**` | The standard library, `spec` | `core`, adapters, the root module: it is a black-box suite over HTTP |
| Every Apache-2.0 module | — | The root module's packages (`LICENSING.md`: Apache code never imports AGPL code) |
| Every non-test file | — | `math/rand`, `math/rand/v2` (X-02); `crypto/rand` is the only source of randomness |

### 5.2 Other linters

Beyond golangci-lint's standard set: `gosec`, `errorlint`, `bodyclose`, `noctx`, `contextcheck`,
`nilerr`, `forcetypeassert`, `gocritic`, `revive`. `staticcheck` also runs standalone, as `CLAUDE.md`
lists it.

### 5.3 `make check`

In order, each across every module where it applies: `go vet`; `staticcheck`; `golangci-lint`; `gosec`
(also writing SARIF for code scanning); `govulncheck`; `spdxcheck`; `trivyignorecheck`; `gitleaks` over
the git history; `actionlint` and `zizmor` over `.github/workflows/`. `gitleaks`, `actionlint` and
`zizmor` run in `make check`, and not only in CI, so a session catches their findings before a push.

## 6. The pipeline

### 6.1 `ci.yml`

Runs on `pull_request` and on `push` to `main`. Superseded runs of the same pull request are cancelled.
Every job has its own `timeout-minutes`. The job names below are the exact required status checks.

| Job | What it does |
|---|---|
| `check` | `make check`, with the full history checked out (`fetch-depth: 0`) for `gitleaks`; uploads `gosec`'s SARIF |
| `test` | `make test`: `go test -race -cover` across every workspace module and `tools/` |
| `integration` | `make integration` with a `postgres:16` service container pinned by digest and `TEST_DATABASE_URL` set. **It has no tests until F4**; the pull request says so |
| `e2e` | Installs Chromium on the runner, runs `make e2e`, uploads logs and screenshots as artefacts |
| `image` | `ko build` to a tarball (no push); Trivy scans the tarball, **failing on HIGH and CRITICAL**, honouring `.trivyignore.yaml`; uploads the full SARIF |

### 6.2 `codeql.yml`

One job, `codeql`, analysing Go with a manual build of every workspace module. Runs on `pull_request`,
on `push` to `main` and weekly.

### 6.3 Workflow security (B7-02)

- `permissions: {}` at the top of every workflow; each job gets `contents: read`, and
  `security-events: write` only in the three jobs that upload SARIF (`check`, `image`, `codeql`).
- Every action pinned by commit SHA with its version in a comment; Dependabot keeps them current. The
  actions used: `actions/checkout`, `actions/setup-go`, `actions/setup-python`, `actions/upload-artifact`,
  `github/codeql-action`, `aquasecurity/setup-trivy` (with the Trivy version pinned).
- `persist-credentials: false` on every checkout; no secrets used; no `pull_request_target`.
- `actionlint` and `zizmor` in `make check` (section 5.3).

### 6.4 Trivy exceptions

An entry in `.trivyignore.yaml` uses Trivy's own fields: `id`, the finding's identifier; `statement`,
a written justification (for example, that the vulnerable code is not reachable, with `govulncheck`'s
evidence); and `expired_at`, the expiry date. `trivyignorecheck` fails `make check` when an entry lacks
the statement or the date, when the date has passed, or when the date is more than 90 days after the
day the check runs. The last rule is what bounds an exception to 90 days: when an entry is added, its
date can be at most 90 days ahead, and nothing extends it without a new pull request. Once the date
passes, Trivy itself stops suppressing the finding. An exception enters only through a pull request the owner approves; Claude Code
never adds one to unblock a check without asking first.

### 6.5 The required checks

The `protect-main` ruleset requires: **`check`**, **`test`**, **`integration`**, **`e2e`**,
**`image`**, **`codeql`**. The owner adds them by hand after the pull request has run them once
(section 10).

## 7. The binary

### 7.1 Commands

The command line uses only the standard library.

- `identity serve` — starts the server.
- `identity version` — prints the build identifier.
- Any other command exits with status 2 and prints the usage.

### 7.2 Configuration

Every variable is read and validated **before** any effect. All errors are reported together, each
naming the variable and quoting the value received.

| Variable | Type | Default | Limits |
|---|---|---|---|
| `IDENTITY_HTTP_ADDR` | `host:port` | `:8080`; `:$PORT` when Cloud Run's `PORT` is set | Both `IDENTITY_HTTP_ADDR` and `PORT` set is refused as ambiguous |
| `IDENTITY_LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` | — |
| `IDENTITY_LOG_SOURCE` | Boolean | `false` | Exactly `true` or `false`; the `1`, `t` and `T` that `strconv.ParseBool` accepts are refused |
| `IDENTITY_SHUTDOWN_TIMEOUT` | Go duration | `8s` | From `1s` to `60s`; Cloud Run allows 10 seconds after `SIGTERM` |

- Any other `IDENTITY_*` variable is refused as unknown.
- A configuration error exits with status **2**; a run-time failure with status **1**.
- When secret variables arrive, their values are never quoted in an error.

### 7.3 Logging

`log/slog` with a JSON handler on standard output, using Cloud Logging's field names:

- `severity`, with the levels `DEBUG`, `INFO`, `WARNING` and `ERROR`;
- `message`;
- `time` in RFC 3339 with nanoseconds;
- `logging.googleapis.com/sourceLocation` (`file`, `line`, `function`) when `IDENTITY_LOG_SOURCE=true`.

Values of type `secret.String` are logged as `[REDACTED]` (B4-04).

### 7.4 Build identifier

`git describe --tags --always --dirty --abbrev=12`, injected with `-ldflags -X` by `make build` and by
`ko`, through `.ko.yaml`. A build without the flag (for example `go run`) falls back to the VCS revision
Go embeds, prefixed `devel`. The start-up log line carries the build identifier, the Go version and the
listening address.

### 7.5 HTTP

- `/health` answers `GET` and `HEAD` with `200` and `{"status":"ok"}`, with `Content-Type:
  application/json`, `Cache-Control: no-store` and `X-Content-Type-Options: nosniff`. Other methods get
  `405` with an `Allow` header; other paths get `404`.
- The build identifier is **not** on `/health`; the operator-only version endpoint is F3's.
- Explicit limits: `ReadHeaderTimeout` 10 s, `ReadTimeout` 30 s, `WriteTimeout` 30 s, `IdleTimeout`
  120 s, `MaxHeaderBytes` 64 KiB.

### 7.6 Shutdown

On `SIGTERM` or `SIGINT` the server stops accepting connections, lets requests in flight finish within
`IDENTITY_SHUTDOWN_TIMEOUT`, logs `stopped` and exits with status 0. If the timeout is exceeded, it
exits with status 1.

## 8. The `Makefile`

| Target | What it does |
|---|---|
| `check` | Section 5.3 |
| `test` | `go test -race -cover` across the workspace modules and `tools/` |
| `integration` | `go test -race -tags integration` across the workspace. In F1 it needs no database; F4 adds the local PostgreSQL 16 cluster started when `TEST_DATABASE_URL` is unset |
| `e2e` | Builds the binary, then runs the suite with `e2e/.venv/bin/python -m pytest` |
| `e2e-deps` | Creates `e2e/.venv` from `e2e/requirements.txt` (hash-pinned, Playwright 1.56.0). In a session it uses the Chromium in `/opt/pw-browsers`; in CI the workflow installs Chromium |
| `build` | Builds `bin/identity` with the build identifier |
| `run` | Builds and runs `identity serve` |
| `image` | `ko build` to a tarball in `dist/` |

`terraform-deps` and `tf` are F2's.

## 9. Testing and verification

### 9.1 Unit tests (test-driven, `make test`)

- **Configuration:** a table covering an unknown variable, a boolean with a typo, duration bounds, the
  `PORT` conflict, and several errors reported together; a **fuzz test** of the loader, since it is a
  parser (X-03).
- **Logging:** field names and the severity mapping.
- **Secrets:** `secret.String` redacted in every formatting path.
- **Build identifier:** the fallback when no flag was injected.
- **HTTP:** `/health` methods, status codes and headers.
- **Shutdown:** a request in flight completes; an exceeded timeout becomes an error.
- **Checkers:** `spdxcheck` and `trivyignorecheck`.

### 9.2 Process tests (Go, in `cmd/identity`)

They build the binary once and check that:

- `IDENTITY_LOG_SOURCE=ture` exits with status 2, and the message names the variable and quotes the
  value (the roadmap's required test);
- `SIGTERM` exits with status 0 and logs `stopped`;
- the start-up log carries the build identifier.

### 9.3 End-to-end tests (pytest with Playwright 1.56.0)

Against the binary from `make build`, on a free port:

- Chromium opens `/health` and receives `200` with the expected JSON;
- `/health` answers within 1 second of the process starting (§18's cold-start target);
- the build identifier in the start-up log equals `git describe`'s output.

### 9.4 Evidence for the pull request

- The output of `make check`, `make test`, `make integration` and `make e2e` run in the session.
- The pipeline green on the pull request.
- A temporary commit in which `core` imports `adapters/postgres`, with `check` failing on `depguard`'s
  message, then reverted.
- A statement that Trivy and CodeQL run only in CI.
- The `identity-security-review` skill run before the pull request opens, since F1 adds a
  configuration parser and the secret type.

## 10. Manual steps for the owner

Given one at a time after the pull request's checks have run once, so that GitHub offers their names:

1. Add the required status checks `check`, `test`, `integration`, `e2e`, `image` and `codeql` to the
   `protect-main` ruleset.
2. Confirm code scanning shows the CodeQL, `gosec` and Trivy results, and that GitHub's *default*
   CodeQL setup is off, since the workflow is the *advanced* setup and the two cannot coexist.

Each step, once done, is recorded in `docs/infrastructure.md`.

## 11. Documents updated in the same pull request

- `docs/roadmap.md`: F1's scope gains CodeQL, `zizmor`, `ko`, the `integration` job and the `tools/`
  module; F8's scope and threats gain the constant-time comparison lint rule (X-01); the appendix
  records section 3's facts.
- `docs/design.md`, section 1.2: the `tools/` module.
- `docs/threat-model.md`:
  - B7-01 — Trivy's threshold and the exception policy;
  - B7-02 — `actionlint` and `zizmor` as the verification;
  - B4-04 — `secret.String`;
  - X-05 — the refuse-to-start test.
- `CLAUDE.md`: the actual contents of `make check` and the `e2e` virtual environment.
- `docs/claude-code-environment.md`: `e2e-deps` now exists.
- `.github/dependabot.yml`: the new ecosystems.
- `docs/infrastructure.md`: the manual steps, once performed.

## 12. Not in this delivery

- Any GCP resource or database (F2 to F4).
- The operator-only version endpoint (F3).
- Security headers beyond `/health`'s, CSP, CSRF (F6).
- Releases with GoReleaser, signing, SBOM and the digest check (the first release).
- **The constant-time comparison lint rule (X-01, §17)**: no delivery of the roadmap assigned it; the
  owner assigned it to **F8**, with the keys, and this pull request records that in `docs/roadmap.md`.
