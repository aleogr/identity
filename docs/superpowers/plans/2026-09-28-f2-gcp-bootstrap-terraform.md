# F2 — GCP bootstrap and the Terraform foundation: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Terraform's state lives in the lab project, and CI plans on every pull request with a read-only identity and applies on every merge into `main` with an identity that cannot widen its own power — with no stored key.

**Architecture:** The bootstrap (state bucket, the two CI service accounts, Workload Identity Federation) is created by hand with `gcloud` and recorded; it stays outside Terraform. One Terraform root in `infra/terraform/`, parameterised by environment (`lab.tfvars`, `lab.gcs.tfbackend`), declares the APIs and the three service accounts of design §12.1 and is tested with `terraform test` against a mocked provider. A verified installer puts the pinned Terraform in `bin/tools/`. `.github/workflows/terraform.yml` has three jobs: `terraform` (plan, required check), `terraform-apply` (push to `main`) and `terraform-drift` (weekly).

**Tech Stack:** Terraform 1.16.4, `hashicorp/google` 8.4.0, `terraform test` with `mock_provider`, bash + GnuPG (`gpgv`) for the installer, GitHub Actions with `google-github-actions/auth` v3.0.0, `gcloud` in Cloud Shell for the bootstrap.

**Spec:** `docs/superpowers/specs/2026-09-27-f2-gcp-bootstrap-terraform-design.md` — read it before any task.

## Global Constraints

- Everything versioned is in English. No model identifier anywhere except commit trailers and the pull request's attribution line.
- Every `.sh` file and every `Makefile` starts with an SPDX identifier (after a shebang): `# SPDX-License-Identifier: AGPL-3.0-only` (nothing in this plan is under `spec/`, `core/`, `adapters/` or `conformance/`). `.tf` and `.tftest.hcl` files carry the same line as their first comment, for consistency, although `spdxcheck` does not require it.
- Project `aleogr-identity-lab-6f2g`, number `894113236465`, region `us-central1`.
- Terraform `1.16.4` (`required_version = "~> 1.16.0"`); provider `hashicorp/google` `~> 8.4`, locked at `8.4.0`.
- GitHub `repository_id` `1387974672`, `repository_owner_id` `2933195` (read from `api.github.com/repos/aleogr/identity` on 2026-09-28).
- Tests never touch GCP: `terraform test` uses `mock_provider "google"`; the installer test uses a local fake release signed by a throwaway key.
- Never skip, disable or weaken a check to get green.
- Actions pinned by commit SHA with the version in a comment; `permissions: {}` at the top of every workflow; `persist-credentials: false` on every checkout.
- Every commit ends with the trailer:

  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_013mmUo2SsqLxE8Zbh71vJnN
  ```

- Commit on branch `claude/cool-fermi-517zqb`.

### Session-only workaround: providers without `registry.terraform.io`

Until the owner allows `registry.terraform.io` and a new session starts, `terraform init` cannot resolve providers in this session. For development **only**, point Terraform at a local filesystem mirror in the scratchpad, filled from `releases.hashicorp.com` and verified with HashiCorp's key. Nothing of this is committed; CI and later sessions use the registry. Set it up once (Task 2, Step 1) and export in every shell that runs Terraform:

```bash
export TF_CLI_CONFIG_FILE=$SCRATCH/tfrc
```

where `$SCRATCH` is this session's scratchpad directory. The pull request states that `make tf` ran in the session through this mirror.

## Review Focus

1. **An older Terraform already in `bin/tools/terraform`** (a version bump in `.terraform-version`) — the installer must replace it, not keep it because a binary exists. Test in Task 1.
2. **A `SHA256SUMS` file without a line for the platform's zip** — the installer must fail, not install an unchecked binary. Test in Task 1.
3. **A tampered download** — the installer must fail and leave no binary behind (no half-installed state). Test in Task 1.
4. **`SHA256SUMS` signed by a key other than HashiCorp's** — refused even though the signature is cryptographically valid. Test in Task 1.
5. **A version string that is not `X.Y.Z`** (e.g. a stray newline or `1.16`) — refused with status 2 before any download, since it is interpolated into URLs. Test in Task 1.

---

### Task 1: The verified Terraform installer and `make terraform-deps`

**Files:**
- Create: `scripts/install-terraform.sh`
- Create: `scripts/install-terraform_test.sh`
- Create: `infra/terraform/.terraform-version`
- Create: `infra/terraform/hashicorp.asc`
- Modify: `Makefile` (add `terraform-deps`; add the installer test to `test`)

**Interfaces:**
- Produces: `scripts/install-terraform.sh VERSION KEY_FILE DEST_DIR`, exit 0 on success (binary at `DEST_DIR/terraform`), 1 on any verification or download failure, 2 on bad arguments. Environment `TERRAFORM_RELEASES` overrides the base URL `https://releases.hashicorp.com/terraform` (used by the test with a `file://` URL).
- Produces: `make terraform-deps` → `bin/tools/terraform` at the version in `infra/terraform/.terraform-version`.

- [ ] **Step 1: Pin the version and commit HashiCorp's key**

```bash
mkdir -p infra/terraform scripts
printf '1.16.4\n' > infra/terraform/.terraform-version
curl -sSf https://www.hashicorp.com/.well-known/pgp-key.txt -o infra/terraform/hashicorp.asc
export GNUPGHOME=$(mktemp -d)
gpg --quiet --import infra/terraform/hashicorp.asc
gpg --fingerprint --with-colons | grep '^fpr' | head -1
```

