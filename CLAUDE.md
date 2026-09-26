# Working agreements for Claude Code

These rules restate section 3 of `docs/requirements.md` and the agreements made in the inaugural
design session. They apply to every session in this repository.

## Language

- Talk to the owner in **Portuguese**.
- Everything versioned is written in **English**: code, comments, commit messages, pull request titles
  and descriptions, documentation. The only exception is translation files for other languages.

## Sources of truth

- `docs/requirements.md` decides what the product is. Do not reopen a decision recorded there; if you
  find a real contradiction, ask the owner.
- `docs/design.md` decides how it is built. `docs/research/` explains why; consult it, do not redo it.
- `docs/threat-model.md` names the threats; `docs/roadmap.md` holds the deliveries of the phase under
  way.
- When a session changes a requirement or a design decision, update the document in the same pull
  request.

## Methodology

- The **Superpowers** plugin is the method. Every roadmap delivery goes through: a spec in
  `docs/superpowers/specs/YYYY-MM-DD-fN-<topic>-design.md`, approved by the owner; a plan in
  `docs/superpowers/plans/`, approved by the owner, who chooses how it is executed; implementation
  with test-driven development; verification before completion; a pull request.
- **One topic at a time.** Before writing a document or a spec, ask the owner the questions whose
  answers change it, one at a time. Do not presume.
- The environment's setup script installs the plugin in every session
  (`docs/claude-code-environment.md`).

## Branches and pull requests

- Each topic is developed in its own branch and merged through a pull request, always opened by Claude
  Code, never directly on `main`.
- Never push to a branch other than the one designated for the session without explicit permission.
- **Claude Code watches every pull request it opens** until it is merged or closed, and drives it to a
  green, mergeable state: the owner should never have to report a red check.
- Diagnose a failing check from **that job's log**, not from a guess. Reproduce the failure where the
  session can, fix it, prove the same check passes, and only then push.
- **Never buy a green check.** No test is skipped, disabled or quarantined; no job is made
  non-blocking; no severity threshold is lowered; no commit bypasses the checks. If a check cannot run
  in a session (no Docker daemon, no GitHub Actions), say so in the pull request.
- Stop and ask the owner when the cause is ambiguous, when the fix would change a design decision, or
  when it would widen the pull request beyond its topic.
- Claude Code **never approves and never merges.** The owner merges.

## Working with the owner

- Manual steps outside a session (GitHub, GCP Cloud Shell, Cloudflare, providers) are given **one at a
  time**, as command blocks or clicks: the owner executes, reports the result, then receives the next.
- When the owner asks to see something before an action, show it and wait for explicit confirmation.
- External dependencies the owner must obtain are recorded as dependencies of the phase that needs
  them, with the recommended moment; they do not block design work.

## Quality

- Follow the domain's consolidated practice and its RFCs; when a choice departs from them, say so and
  why.
- Definition of done: every user-facing text in **both en-US and pt-BR** through translation keys,
  tests pass, and there is verification evidence (test output, screenshots or equivalent).
- UI principles: immediate visual feedback when an element is pressed; respect the reduced-motion
  preference; animations only when they serve a purpose.
- Tests use fake implementations of external providers, never real services.

## Security

- The repository is public. **No secret may ever be committed.**
- Model identifiers appear only in the attribution trailer of commit messages and pull request
  descriptions; never in code, documentation, titles or comments.
- **Security defaults are floors** (`docs/requirements.md`, section 15.2). Never add a way to relax a
  setting below its floor.
- Cryptography only from the Go standard library and `golang.org/x/crypto`; secrets compared in
  constant time; the refused algorithms of section 17 stay refused.
- The `core` module performs no I/O and imports no adapter.
- A change that adds a trust boundary, a parser, an outbound call, a secret or an extension point
  updates `docs/threat-model.md` in the same pull request. Run the `identity-security-review` skill on
  every pull request that touches authentication, tokens, keys, parsers or tenancy.
- Every outbound HTTP call goes through the SSRF-safe client (threat B3-01).

## Commands

Once the code exists, the checks CI runs are the ones to run before pushing; keep this list current:

```
make check        # go vet, staticcheck, golangci-lint (with depguard), gosec, govulncheck, licence
                  # identifiers, Trivy exceptions, gitleaks over the history, actionlint and zizmor
make test         # go test ./... -race -cover, across the workspace and tools/checks
make integration  # tests that need a real PostgreSQL, behind the `integration` tag
make e2e          # builds the binary and runs the end-to-end suite against it
make image        # builds the container image with ko into dist/ (Trivy scans it in CI only)
make tf           # terraform fmt -check, init -backend=false and validate (from F2)
```

The tools are pinned as `tool` directives, one module per tool under `tools/`, outside the workspace;
`make tools` builds them into `bin/tools/`. `make integration` and `make e2e` start a PostgreSQL 16
cluster of their own when `TEST_DATABASE_URL` is unset (from F4), because the Docker daemon does not run
in a session. The end-to-end suite runs from its own virtual environment, `e2e/.venv`, created by
`make e2e-deps`, never with the `pytest` on the path. `govulncheck` needs `vuln.go.dev`, allowed in the
environment's network settings. The OpenID Foundation's conformance suite, Trivy and CodeQL run only in
CI, with Docker. Terraform is only ever applied by CI.
