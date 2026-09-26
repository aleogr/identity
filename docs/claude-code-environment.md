# Claude Code Environment

Reference for the configuration of the **identity** cloud environment in
[claude.ai/code](https://claude.ai/code).

## Environment setup script

It lives in the environment settings on claude.ai/code, **not in the repository**. It runs at the start
of every cloud session:

```bash
#!/bin/bash
LOG=/var/log/setup-environment.log
REPO=/home/user/identity

# Tool versions live in the repository (e2e/requirements.txt,
# infra/terraform/.terraform-version), never here: nothing in CI can see this
# file, so a version repeated here would drift.
if [ -f "$REPO/Makefile" ]; then
  make -C "$REPO" e2e-deps >> "$LOG" 2>&1 \
    || echo "FAILURE: e2e dependencies not installed" >> "$LOG"
  make -C "$REPO" terraform-deps >> "$LOG" 2>&1 \
    || echo "FAILURE: terraform not installed" >> "$LOG"
else
  echo "NOTE: $REPO/Makefile absent at setup time; run 'make e2e-deps terraform-deps' in the session once it exists" >> "$LOG"
fi

claude plugin marketplace add anthropics/claude-plugins-official >> "$LOG" 2>&1 \
  || echo "FAILURE: official marketplace not added" >> "$LOG"

claude plugin install superpowers@claude-plugins-official >> "$LOG" 2>&1 \
  || echo "FAILURE: superpowers not installed" >> "$LOG"

exit 0
```

All output goes to `/var/log/setup-environment.log`; installation failures are lines starting with
`FAILURE`.

### Why the guard tests the Makefile, not the directory

The marketplace's script tests whether the repository directory exists. Here the repository existed
long before it had a `Makefile`, and testing the directory would have written two false `FAILURE`
lines into every session's log until F1 — and a log that always reports failure stops being read.
Testing the `Makefile` makes the script correct before and after F1 without editing it.

### Why Superpowers is installed by the script

Plugins enabled only through a repository's `.claude/settings.json` are not installed in cloud
sessions. The script installs the plugin directly. It pins no version: the installed version is the
latest in the marketplace when the session starts (6.4.1 on 2026-09-25).

### Why the end-to-end and Terraform dependencies are installed by the script

A session has neither Playwright for Python nor `terraform`. `make e2e-deps` (F1) creates `e2e/.venv`
with the pinned Playwright release, the one built against the Chromium revision pre-installed in
`/opt/pw-browsers` (1194 on 2026-09-26, Playwright 1.56.0), so no browser is downloaded;
`make terraform-deps` arrives with F2.

## Network

Set on 2026-09-26 in the environment's settings (**Network access**): level **Custom**, with the
default list of common package managers included (it covers `proxy.golang.org` and `pypi.org`), and
these allowed domains:

| Domain | Needed by |
|---|---|
| `vuln.go.dev` | `govulncheck`, in `make check` |
| `github.com` | The setup script, which clones the plugin marketplace |
| `gcr.io`, `storage.googleapis.com` | `make image`, which pulls the distroless base image |

No target needs `go.dev` or GitHub release downloads: Go tools come through `proxy.golang.org` and
Python packages through `pypi.org`. Changes apply to new sessions only.

## Skills in `.claude/skills/`

| Skill | Origin |
|---|---|
| `webapp-testing` | Copied from the marketplace repository, which copied it from [anthropics/skills](https://github.com/anthropics/skills), commit `34040c9` |
| `frontend-design` | Same origin |
| `identity-security-review` | Written for this project: the domain checklist for pull requests |
| `spec-catalog` | Written for this project: how to find and cite the standards the product implements |

A `conformance-run` skill, describing how to read the OpenID Foundation suite's reports from CI,
is added by F16, when there is something to run.

The `skill-creator` skill is synced from the claude.ai account and appears in sessions as
`anthropic-skills:skill-creator`; it is not kept in the repository.