Expected: the first fingerprint is `C874011F0AB405110D02105534365D9472D7468F` (HashiCorp's release key, as published at https://www.hashicorp.com/security). Stop if it is not.

- [ ] **Step 2: Write the failing test**

`scripts/install-terraform_test.sh`:

```bash
#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# Tests for install-terraform.sh against a fake release signed by a throwaway
# key; nothing here reaches the network.

set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
installer=$here/install-terraform.sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export GNUPGHOME=$work/gnupg
mkdir -m 700 "$GNUPGHOME"

failures=0
fail() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }

key() { # key NAME: a signing key, exported armoured to $work/NAME.asc
	gpg --batch --quiet --passphrase '' --quick-gen-key "$1@example.invalid" ed25519 sign never
	gpg --batch --armor --export "$1@example.invalid" > "$work/$1.asc"
}

release() { # release VERSION SIGNER: a fake release under $work/releases
	local v=$1 signer=$2 dir=$work/releases/$1 bin
	mkdir -p "$dir" "$work/build"
	bin=$work/build/terraform
	printf '#!/bin/sh\necho '"'"'{"terraform_version": "%s"}'"'"'\n' "$v" > "$bin"
	chmod +x "$bin"
	rm -f "$dir/terraform_${v}_linux_amd64.zip"
	(cd "$work/build" && zip -q "$dir/terraform_${v}_linux_amd64.zip" terraform)
	(cd "$dir" && sha256sum "terraform_${v}_linux_amd64.zip" > "terraform_${v}_SHA256SUMS")
	gpg --batch --yes --local-user "$signer@example.invalid" --detach-sign \
		-o "$dir/terraform_${v}_SHA256SUMS.sig" "$dir/terraform_${v}_SHA256SUMS"
}

run() { # run VERSION DEST: the installer against the fake releases
	TERRAFORM_RELEASES="file://$work/releases" "$installer" "$1" "$work/hashicorp.asc" "$2"
}

key hashicorp
key intruder
release 9.9.8 hashicorp
release 9.9.9 hashicorp

# A valid release installs, and the binary reports its version.
dest=$work/bin1
if ! run 9.9.9 "$dest" >/dev/null 2>&1; then fail "valid release refused"; fi
if ! "$dest/terraform" version -json 2>/dev/null | grep -q '"terraform_version": "9.9.9"'; then
	fail "installed binary does not report 9.9.9"
fi

# An installed binary at the pinned version is kept without downloading.
mv "$work/releases" "$work/releases.away"
if ! run 9.9.9 "$dest" >/dev/null 2>&1; then fail "reinstall of the same version downloaded again"; fi
mv "$work/releases.away" "$work/releases"

# Review focus 1: an older binary is replaced.
dest=$work/bin2
run 9.9.8 "$dest" >/dev/null 2>&1 || fail "9.9.8 not installed"
run 9.9.9 "$dest" >/dev/null 2>&1 || fail "upgrade to 9.9.9 refused"
"$dest/terraform" version -json | grep -q '"terraform_version": "9.9.9"' || fail "older binary kept"

# Review focus 3: a tampered zip is refused and leaves no binary.
dest=$work/bin3
printf 'x' >> "$work/releases/9.9.9/terraform_9.9.9_linux_amd64.zip"
if run 9.9.9 "$dest" >/dev/null 2>&1; then fail "tampered zip accepted"; fi
[[ -e $dest/terraform ]] && fail "tampered zip left a binary"
release 9.9.9 hashicorp

# Review focus 2: sums without the platform's line are refused.
dest=$work/bin4
: > "$work/releases/9.9.9/terraform_9.9.9_SHA256SUMS"
gpg --batch --yes --local-user hashicorp@example.invalid --detach-sign \
	-o "$work/releases/9.9.9/terraform_9.9.9_SHA256SUMS.sig" "$work/releases/9.9.9/terraform_9.9.9_SHA256SUMS"
if run 9.9.9 "$dest" >/dev/null 2>&1; then fail "sums without the zip's line accepted"; fi
[[ -e $dest/terraform ]] && fail "sums without the zip's line left a binary"
release 9.9.9 hashicorp

# Review focus 4: sums signed by another key are refused.
dest=$work/bin5
release 9.9.9 intruder
if run 9.9.9 "$dest" >/dev/null 2>&1; then fail "signature by another key accepted"; fi
[[ -e $dest/terraform ]] && fail "signature by another key left a binary"
release 9.9.9 hashicorp

# A missing signature is refused.
dest=$work/bin6
rm "$work/releases/9.9.9/terraform_9.9.9_SHA256SUMS.sig"
if run 9.9.9 "$dest" >/dev/null 2>&1; then fail "missing signature accepted"; fi
release 9.9.9 hashicorp

# Review focus 5: a malformed version is refused with status 2.
for v in 1.16 '9.9.9
' 'v9.9.9' '9.9.9/../x'; do
	status=0
	run "$v" "$work/bin7" >/dev/null 2>&1 || status=$?
	[[ $status -eq 2 ]] || fail "version '$v' gave status $status, want 2"
done

if [[ $failures -gt 0 ]]; then
	echo "install-terraform_test: $failures failure(s)" >&2
	exit 1
fi
echo "install-terraform_test: ok"
```

```bash
chmod +x scripts/install-terraform_test.sh
```

- [ ] **Step 3: Run the test to see it fail**

Run: `scripts/install-terraform_test.sh`
Expected: FAIL lines (the installer does not exist yet), exit status 1.

- [ ] **Step 4: Write the installer**

`scripts/install-terraform.sh`:

```bash
#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# Installs Terraform VERSION into DEST_DIR, verified against KEY_FILE
# (docs/superpowers/specs/2026-09-27-f2-gcp-bootstrap-terraform-design.md,
# section 5.6): the SHA256SUMS signature is checked with gpgv against that key
# alone, then the zip's SHA-256 against the signed sums. Nothing is installed
# unless both hold.
#
# Usage: install-terraform.sh VERSION KEY_FILE DEST_DIR
# TERRAFORM_RELEASES overrides the release base URL (tests only).

set -euo pipefail

if [[ $# -ne 3 ]]; then
	echo "usage: install-terraform.sh VERSION KEY_FILE DEST_DIR" >&2
	exit 2
fi
version=$1 key=$2 dest=$3
if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "install-terraform: invalid version $(printf '%q' "$version")" >&2
	exit 2
fi

if [[ -x $dest/terraform ]] &&
	"$dest/terraform" version -json 2>/dev/null | grep -q "\"terraform_version\": \"$version\""; then
	exit 0
fi

case "$(uname -m)" in
x86_64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*)
	echo "install-terraform: unsupported machine $(uname -m)" >&2
	exit 1
	;;
esac
if [[ $(uname -s) != Linux ]]; then
	echo "install-terraform: only Linux is supported (sessions and CI)" >&2
	exit 1
fi

base=${TERRAFORM_RELEASES:-https://releases.hashicorp.com/terraform}
zip=terraform_${version}_linux_${arch}.zip
sums=terraform_${version}_SHA256SUMS

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

for f in "$zip" "$sums" "$sums.sig"; do
	curl -fsSL --proto '=https,file' -o "$work/$f" "$base/$version/$f"
done

# A keyring holding only KEY_FILE: no other key the machine trusts counts.
mkdir -m 700 "$work/gnupg"
gpg --batch --quiet --homedir "$work/gnupg" --dearmor < "$key" > "$work/keyring.gpg"
if ! gpgv --quiet --keyring "$work/keyring.gpg" "$work/$sums.sig" "$work/$sums" 2>/dev/null; then
	echo "install-terraform: bad signature on $sums" >&2
	exit 1
fi

line=$(grep -E "^[0-9a-f]{64}  ${zip//./\\.}\$" "$work/$sums" || true)
if [[ -z $line ]]; then
	echo "install-terraform: $zip is not listed in $sums" >&2
	exit 1
fi
if ! (cd "$work" && printf '%s\n' "$line" | sha256sum --check --status); then
	echo "install-terraform: checksum mismatch for $zip" >&2
	exit 1
fi

mkdir -p "$work/out" "$dest"
unzip -q -o "$work/$zip" terraform -d "$work/out"
install -m 0755 "$work/out/terraform" "$dest/terraform.new"
mv -f "$dest/terraform.new" "$dest/terraform"
```

```bash
chmod +x scripts/install-terraform.sh
```

- [ ] **Step 5: Run the test to see it pass**

Run: `scripts/install-terraform_test.sh`
Expected: `install-terraform_test: ok`, exit 0.

- [ ] **Step 6: Add the Makefile targets**

In `Makefile`, after the `image` target, add:

```make
TF_DIR := infra/terraform
TERRAFORM := $(CURDIR)/$(T)/terraform

.PHONY: terraform-deps
terraform-deps: ## Install the pinned Terraform, verified against HashiCorp's signature
	scripts/install-terraform.sh "$$(cat $(TF_DIR)/.terraform-version)" $(TF_DIR)/hashicorp.asc $(CURDIR)/$(T)
```

and in the `test` target, as its last line:

```make
	scripts/install-terraform_test.sh
```

- [ ] **Step 7: Run the targets**

Run: `make terraform-deps && bin/tools/terraform version && make test 2>&1 | tail -3`
Expected: `Terraform v1.16.4`; the last line of `make test` is `install-terraform_test: ok`.

- [ ] **Step 8: Check the licence identifiers and commit**

Run: `make tools && bin/tools/spdxcheck`
Expected: no output, exit 0.

```bash
git add scripts/ infra/terraform/.terraform-version infra/terraform/hashicorp.asc Makefile
git commit -m "Install Terraform verified against HashiCorp's signature

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013mmUo2SsqLxE8Zbh71vJnN"
```

---

### Task 2: The Terraform root, its variables and `make tf`

**Files:**
- Create: `infra/terraform/versions.tf`, `backend.tf`, `providers.tf`, `variables.tf`, `lab.tfvars`, `lab.gcs.tfbackend`, `.terraform.lock.hcl`
- Create: `infra/terraform/tests/variables.tftest.hcl`
- Modify: `Makefile` (add `tf`)
- Modify: `.gitignore` (Terraform plan files)

**Interfaces:**
- Consumes: `make terraform-deps`, `$(TERRAFORM)`, `$(TF_DIR)` from Task 1.
- Produces: variables `project_id` (string), `region` (string), `environment` (string); `make tf`.

- [ ] **Step 1: Session only — the provider mirror**

```bash
SCRATCH=<this session's scratchpad directory>
P=8.4.0; PB=https://releases.hashicorp.com/terraform-provider-google/$P
M=$SCRATCH/mirror/registry.terraform.io/hashicorp/google
mkdir -p "$M" "$SCRATCH/sums"
for pl in linux_amd64 darwin_arm64; do curl -sSf -o "$M/terraform-provider-google_${P}_${pl}.zip" "$PB/terraform-provider-google_${P}_${pl}.zip"; done
curl -sSf -o "$SCRATCH/sums/SHA256SUMS" "$PB/terraform-provider-google_${P}_SHA256SUMS"
curl -sSf -o "$SCRATCH/sums/SHA256SUMS.sig" "$PB/terraform-provider-google_${P}_SHA256SUMS.sig"
gpg --batch --dearmor < infra/terraform/hashicorp.asc > "$SCRATCH/sums/keyring.gpg"
gpgv --keyring "$SCRATCH/sums/keyring.gpg" "$SCRATCH/sums/SHA256SUMS.sig" "$SCRATCH/sums/SHA256SUMS"
(cd "$M" && grep -E 'linux_amd64|darwin_arm64' "$SCRATCH/sums/SHA256SUMS" | sha256sum -c -)
printf 'provider_installation {\n  filesystem_mirror {\n    path = "%s/mirror"\n  }\n}\n' "$SCRATCH" > "$SCRATCH/tfrc"
export TF_CLI_CONFIG_FILE=$SCRATCH/tfrc
```

Expected: `Good signature`; both zips `OK`. Skip this step in a session where `registry.terraform.io` is allowed.

- [ ] **Step 2: Write the failing test**

`infra/terraform/tests/variables.tftest.hcl`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

# The variables refuse values outside the lab's decisions
# (docs/superpowers/specs/2026-09-27-f2-gcp-bootstrap-terraform-design.md,
# section 5.2). No test here reaches GCP.

mock_provider "google" {}

variables {
  project_id  = "aleogr-identity-lab-6f2g"
  region      = "us-central1"
  environment = "lab"
}

run "lab_values_are_accepted" {
  command = plan
}

run "a_malformed_project_id_is_refused" {
  command = plan
  variables {
    project_id = "Aleogr_Identity"
  }
  expect_failures = [var.project_id]
}

run "a_project_id_ending_in_a_hyphen_is_refused" {
  command = plan
  variables {
    project_id = "aleogr-identity-"
  }
  expect_failures = [var.project_id]
}

run "another_region_is_refused" {
  command = plan
  variables {
    region = "southamerica-east1"
  }
  expect_failures = [var.region]
}

run "another_environment_is_refused" {
  command = plan
  variables {
    environment = "prod"
  }
  expect_failures = [var.environment]
}
```

- [ ] **Step 3: Write the root's fixed files**

`infra/terraform/versions.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

terraform {
  required_version = "~> 1.16.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.4"
    }
  }
}
```

`infra/terraform/backend.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

