# Infrastructure

Reference for the environments behind the platform, and the record of the steps performed by hand.

> **Everything that can be declared is declared in Terraform** (`infra/terraform/`, from F2 of
> `docs/roadmap.md`). This document covers only what cannot be: the resources that must exist before
> Terraform can hold its own state, and the settings that live in a provider's console rather than in
> this repository. Every manual step is recorded here so it can be repeated for another environment.

---

## Environments

| Environment | Project | Number | State |
|---|---|---|---|
| lab | `aleogr-identity-lab-6f2g` | `894113236465` | created on 2026-09-26; no resources yet |
| production | not created | — | planned for phase 12 (`docs/design.md`, section 15) |

The project **number** appears inside the Workload Identity Federation principal and cannot be replaced
by the project id there.

### Naming

The marketplace's convention (owner, product, environment, random suffix):

```
aleogr  -  identity  -  lab  -  6f2g
  ^           ^          ^       ^
owner      product  environment  random suffix
```

Project ids are global and can never be reused after deletion, which is why the suffix exists. The
display name, `identity-lab`, is mutable and carries no suffix. Production will be
`aleogr-identity-prod-6f2g`.

### Created on 2026-09-26, in Cloud Shell

```bash
SUFFIX=$(tr -dc 'a-z0-9' </dev/urandom | head -c 4)
PROJECT="aleogr-identity-lab-$SUFFIX"
gcloud projects create "$PROJECT" --name="identity-lab"
gcloud projects describe "$PROJECT" --format="value(projectId,projectNumber)"
```

## Billing

Linked to the owner's only billing account, `0194C5-7DEBC1-C23BB3`, the same **paid** account as the
marketplace's lab. A free trial would stop every resource in the project when it ends.

```bash
gcloud billing projects link aleogr-identity-lab-6f2g --billing-account=0194C5-7DEBC1-C23BB3
gcloud billing projects describe aleogr-identity-lab-6f2g --format="value(billingEnabled)"
# True
```

The budget alert is created in F2, with the Terraform bootstrap.

## The host

`identity.lab.aleogr.dev` is a CNAME to `ghs.googlehosted.com` in Cloudflare, in **DNS only** mode,
created on 2026-09-26. The mode is required: with Cloudflare proxying, the host resolves to Cloudflare
and Google can never validate the domain to issue a certificate. The public resolution is the check:

```
$ getent hosts identity.lab.aleogr.dev
2607:f8b0:400c:c07::79 ghs.googlehosted.com identity.lab.aleogr.dev
```

— an address in Google's range, not Cloudflare's. Until F3 creates the Cloud Run domain mapping, the
host answers with Google's own 404.

## GitHub repository settings

Applied by hand on 2026-09-26:

- **Private vulnerability reporting:** enabled (**Settings → Code security**). It is the only reporting
  channel `SECURITY.md` names.
- **Dependabot alerts and Dependabot security updates:** enabled.
- **Secret scanning:** enabled, with push protection.
- **Code scanning:** CodeQL in **advanced setup**, through `.github/workflows/codeql.yml` (F1); the
  default setup is off, since the two cannot coexist (checked on 2026-09-26 in **Settings → Code
  security → Code scanning**). `golangci-lint` and Trivy upload their SARIF from `ci.yml`. The code
  scanning check runs (`CodeQL`, `golangci-lint`, `Trivy`) are not required checks; the blocking gate is
  the jobs below.
- **Branch protection:** the `protect-main` ruleset on the default branch, active: deletions restricted,
  force pushes blocked, a pull request required with **0 required approvals** — Claude Code never
  approves and the owner is the only human on the repository, so the guarantee comes from the required
  checks, not an approval count. **Required status checks**, added on 2026-09-26 (F1), in
  **Settings → Rules → Rulesets → protect-main → Require status checks to pass**, source GitHub
  Actions: `check`, `test`, `integration`, `e2e`, `image`, `codeql`. "Require branches to be up to
  date" is off. A delivery that adds a CI job adds its name here and in the ruleset.

## Still to do by hand, and when

| Step | Delivery |
|---|---|
| Terraform bootstrap: state bucket, Terraform service account, Workload Identity Federation; budget alert; repository variables | F2 |
| Add the Terraform service account as an owner of the `aleogr.dev` property in Search Console (**Settings → Users and permissions**), as the marketplace did; without it Cloud Run refuses the domain mapping | F2 or F3 |
| Grant this project's Terraform identity the `sqlTenant` role on the shared instance, in `aleogr/lab` | F4 |
| Isolation grants on the `identity` database through the Cloud SQL Auth Proxy: `CONNECT` revoked from `PUBLIC`, granted to this project's two users, `CONNECTION LIMIT 8` | F4 |
| Brevo API key for this project | F11 |
