# F2 — GCP bootstrap and the Terraform foundation: Design

> **Status:** spec for delivery F2 of `docs/roadmap.md`, written through the brainstorming process on
> 2026-09-27. Each section below was approved by the owner in the session; the written spec awaits the
> owner's review.
>
> **Sources:** `docs/requirements.md` (§N), `docs/design.md` ("design §N"), `docs/threat-model.md`
> (threat identifiers), `docs/roadmap.md` (F2 and its appendix), `docs/infrastructure.md`, `CLAUDE.md`,
> and the F1 spec (`2026-09-26-f1-repository-skeleton-design.md`) for the conventions it set.

---

## 1. Goal

The lab project exists, Terraform's state lives in it, and CI plans on every pull request and applies on
every merge into `main` without any stored key. The identity that runs a pull request's plan can only
read; the identity that applies can act only from `main`, and cannot widen its own power.

## 2. Decisions taken in the brainstorming session

| # | Decision | Alternative rejected |
|---|---|---|
| 1 | **The owner allows `registry.terraform.io`** in the environment's network settings, so `terraform init` fetches providers in a session exactly as in CI | A filesystem provider mirror built from `releases.hashicorp.com` (the session would run Terraform differently from CI, and the mirror must follow every provider bump); `fmt` only in the session (breaks the roadmap's "`make tf` green in the session") |
| 2 | **Two CI identities:** `terraform-plan`, read-only, for pull requests and the drift check; `terraform-apply`, for `push` to `main` from `terraform.yml` only | One identity for both (any branch with a pull request executes code with write access); a GitHub Environment with a manual approval before each apply (beyond the roadmap's "apply on merge") |
| 3 | **The bootstrap stays outside Terraform**: created with `gcloud` in Cloud Shell and recorded in `docs/infrastructure.md` | Adopting it into Terraform with `import` blocks (CI could loosen its own federation condition or lock itself out); a separate bootstrap root applied by the owner (contradicts §20, "Terraform is applied only by CI") |
| 4 | **Least privilege for the apply identity, growing per delivery, with role granting bounded by an IAM condition** from the delivery that first needs it (F3). Each new role is a manual `gcloud` step, recorded | The same roles without conditions (the apply identity could grant itself `owner`); `roles/owner` |
| 5 | **Budget of US$ 10 a month** on this project, alerts at 50%, 90% and 100% of actual spend and 100% of forecast, e-mailed to the billing administrators; created by hand | US$ 5 (noisy at 50%); automatic billing shutdown (out of scope) |
| 6 | **Each delivery enables only the APIs it uses**, in the pull request that creates the resources; F2 enables the APIs Terraform itself needs | Every phase-1 API now (unused services enabled for weeks) |
| 7 | **A weekly scheduled drift check**: a plan with `-detailed-exitcode` that fails on any difference | Relying on the next pull request's plan (a week without pull requests is blind) |

**Correction recorded during the session.** Decision 6 was first presented with the apply identity's
`serviceusage.serviceUsageAdmin` bounded by an IAM condition on the service name. No such condition
attribute exists for Service Usage (the only per-service restriction is the organisation policy
`gcp.restrictServiceUsage`, which needs an organisation). The role is granted **without a condition**;
an unexpected API shows up in the pull request's plan, and the budget catches a cost.

## 3. Environment facts this design depends on

Verified in the session on 2026-09-27.

| Fact | Value | Consequence |
|---|---|---|
| Latest Terraform | `1.16.4` (from `releases.hashicorp.com/terraform/index.json`) | Pinned in `infra/terraform/.terraform-version` |
| Latest Google provider | `8.4.0` (from `releases.hashicorp.com/terraform-provider-google/index.json`) | `~> 8.4` in `versions.tf`; the exact version and hashes in the committed lock file |
| `releases.hashicorp.com`, `www.hashicorp.com` | Reachable | `make terraform-deps` downloads the binary, `SHA256SUMS` and its signature; HashiCorp's public key is fetched once and committed |
| `registry.terraform.io` | **Refused** by the session's proxy (403 on `CONNECT`) | The owner allows it (decision 1); until a session starts with it allowed, `make tf` stops at `init` in the session and the pull request says so |
| `checkpoint-api.hashicorp.com` | Refused | `CHECKPOINT_DISABLE=1` wherever Terraform runs |
| `gpg` | Installed in the session (`/usr/bin/gpg`) and on GitHub's Ubuntu runners | Signature verification needs no extra tool |
| `vuln.go.dev`, `gcr.io` | Reachable | `make check` and `make image` run complete in the session (checked on 2026-09-27) |
| Dependabot and OIDC | Workflows triggered by Dependabot pull requests honour `permissions:` since 2021-10, including `id-token: write` | The plan runs on Dependabot's pull requests too |
| Pull requests from forks | GitHub issues no OIDC token to them | The plan job fails closed on a fork (section 6.2) |

## 4. The bootstrap (by hand, outside Terraform)

Given to the owner one block at a time, in Cloud Shell, with project `aleogr-identity-lab-6f2g` (number
`894113236465`); each block and its output is recorded in `docs/infrastructure.md` so it can be
repeated for production by changing the project.

### 4.1 APIs Terraform needs to run

`iam`, `iamcredentials`, `sts`, `cloudresourcemanager`, `serviceusage`, `storage`. Terraform declares the
same six (section 5.3), so the plan shows no difference.

### 4.2 The state bucket

`aleogr-identity-lab-6f2g-tfstate` in `us-central1`: uniform bucket-level access, public access
prevention enforced, object versioning with the last 20 non-current versions kept, Google-managed
encryption (a customer-managed key is a production decision). From F4 the state holds the migrator's
password, so it is a secret store (threat B4-06). Access: `terraform-plan` reads, `terraform-apply`
reads and writes, the owner through the project's owner role; nobody else.

### 4.3 The CI identities

| Service account | Roles |
|---|---|
| `terraform-plan` | `roles/viewer` and `roles/iam.securityReviewer` on the project (to read resources and IAM policies); `roles/storage.objectViewer` **on the state bucket only**. The plan runs with `-lock=false`, so it writes nothing |
| `terraform-apply` | `roles/serviceusage.serviceUsageAdmin` on the project (without a condition, section 2); the custom role **`identityTerraformServiceAccounts`** on the project; `roles/storage.objectAdmin` **on the state bucket only** |

The custom role `identityTerraformServiceAccounts` holds `iam.serviceAccounts.create`, `.delete`,
`.disable`, `.enable`, `.get`, `.list`, `.undelete`, `.update` and `resourcemanager.projects.get` —
**never** `iam.serviceAccounts.setIamPolicy`, so the apply identity cannot let anyone impersonate a
service account, its own included.

**Not in F2:** `roles/resourcemanager.projectIamAdmin`. No F2 service account receives a role, so the
apply identity needs no role-granting power yet. F3 adds it, with the condition
`api.getAttribute('iam.googleapis.com/modifiedGrantsByRole', []).hasOnly([...])` naming exactly the
roles it may grant; every later delivery that needs another role extends that list by hand.

### 4.4 Workload Identity Federation

- Pool `github`; OIDC provider `github-actions` with issuer `https://token.actions.githubusercontent.com`.
- **Provider condition:** `assertion.repository_id == '<id>' && assertion.repository_owner_id == '<id>'`,
  with the numeric identifiers of `aleogr/identity` and `aleogr` (read once from the GitHub API and
  recorded). A fork, or another repository later given the same name, never matches.
- **Attribute mapping:**
  - `google.subject = assertion.sub`
  - `attribute.repository_id = assertion.repository_id`
  - `attribute.tf_role` =
    `"apply"` when `assertion.job_workflow_ref == 'aleogr/identity/.github/workflows/terraform.yml@refs/heads/main' && assertion.event_name == 'push'`;
    `"plan"` when `assertion.job_workflow_ref.startsWith('aleogr/identity/.github/workflows/terraform.yml@') && assertion.event_name in ['pull_request', 'schedule']`;
    `"none"` otherwise.
- **Impersonation:** `roles/iam.workloadIdentityUser` on `terraform-plan` for
  `principalSet://iam.googleapis.com/projects/894113236465/locations/global/workloadIdentityPools/github/attribute.tf_role/plan`,
  and on `terraform-apply` for the same with `.../attribute.tf_role/apply`. Nothing is granted to
  `none`.

Only the `terraform.yml` on `main`, run by a push, can apply. A pull request can change `terraform.yml`
on its branch, but its token then carries `event_name == 'pull_request'` and at most reaches the
read-only identity.

### 4.5 The budget

US$ 10 a month, scoped to this project, thresholds 50%, 90% and 100% of actual spend and 100% of
forecast, e-mail to the billing account's administrators (the default). A budget only alerts; it stops
nothing. Created by hand because the apply identity has, and must have, no access to the billing
account.

### 4.6 The repository variables

Plain variables, not secrets (none of them is secret): `GCP_PROJECT_ID`, `GCP_WIF_PROVIDER` (the full
provider resource name), `GCP_TF_PLAN_SA`, `GCP_TF_APPLY_SA`, `GCP_TF_STATE_BUCKET`.

## 5. The Terraform code

### 5.1 Layout

One root module parameterised by environment, so production (phase 12) is one more pair of files, not a
copy of the code:

```
infra/terraform/
  .terraform-version        1.16.4; read by make terraform-deps (session, setup script and CI)
  versions.tf               required_version "~> 1.16.0"; hashicorp/google "~> 8.4"
  backend.tf                backend "gcs" {} (partial configuration)
  lab.gcs.tfbackend         bucket = "aleogr-identity-lab-6f2g-tfstate", prefix = "lab"
  lab.tfvars                project_id, region
  variables.tf              project_id and region, with validation
  providers.tf              google provider with project, region and default labels
  apis.tf                   the six APIs of section 4.1
  service_accounts.tf       the three service accounts of section 5.4
  outputs.tf                the service accounts' e-mails
  .terraform.lock.hcl       committed; hashes for linux_amd64 and darwin_arm64
  tests/*.tftest.hcl        terraform test with a mocked google provider
  hashicorp.asc             HashiCorp's release-signing public key
```

### 5.2 Variables

- `project_id`: string; validated against the GCP project id format
  (`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`).
- `region`: string; validated to be exactly `us-central1` (§20). A second region is a design change,
  not a variable.

Default labels on every resource that takes them: `app = "identity"`, `env = "lab"` (from a variable
`environment`, validated to `lab` for now) and `managed-by = "terraform"`.

### 5.3 APIs

`google_project_service` for each of the six APIs of section 4.1, with `disable_on_destroy = false` and
`disable_dependent_services = false`: removing one from the code never switches off a service
something else uses.

### 5.4 Service accounts of design §12.1

| Account id | Display name | For | Roles in F2 |
|---|---|---|---|
| `identity-run` | identity service | The Cloud Run service (F3) | None |
| `identity-migrate` | identity migrations | The migration job (F4) | None |
| `identity-invoker` | identity task and scheduler invoker | Cloud Tasks and Cloud Scheduler calling the internal endpoint (F9) | None |

Each receives its roles in the delivery that creates the resource it uses. The fourth account of
design §12.1, CI's, is the pair of section 4.3. The `identity-` prefix sets Terraform's accounts apart
from the bootstrap's at a glance; nothing in the permissions depends on it.

### 5.5 Tests

`terraform test` with `mock_provider "google"` runs in the session and in CI without credentials and
without touching GCP ("tests use fakes of external providers"). Written before the code they test:

- the three service accounts exist with exactly the ids and display names of section 5.4;
- the enabled APIs are exactly the six of section 4.1, each with `disable_on_destroy = false`;
- the default labels are set;
- `variables.tf` refuses a region other than `us-central1`, a malformed project id and an environment
  other than `lab` (`expect_failures`).

### 5.6 `make tf` and `make terraform-deps`

| Target | What it does |
|---|---|
| `terraform-deps` | Reads `.terraform-version`; downloads `terraform_<v>_linux_amd64.zip`, `terraform_<v>_SHA256SUMS` and `terraform_<v>_SHA256SUMS.sig` from `releases.hashicorp.com`; verifies the signature with `gpg` against the committed `hashicorp.asc` in a temporary keyring, then the zip's SHA-256 against the signed sums; installs `bin/tools/terraform`. Does nothing when that binary already reports the pinned version. Any failed verification stops with a non-zero status and installs nothing |
| `tf` | Depends on `terraform-deps`; in `infra/terraform/`, with `CHECKPOINT_DISABLE=1` and `TF_IN_AUTOMATION=1`: `terraform fmt -check -recursive`, `terraform init -backend=false -lockfile=readonly -input=false`, `terraform validate`, `terraform test` |

`-lockfile=readonly` makes a provider change impossible without a deliberate lock-file update in the
pull request.

### 5.7 Dependabot

`.github/dependabot.yml` gains the `terraform` ecosystem for `/infra/terraform`, weekly, grouped. A
provider bump arrives with its lock file regenerated, and is planned like any other pull request.
The Terraform binary itself is pinned by `.terraform-version` and moves by hand.

## 6. The workflow `.github/workflows/terraform.yml`

### 6.1 Shape

- Triggers: `pull_request`; `push` to `main`; `schedule`, Mondays at 10:00 UTC. No
  `workflow_dispatch`: no manual path can apply.
- `permissions: {}` at the top; each job gets `contents: read` and `id-token: write`, nothing else.
- Every action pinned by commit SHA with its version in a comment: `actions/checkout`
  (`persist-credentials: false`) and `google-github-actions/auth`. Terraform is installed by
  `make terraform-deps`, not by a third-party action.
- Environment for every Terraform step: `CHECKPOINT_DISABLE=1`, `TF_IN_AUTOMATION=1`, `TF_INPUT=0`.
- Repository variables read through `vars.*`; nothing from `secrets.*`.

### 6.2 Jobs

| Job | Runs on | What it does |
|---|---|---|
| **`terraform`** (required status check) | Every pull request, with no path filter (a required check that does not run blocks the merge) | Fails at once with "the plan needs a branch of this repository" when the pull request comes from a fork. Then `make tf`; authenticates as `terraform-plan`; `terraform init -backend-config=lab.gcs.tfbackend -lockfile=readonly`; `terraform plan -lock=false -var-file=lab.tfvars -out=tfplan`; writes `terraform show -no-color tfplan` to the job summary. It does not comment on the pull request, so it needs no `pull-requests: write` |
| **`terraform-apply`** | `push` to `main` | Its own `concurrency` group (`terraform-lab`) with `cancel-in-progress: false`, so merges apply one after another, never in parallel. `make terraform-deps`; authenticates as `terraform-apply`; `init`; `plan -out=tfplan` (with the state lock); `apply tfplan`; the plan and the apply's output in the job summary |
| **`terraform-drift`** | The schedule | `make terraform-deps`; authenticates as `terraform-plan`; `init`; `plan -lock=false -detailed-exitcode`. Exit code 2 fails the job and the summary shows what differs; GitHub notifies the owner of the failed scheduled run. Not a required check |

Only `terraform` is a pull request check; the ruleset `protect-main` gains it (section 9).

### 6.3 Workflow security (B7-02)

`actionlint` and `zizmor` in `make check` cover the new workflow; no `pull_request_target`; no secrets;
tokens are short-lived (one hour at most) and never written to disk beyond what
`google-github-actions/auth` does for the job's own credentials file, which lives in the runner's
temporary directory and is removed at the end of the job.

## 7. Threat model changes

- **B7-02** (compromised workflow): the federation is bound to `repository_id` and
  `repository_owner_id`; the apply identity is reachable only from `terraform.yml` on `main` by a push;
  the plan identity only reads. Verification: the condition and mapping recorded in
  `docs/infrastructure.md`; `actionlint` and `zizmor` in `make check`.
- **B7-05 (new)** — *A pull request executes code during the plan* (T, E): providers and data sources
  run code the pull request controls. Mitigation: the plan identity is read-only and cannot write the
  state; forks get no token. Verification: the roles of section 4.3 recorded in
  `docs/infrastructure.md`; the fork check in the `terraform` job.
- **B7-06 (new)** — *CI widens its own power* (E): Mitigation: the bootstrap is outside Terraform;
  the apply identity's custom role excludes `setIamPolicy`; role granting, from F3, is bounded by
  `modifiedGrantsByRole` to a list only the owner extends. Verification: the recorded roles and
  condition; a drift check that would show any change to managed resources.
- **B4-06 (new)** — *Terraform's state exposes secrets* (I): Mitigation: a private bucket with uniform
  access and public access prevention, versioned, readable only by the two CI identities and the owner;
  sensitive values marked `sensitive` in the code from F4. Verification: the bucket's settings recorded
  in `docs/infrastructure.md`.

## 8. Testing and verification

### 8.1 In the session

- `make tf`, with the tests of section 5.5 written first and seen failing (test-driven).
- `make check` (now also linting `terraform.yml`), `make test`, `make integration`, `make e2e`: nothing
  else changed.
- A check that `make terraform-deps` refuses a tampered download: the verification is run against a
  corrupted copy of the zip and must fail.
- The `identity-security-review` skill, since F2 adds a trust boundary and a secret store.

`make tf` needs `registry.terraform.io`; if this session still refuses it, the pull request says that
`make tf` ran completely only in CI, and the next session with the host allowed runs it again.

### 8.2 In CI

- The `terraform` job's plan on the pull request, showing the six APIs and three service accounts to
  create.
- After the owner merges: the `terraform-apply` run's log, attached in a comment on the pull request.
- The first scheduled drift run, reported when it happens.

## 9. Manual steps for the owner

Given one at a time; each is recorded in `docs/infrastructure.md` (or, for the environment, in
`docs/claude-code-environment.md`) once performed.

1. Allow `registry.terraform.io` in the `identity` environment's network settings.
2. The bootstrap in Cloud Shell, in blocks: the APIs; the state bucket; the custom role and the two
   service accounts with their roles; the Workload Identity pool and provider; the impersonation
   bindings.
3. The budget.
4. The repository variables.
5. After the pull request's first run: add `terraform` to the required status checks of `protect-main`.

Steps 2 and 4 must be done before the pull request's plan can pass.

## 10. Documents updated in the same pull request

- `docs/infrastructure.md`: every manual step with its commands and output; the "Still to do by hand"
  table; the rule that a delivery needing a new role for the apply identity adds a recorded `gcloud`
  step.
- `docs/design.md`: section 1.2, the layout of `infra/terraform/`; section 12.1, the two CI identities
  and the bootstrap outside Terraform; section 12.5, plan on pull requests, apply on merge, the weekly
  drift check.
- `docs/requirements.md` §20: "the bootstrap (state bucket, CI identities, federation) is created by
  hand, recorded in `docs/infrastructure.md`, and stays outside Terraform".
- `docs/roadmap.md`: F2's scope and owner steps as designed here; F3's owner steps gain the first
  role-granting condition; F1's checkbox marked done; the appendix's network row brought up to date.
- `docs/threat-model.md`: section 7.
- `CLAUDE.md`: `make tf` and `make terraform-deps` as they are; "Once the code exists" removed.
- `docs/claude-code-environment.md`: `registry.terraform.io` in the network table; the setup script
  section no longer describes a false `FAILURE` line.
- `.github/dependabot.yml`: the `terraform` ecosystem.

## 11. Not in this delivery

- Cloud Run, Artifact Registry, the domain mapping (F3); the database and Secret Manager (F4); any
  role for the three service accounts.
- The role-granting power of the apply identity (F3).
- The leftover inconsistencies found after F1 in the threat model, the roadmap's Makefile list, the F1
  spec and the depguard regression test: a separate clean-up delivery, agreed with the owner.