# Partial configuration: the bucket and prefix come from <environment>.gcs.tfbackend,
# passed with -backend-config. The bucket is created by hand, outside Terraform
# (docs/infrastructure.md).
terraform {
  backend "gcs" {}
}
```

`infra/terraform/lab.gcs.tfbackend`:

```hcl
bucket = "aleogr-identity-lab-6f2g-tfstate"
prefix = "lab"
```

`infra/terraform/lab.tfvars`:

```hcl
project_id  = "aleogr-identity-lab-6f2g"
region      = "us-central1"
environment = "lab"
```

`infra/terraform/providers.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

provider "google" {
  project = var.project_id
  region  = var.region

  default_labels = {
    app        = "identity"
    env        = var.environment
    managed-by = "terraform"
  }
}
```

`infra/terraform/variables.tf` (empty for now so the test fails on the missing variables):

```hcl
# SPDX-License-Identifier: AGPL-3.0-only
```

- [ ] **Step 4: Generate the lock file**

```bash
cd infra/terraform
"$OLDPWD/bin/tools/terraform" providers lock -fs-mirror="$SCRATCH/mirror" -platform=linux_amd64 -platform=darwin_arm64
```

(In a session with the registry allowed: `terraform providers lock -platform=linux_amd64 -platform=darwin_arm64`, which also records the `zh:` hashes; skip the next paragraph.)

Through the mirror, Terraform records only `h1:` hashes. Add the `zh:` hashes the registry would record — the SHA-256 of every platform's zip, from the signed `SHA256SUMS` verified in Step 1 — so the file matches what CI's registry install expects:

```bash
{ grep -E '^\s+"h1:' .terraform.lock.hcl; grep -E '\.zip$' "$SCRATCH/sums/SHA256SUMS" | awk '{printf "    \"zh:%s\",\n", $1}'; } | sort > /tmp/hashes
python3 - <<'EOF'
import re
p = ".terraform.lock.hcl"
s = open(p).read()
hashes = open("/tmp/hashes").read()
s = re.sub(r"(  hashes = \[\n)(.*?)(  \])", lambda m: m.group(1) + hashes + m.group(3), s, flags=re.S)
open(p, "w").write(s)
EOF
rm /tmp/hashes
cat .terraform.lock.hcl
cd "$OLDPWD"
```

Expected: one `provider "registry.terraform.io/hashicorp/google"` block, `version = "8.4.0"`, `constraints = "~> 8.4"`, two `h1:` and one `zh:` per platform in `SHA256SUMS`, sorted.

- [ ] **Step 5: Add `make tf` and the plan-file ignores**

In `Makefile`, after `terraform-deps`:

```make
TF_ENV := CHECKPOINT_DISABLE=1 TF_IN_AUTOMATION=1 TF_INPUT=0

