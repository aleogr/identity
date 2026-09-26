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

A session has neither Playwright for Python nor `terraform`. The `Makefile` targets that install them
are delivered in F1 and F2 (`docs/roadmap.md`); the pinned Playwright release is the one built against
the Chromium revision pre-installed in `/opt/pw-browsers` (1194 on 2026-09-26), so no browser is
downloaded.

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