.PHONY: tf
tf: terraform-deps ## terraform fmt, init without a backend, validate and test
	cd $(TF_DIR) && $(TF_ENV) $(TERRAFORM) fmt -check -recursive
	cd $(TF_DIR) && $(TF_ENV) $(TERRAFORM) init -backend=false -lockfile=readonly
	cd $(TF_DIR) && $(TF_ENV) $(TERRAFORM) validate
	cd $(TF_DIR) && $(TF_ENV) $(TERRAFORM) test
```

In `.gitignore`, under `# Terraform`:

```
tfplan
*.tfplan
```

- [ ] **Step 6: Run `make tf` to see the test fail**

Run: `make tf`
Expected: FAIL — `validate` or `test` reports `Reference to undeclared input variable` for `var.project_id`, `var.region`, `var.environment`.

- [ ] **Step 7: Write the variables**

`infra/terraform/variables.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

variable "project_id" {
  description = "The GCP project of this environment."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id must be a GCP project id: 6 to 30 lowercase letters, digits or hyphens, starting with a letter and not ending with a hyphen."
  }
}

variable "region" {
  description = "The region of every regional resource."
  type        = string

  validation {
    condition     = var.region == "us-central1"
    error_message = "region must be us-central1 (docs/requirements.md, section 20); a second region is a design change, not a variable."
  }
}

variable "environment" {
  description = "The environment's name, used in labels."
  type        = string

  validation {
    condition     = contains(["lab"], var.environment)
    error_message = "environment must be lab; production is added in phase 12 (docs/design.md, section 15)."
  }
}
```

- [ ] **Step 8: Run `make tf` to see it pass**

Run: `make tf`
Expected: `Success! The configuration is valid.` and `Success! 5 passed, 0 failed.`

- [ ] **Step 9: Commit**

```bash
git add infra/terraform Makefile .gitignore
git commit -m "Add the Terraform root with validated variables and make tf

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013mmUo2SsqLxE8Zbh71vJnN"
```

---

### Task 3: The APIs and the service accounts of design §12.1

**Files:**
- Create: `infra/terraform/apis.tf`, `service_accounts.tf`, `outputs.tf`
- Create: `infra/terraform/tests/resources.tftest.hcl`

**Interfaces:**
- Consumes: the variables of Task 2.
- Produces: `google_project_service.api` (keyed by service name); `google_service_account.this` (keyed `run`, `migrate`, `invoker`); output `service_accounts` (map of key → e-mail).

- [ ] **Step 1: Write the failing test**

`infra/terraform/tests/resources.tftest.hcl`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

# What F2 declares (docs/superpowers/specs/2026-09-27-f2-gcp-bootstrap-terraform-design.md,
# sections 5.3 and 5.4). No test here reaches GCP.

mock_provider "google" {}

variables {
  project_id  = "aleogr-identity-lab-6f2g"
  region      = "us-central1"
  environment = "lab"
}

run "exactly_the_apis_terraform_needs" {
  command = plan

  assert {
    condition = toset(keys(google_project_service.api)) == toset([
      "cloudresourcemanager.googleapis.com",
      "iam.googleapis.com",
      "iamcredentials.googleapis.com",
      "serviceusage.googleapis.com",
      "storage.googleapis.com",
      "sts.googleapis.com",
    ])
    error_message = "F2 enables exactly the six APIs Terraform and the federation need."
  }

  assert {
    condition     = alltrue([for s in google_project_service.api : s.disable_on_destroy == false && s.disable_dependent_services == false])
    error_message = "Removing an API from the code must never switch the service off."
  }

  assert {
    condition     = alltrue([for s in google_project_service.api : s.project == var.project_id])
    error_message = "Every API is enabled in the environment's project."
  }
}

run "the_three_service_accounts_of_design_12_1" {
  command = plan

  assert {
    condition = {
      for k, sa in google_service_account.this : k => [sa.account_id, sa.display_name]
      } == {
      run     = ["identity-run", "identity service"]
      migrate = ["identity-migrate", "identity migrations"]
      invoker = ["identity-invoker", "identity task and scheduler invoker"]
    }
    error_message = "The service accounts are exactly identity-run, identity-migrate and identity-invoker."
  }

  assert {
    condition     = alltrue([for sa in google_service_account.this : sa.project == var.project_id])
    error_message = "Every service account lives in the environment's project."
  }
}

run "the_outputs_name_every_service_account" {
  command = plan

  assert {
    condition     = toset(keys(output.service_accounts)) == toset(["run", "migrate", "invoker"])
    error_message = "The output lists the three service accounts."
  }
}
```

- [ ] **Step 2: Run to see it fail**

Run: `make tf`
Expected: FAIL — `Reference to undeclared resource` for `google_project_service.api` and `google_service_account.this`.

- [ ] **Step 3: Write the resources**

`infra/terraform/apis.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

# Each delivery enables only the APIs it uses, in the pull request that creates
# the resources (docs/design.md, section 12.1). F2's are the ones Terraform and
# the federation need; the bootstrap enabled them first, so declaring them
# changes nothing.
locals {
  apis = toset([
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
    "sts.googleapis.com",
  ])
}

resource "google_project_service" "api" {
  for_each = local.apis

  project = var.project_id
  service = each.value

  # Removing an API from this list never switches off a service something else uses.
  disable_on_destroy         = false
  disable_dependent_services = false
}
```

`infra/terraform/service_accounts.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

# The service accounts of docs/design.md, section 12.1. None has a role in F2:
# each receives its roles in the delivery that creates the resource it uses.
# CI's identities (terraform-plan, terraform-apply) belong to the bootstrap and
# are not managed here (docs/infrastructure.md).
locals {
  service_accounts = {
    run     = { id = "identity-run", name = "identity service" }
    migrate = { id = "identity-migrate", name = "identity migrations" }
    invoker = { id = "identity-invoker", name = "identity task and scheduler invoker" }
  }
}

resource "google_service_account" "this" {
  for_each = local.service_accounts

  project      = var.project_id
  account_id   = each.value.id
  display_name = each.value.name

  depends_on = [google_project_service.api]
}
```

`infra/terraform/outputs.tf`:

```hcl
# SPDX-License-Identifier: AGPL-3.0-only

output "service_accounts" {
  description = "The e-mail of each service account of docs/design.md, section 12.1."
  value       = { for k, sa in google_service_account.this : k => sa.email }
}
```

- [ ] **Step 4: Run to see it pass**

Run: `make tf`
Expected: `Success! 8 passed, 0 failed.` (5 from Task 2, 3 here).

- [ ] **Step 5: Commit**

```bash
git add infra/terraform
git commit -m "Declare the F2 APIs and the service accounts of design 12.1

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013mmUo2SsqLxE8Zbh71vJnN"
```

---

### Task 4: The workflow and Dependabot

**Files:**
- Create: `.github/workflows/terraform.yml`
- Modify: `.github/dependabot.yml`

**Interfaces:**
- Consumes: `make terraform-deps`, `make tf`; repository variables `GCP_WIF_PROVIDER`, `GCP_TF_PLAN_SA`, `GCP_TF_APPLY_SA` (Task 6).
- Produces: the required check `terraform`; the jobs `terraform-apply` and `terraform-drift`.

- [ ] **Step 1: Write the workflow**

`.github/workflows/terraform.yml`:

```yaml
# Terraform for the lab (docs/design.md, section 12.5). The job `terraform` is
# a required status check of the protect-main ruleset. The identities and their
# federation are created by hand (docs/infrastructure.md): terraform-plan only
# reads and is reachable from pull requests and the schedule; terraform-apply is
# reachable only from this file on main, by a push.
name: terraform

on:
  pull_request:
  push:
    branches: [main]
  schedule:
    - cron: "0 10 * * 1" # Mondays 10:00 UTC: the drift check

permissions: {}

env:
  CHECKPOINT_DISABLE: "1"
  TF_IN_AUTOMATION: "1"
  TF_INPUT: "0"

jobs:
  terraform:
    name: terraform
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
      id-token: write # the plan identity, through Workload Identity Federation
    defaults:
      run:
        working-directory: infra/terraform
    steps:
      - name: Refuse pull requests from forks
        if: github.event.pull_request.head.repo.full_name != github.repository
        working-directory: .
        run: |
          echo "::error::the plan needs a branch of this repository; GitHub issues no OIDC token to forks"
          exit 1
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - run: make tf
        working-directory: .
      - uses: google-github-actions/auth@7c6bc770dae815cd3e89ee6cdf493a5fab2cc093 # v3.0.0
        with:
          workload_identity_provider: ${{ vars.GCP_WIF_PROVIDER }}
          service_account: ${{ vars.GCP_TF_PLAN_SA }}
      - run: ../../bin/tools/terraform init -backend-config=lab.gcs.tfbackend -lockfile=readonly
      - run: ../../bin/tools/terraform plan -lock=false -var-file=lab.tfvars -out=tfplan
      - name: Show the plan in the job summary
        run: |
          {
            echo '### Terraform plan (lab)'
            echo '```'
            ../../bin/tools/terraform show -no-color tfplan
            echo '```'
          } >> "$GITHUB_STEP_SUMMARY"

  terraform-apply:
    name: terraform-apply
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-24.04
    timeout-minutes: 30
    concurrency:
      group: terraform-lab
      cancel-in-progress: false # merges apply one after another, never in parallel
    permissions:
      contents: read
      id-token: write # the apply identity, through Workload Identity Federation
    defaults:
      run:
        working-directory: infra/terraform
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - run: make terraform-deps
        working-directory: .
      - uses: google-github-actions/auth@7c6bc770dae815cd3e89ee6cdf493a5fab2cc093 # v3.0.0
        with:
          workload_identity_provider: ${{ vars.GCP_WIF_PROVIDER }}
          service_account: ${{ vars.GCP_TF_APPLY_SA }}
      - run: ../../bin/tools/terraform init -backend-config=lab.gcs.tfbackend -lockfile=readonly
      - run: ../../bin/tools/terraform plan -var-file=lab.tfvars -out=tfplan
      - run: ../../bin/tools/terraform apply tfplan
      - name: Show the applied plan in the job summary
        if: always()
        run: |
          {
            echo '### Terraform apply (lab)'
            echo '```'
            ../../bin/tools/terraform show -no-color tfplan 2>/dev/null || echo "no plan was produced"
            echo '```'
          } >> "$GITHUB_STEP_SUMMARY"

  terraform-drift:
    name: terraform-drift
    if: github.event_name == 'schedule'
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
      id-token: write # the plan identity, through Workload Identity Federation
    defaults:
      run:
        working-directory: infra/terraform
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - run: make terraform-deps
        working-directory: .
      - uses: google-github-actions/auth@7c6bc770dae815cd3e89ee6cdf493a5fab2cc093 # v3.0.0
        with:
          workload_identity_provider: ${{ vars.GCP_WIF_PROVIDER }}
          service_account: ${{ vars.GCP_TF_PLAN_SA }}
      - run: ../../bin/tools/terraform init -backend-config=lab.gcs.tfbackend -lockfile=readonly
      - name: Fail on any difference between the code and the lab
        run: |
          status=0
          ../../bin/tools/terraform plan -lock=false -var-file=lab.tfvars -detailed-exitcode -out=tfplan || status=$?
          if [ "$status" -eq 2 ]; then
            {
              echo '### Drift in the lab'
              echo '```'
              ../../bin/tools/terraform show -no-color tfplan
              echo '```'
            } >> "$GITHUB_STEP_SUMMARY"
            echo "::error::the lab differs from the code; see the job summary"
          fi
          exit "$status"
```

- [ ] **Step 2: Add Terraform to Dependabot**

In `.github/dependabot.yml`, append:

```yaml

  # The provider; the lock file is regenerated in the same pull request. The
  # Terraform binary is pinned by infra/terraform/.terraform-version and moves
  # by hand.
  - package-ecosystem: terraform
    directory: /infra/terraform
    schedule:
      interval: weekly
    groups:
      terraform:
        patterns: ["*"]
```

- [ ] **Step 3: Lint the workflows**

Run: `make check 2>&1 | tail -15`
Expected: `actionlint` silent; `zizmor` prints `No findings to report`; `make check` exits 0. If `zizmor` reports a finding, fix the workflow (never add an ignore without asking the owner).

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/terraform.yml .github/dependabot.yml
git commit -m "Plan on pull requests, apply on merge, check drift weekly

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013mmUo2SsqLxE8Zbh71vJnN"
```

---

### Task 5: The documents

**Files:**
- Modify: `docs/infrastructure.md`, `docs/design.md`, `docs/requirements.md`, `docs/roadmap.md`, `docs/threat-model.md`, `CLAUDE.md`, `docs/claude-code-environment.md`, `docs/superpowers/specs/2026-09-27-f2-gcp-bootstrap-terraform-design.md`

- [ ] **Step 1: `docs/infrastructure.md` — the bootstrap, ready to record**

Replace the environments row with `| lab | aleogr-identity-lab-6f2g | 894113236465 | created on 2026-09-26; bootstrap and Terraform foundation in F2 |`. Replace the sentence "The budget alert is created in F2, with the Terraform bootstrap." with "The budget is recorded under **The Terraform bootstrap** below." Add a section **The Terraform bootstrap (F2)** after **Billing**, containing: the list of what it creates and why it stays outside Terraform (spec, section 2, decision 3); the blocks of Task 6 exactly as given to the owner, each followed by `Output:` and the output the owner reports (filled in during Task 6); a subsection **Growing the apply identity's roles** stating: "A delivery that needs the apply identity to hold a new role, or to grant a new role, adds a `gcloud` step here, run by the owner; from F3 the grantable roles are the list in the `modifiedGrantsByRole` condition of `roles/resourcemanager.projectIamAdmin`." In **GitHub repository settings**, add `terraform` to the required checks list and a line for the repository variables. In **Still to do by hand**, remove the F2 row, and keep the Search Console row, now marked F3.

- [ ] **Step 2: `docs/design.md`**

- Section 1.2: replace `infra/terraform/        the lab` with `infra/terraform/        one Terraform root per deployment, parameterised by <env>.tfvars and <env>.gcs.tfbackend`.
- Section 12.1, the service accounts bullet: replace with "**Service accounts:** separate for the service (`identity-run`), the migration job (`identity-migrate`) and the task and scheduler invoker (`identity-invoker`), declared in Terraform; and CI's two identities, `terraform-plan` (read-only, pull requests and the drift check) and `terraform-apply` (push to `main` from `terraform.yml` only), created with Workload Identity Federation restricted to this repository **by hand, outside Terraform**, so CI cannot widen its own power (`docs/infrastructure.md`)."
- Section 12.5, the pull request bullet: "…plus `terraform fmt`, `validate`, `test` and a plan as the read-only identity." Add a bullet: "**Weekly:** a plan with `-detailed-exitcode` fails on any drift between the code and the lab."

- [ ] **Step 3: `docs/requirements.md` §20**

Replace the bullet "**Terraform is applied only by CI**; GitHub authenticates to GCP through Workload Identity Federation restricted to this repository." with "**Terraform is applied only by CI**; GitHub authenticates to GCP through Workload Identity Federation restricted to this repository. The bootstrap (the state bucket, CI's identities and the federation) is created by hand, recorded in `docs/infrastructure.md`, and stays outside Terraform."

- [ ] **Step 4: `docs/roadmap.md`**

- F1: `- [ ]` → `- [x]`.
- F2's **What is needed from the owner**: replace the list with the five steps of spec section 9 (the network setting first; the project already exists).
- F2's **Scope**: replace with "`infra/terraform/` (one root parameterised by environment, provider and lock file pinned, remote state in the bootstrap's bucket, the six APIs Terraform needs, the three service accounts of design §12.1 without roles, `terraform test` against a mocked provider); `.github/workflows/terraform.yml` (`terraform` plans every pull request as the read-only identity, `terraform-apply` applies on merge, `terraform-drift` weekly); `make tf` and `make terraform-deps` (Terraform verified against HashiCorp's signature); `docs/infrastructure.md` recording every manual step."
- F3's **What is needed from the owner**: add "Grant `terraform-apply` `roles/resourcemanager.projectIamAdmin` with the `modifiedGrantsByRole` condition listing the roles F3 grants, and the roles Cloud Run and Artifact Registry need (commands given)."
- Appendix, network row: "`vuln.go.dev`, `github.com`, `gcr.io`, `storage.googleapis.com` and (from F2) `registry.terraform.io` allowed; `go.dev` and GitHub release downloads refused; `proxy.golang.org`, `pypi.org`, `releases.hashicorp.com` reachable" and consequence "Tools come through the Go proxy, PyPI or HashiCorp's signed releases; Trivy runs only in CI". Appendix `gcloud, terraform` row: consequence "GCP steps are written for Cloud Shell; Terraform is installed by `make terraform-deps` and applied only in CI".

- [ ] **Step 5: `docs/threat-model.md`**

- B7-02, mitigation cell: append "; Workload Identity Federation bound to the repository's and owner's numeric ids, the apply identity reachable only from `terraform.yml` on `main` by a push". Verification cell: append "; the federation's condition and mapping recorded in `docs/infrastructure.md`".
- Add rows after B7-04:
  - `| B7-05 | T, E | A pull request executes code during the Terraform plan (providers, data sources) | The plan identity only reads and cannot write the state; forks get no token and the plan job refuses them | The roles recorded in docs/infrastructure.md; the fork check in the terraform job | 1 |`
  - `| B7-06 | E | CI widens its own power | The bootstrap is outside Terraform; the apply identity's custom role excludes setIamPolicy; from F3, role granting is bounded by modifiedGrantsByRole to a list only the owner extends | The recorded roles and condition; the weekly drift check | 1 |`
- Add a row after B4-05:
  - `| B4-06 | I | Terraform's state exposes secrets | A private bucket with uniform access and public access prevention, versioned, readable only by the two CI identities and the owner; sensitive values marked sensitive in the code | The bucket's settings recorded in docs/infrastructure.md | 1 |`
- Section 3's diagram: add `[GitHub Actions] ──B7── [GCP: Terraform state, lab resources] (Workload Identity Federation)` under the B7 line.

- [ ] **Step 6: `CLAUDE.md`**

- "Once the code exists, the checks CI runs are the ones to run before pushing; keep this list current:" → "The checks CI runs are the ones to run before pushing; keep this list current:".
- Replace the `make tf` line with:

```
make tf           # terraform fmt -check, init -backend=false, validate and test (mocked provider)
make terraform-deps  # installs the pinned Terraform into bin/tools/, verified against HashiCorp's key
```

- After "The OpenID Foundation's conformance suite, Trivy and CodeQL run only in CI, with Docker. Terraform is only ever applied by CI." add: "`make tf` needs `registry.terraform.io`, allowed in the environment's network settings."

- [ ] **Step 7: `docs/claude-code-environment.md`**

- Network table: add `| registry.terraform.io | make tf, which downloads the Google provider (from F2) |` and `| releases.hashicorp.com | make terraform-deps (reachable through the default list) |` — check first with `curl -sS -o /dev/null -w '%{http_code}' https://releases.hashicorp.com/terraform/` and record what is true. Change the date sentence to name 2026-09-26 and the F2 addition with its date.
- "### Why the guard tests the Makefile, not the directory": replace the last sentence with "Testing the `Makefile` keeps the script correct before F1; between F1 and F2 the Makefile existed without `terraform-deps`, and the log carried one expected `FAILURE: terraform not installed` line until F2 merged."
- "### Why the end-to-end and Terraform dependencies are installed by the script": replace "`make terraform-deps` arrives with F2." with "`make terraform-deps` (F2) installs the Terraform pinned in `infra/terraform/.terraform-version` into `bin/tools/`, verified against HashiCorp's signing key."

- [ ] **Step 8: The spec, where implementation refined it**

In the spec:
- Section 4.6: the repository variables are only those the workflow reads: `GCP_WIF_PROVIDER`, `GCP_TF_PLAN_SA`, `GCP_TF_APPLY_SA`. The project and the bucket come from the versioned `lab.tfvars` and `lab.gcs.tfbackend`, so there is one source for each.
- Section 6.3: `google-github-actions/auth` writes its credentials file in the workspace and removes it in its post step; no job of `terraform.yml` uploads the workspace.
- Section 5.6: the installer supports Linux only (sessions and CI).

- [ ] **Step 9: Check and commit**

Run: `make check 2>&1 | tail -5` (gitleaks over the new text).
Expected: exit 0.

```bash
git add docs CLAUDE.md
git commit -m "Record the F2 design in the requirements, design, roadmap and threat model

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013mmUo2SsqLxE8Zbh71vJnN"
```

---

### Task 6: The owner's manual steps, one at a time

Give each block to the owner **alone**, wait for the reported output, record block and output in `docs/infrastructure.md` (or `docs/claude-code-environment.md` for step 1), then give the next. Commit after each recorded step with a message naming it.

- [ ] **Step 1: Network** — in claude.ai/code, environment **identity** → **Network access** → Custom → add `registry.terraform.io` → save. Applies to new sessions.

- [ ] **Step 2: Bootstrap A — the project and the APIs** (Cloud Shell)

```bash
gcloud config set project aleogr-identity-lab-6f2g
gcloud services enable iam.googleapis.com iamcredentials.googleapis.com sts.googleapis.com \
  cloudresourcemanager.googleapis.com serviceusage.googleapis.com storage.googleapis.com
gcloud services list --enabled --format="value(config.name)" | sort
```

- [ ] **Step 3: Bootstrap B — the state bucket**

```bash
gcloud storage buckets create gs://aleogr-identity-lab-6f2g-tfstate \
  --location=us-central1 --uniform-bucket-level-access --public-access-prevention
gcloud storage buckets update gs://aleogr-identity-lab-6f2g-tfstate --versioning
cat > /tmp/lifecycle.json <<'EOF'
{"rule": [{"action": {"type": "Delete"}, "condition": {"isLive": false, "numNewerVersions": 20}}]}
EOF
gcloud storage buckets update gs://aleogr-identity-lab-6f2g-tfstate --lifecycle-file=/tmp/lifecycle.json
gcloud storage buckets describe gs://aleogr-identity-lab-6f2g-tfstate \
  --format="yaml(location,uniform_bucket_level_access,public_access_prevention,versioning_enabled,lifecycle_config)"
```

- [ ] **Step 4: Bootstrap C — the custom role, the two identities and their roles**

```bash
P=aleogr-identity-lab-6f2g
gcloud iam roles create identityTerraformServiceAccounts --project=$P \
  --title="identity Terraform: service accounts" \
  --description="Create and manage service accounts, never their IAM policies (threat B7-06)" \
  --permissions=iam.serviceAccounts.create,iam.serviceAccounts.delete,iam.serviceAccounts.disable,iam.serviceAccounts.enable,iam.serviceAccounts.get,iam.serviceAccounts.list,iam.serviceAccounts.undelete,iam.serviceAccounts.update,resourcemanager.projects.get \
  --stage=GA
gcloud iam service-accounts create terraform-plan --display-name="Terraform plan (CI, read-only)"
gcloud iam service-accounts create terraform-apply --display-name="Terraform apply (CI, main only)"
PLAN=serviceAccount:terraform-plan@$P.iam.gserviceaccount.com
APPLY=serviceAccount:terraform-apply@$P.iam.gserviceaccount.com
gcloud projects add-iam-policy-binding $P --member=$PLAN --role=roles/viewer --condition=None --format=none
gcloud projects add-iam-policy-binding $P --member=$PLAN --role=roles/iam.securityReviewer --condition=None --format=none
gcloud projects add-iam-policy-binding $P --member=$APPLY --role=roles/serviceusage.serviceUsageAdmin --condition=None --format=none
gcloud projects add-iam-policy-binding $P --member=$APPLY --role=projects/$P/roles/identityTerraformServiceAccounts --condition=None --format=none
gcloud storage buckets add-iam-policy-binding gs://$P-tfstate --member=$PLAN --role=roles/storage.objectViewer --format=none
gcloud storage buckets add-iam-policy-binding gs://$P-tfstate --member=$APPLY --role=roles/storage.objectAdmin --format=none
gcloud projects get-iam-policy $P --flatten=bindings --filter="bindings.members:terraform-" --format="table(bindings.role,bindings.members)"
gcloud storage buckets get-iam-policy gs://$P-tfstate --format=json | grep -A3 terraform-
```

- [ ] **Step 5: Bootstrap D — the federation**

```bash
gcloud iam workload-identity-pools create github --location=global --display-name="GitHub Actions"
gcloud iam workload-identity-pools providers create-oidc github-actions \
  --location=global --workload-identity-pool=github \
  --display-name="aleogr/identity" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-condition="assertion.repository_id == '1387974672' && assertion.repository_owner_id == '2933195'" \
  --attribute-mapping="^;^google.subject=assertion.sub;attribute.repository_id=assertion.repository_id;attribute.tf_role=(assertion.job_workflow_ref == 'aleogr/identity/.github/workflows/terraform.yml@refs/heads/main' && assertion.event_name == 'push') ? 'apply' : ((assertion.job_workflow_ref.startsWith('aleogr/identity/.github/workflows/terraform.yml@') && (assertion.event_name == 'pull_request' || assertion.event_name == 'schedule')) ? 'plan' : 'none')"
gcloud iam workload-identity-pools providers describe github-actions --location=global \
  --workload-identity-pool=github --format="yaml(name,attributeCondition,attributeMapping,state)"
```

(`^;^` makes `;` the separator, since the expressions contain commas and equals signs.)

- [ ] **Step 6: Bootstrap E — who may impersonate whom**

```bash
P=aleogr-identity-lab-6f2g
POOL=principalSet://iam.googleapis.com/projects/894113236465/locations/global/workloadIdentityPools/github
gcloud iam service-accounts add-iam-policy-binding terraform-plan@$P.iam.gserviceaccount.com \
  --role=roles/iam.workloadIdentityUser --member="$POOL/attribute.tf_role/plan" --format=none
gcloud iam service-accounts add-iam-policy-binding terraform-apply@$P.iam.gserviceaccount.com \
  --role=roles/iam.workloadIdentityUser --member="$POOL/attribute.tf_role/apply" --format=none
for sa in terraform-plan terraform-apply; do
  gcloud iam service-accounts get-iam-policy $sa@$P.iam.gserviceaccount.com --format="table(bindings.role,bindings.members)"
done
```

- [ ] **Step 7: The budget**

```bash
gcloud services enable billingbudgets.googleapis.com --project=aleogr-identity-lab-6f2g
gcloud billing budgets create --billing-account=0194C5-7DEBC1-C23BB3 \
  --display-name="identity-lab" --budget-amount=10USD \
  --filter-projects=projects/aleogr-identity-lab-6f2g \
  --threshold-rule=percent=0.5 --threshold-rule=percent=0.9 \
  --threshold-rule=percent=1.0 --threshold-rule=percent=1.0,basis=forecasted-spend
gcloud billing budgets list --billing-account=0194C5-7DEBC1-C23BB3 \
  --format="yaml(displayName,amount,budgetFilter.projects,thresholdRules)"
```

(`billingbudgets.googleapis.com` is enabled for the owner's `gcloud` call only; Terraform does not declare it and the plan ignores it.)

- [ ] **Step 8: The repository variables** — GitHub → **Settings → Secrets and variables → Actions → Variables → New repository variable**, three times:

| Name | Value |
|---|---|
| `GCP_WIF_PROVIDER` | `projects/894113236465/locations/global/workloadIdentityPools/github/providers/github-actions` |
| `GCP_TF_PLAN_SA` | `terraform-plan@aleogr-identity-lab-6f2g.iam.gserviceaccount.com` |
| `GCP_TF_APPLY_SA` | `terraform-apply@aleogr-identity-lab-6f2g.iam.gserviceaccount.com` |

- [ ] **Step 9: After the pull request's first run** (Task 7) — **Settings → Rules → Rulesets → protect-main → Require status checks to pass → Add checks → `terraform`** (source GitHub Actions) → save.

---

### Task 7: Verification, security review, pull request and CI

- [ ] **Step 1: Full local verification from a clean tree**

```bash
git status --short   # empty
make check && make test && make integration && make e2e && make tf
```

Expected: every target exits 0; `make tf` ends with `Success! 8 passed, 0 failed.`; `make test` ends with `install-terraform_test: ok`. Keep the output for the pull request.

- [ ] **Step 2: Security review**

Run the `identity-security-review` skill over the branch (F2 adds a trust boundary and a secret store). Fix every failure with a test first where one applies; record the result for the pull request.

- [ ] **Step 3: Push and open the pull request**

```bash
git push -u origin claude/cool-fermi-517zqb
```

Open the pull request "F2: GCP bootstrap and the Terraform foundation" with: a summary per spec section; the verification output; the statement that `make tf` ran in this session through a verified local provider mirror because `registry.terraform.io` was allowed only for new sessions; that the real plan runs in CI and the apply only after merge; the security review's result; the checklist of the F1 pull request; the attribution line.

- [ ] **Step 4: Watch until green**

Subscribe to the pull request's activity. On a red check, read that job's log, reproduce locally where the session can, fix, prove the same check passes, then push. The `terraform` job's plan must show 6 `google_project_service` and 3 `google_service_account` to create, nothing to change or destroy. Then give the owner Task 6, Step 9.

- [ ] **Step 5: After the owner merges**

Watch the `terraform-apply` run on `main`; attach its log excerpt (the resources created) in a comment on the pull request. Report the first `terraform-drift` run when it happens.
