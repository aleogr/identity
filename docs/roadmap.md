# Phase 1 (Foundation and essential OIDC): Roadmap

> **Status:** implementation roadmap for phase 1 of `docs/design.md`, section 15, written in the
> inaugural design session of 2026-09-26; the list of deliveries was approved by the owner. It turns
> the phase into an ordered list of deliveries, each small enough for its own pull request.
>
> **Sources of truth:** `docs/requirements.md` decides *what*, `docs/design.md` decides *how*,
> `docs/threat-model.md` names the threats, `docs/research/` explains *why*. This document decides
> *in which order* and *how each step is proved done*. It reopens no decision recorded in those
> documents. Section numbers in the form §N refer to `docs/requirements.md`; "design §N" to
> `docs/design.md`; threat identifiers such as B1-02 to `docs/threat-model.md`.
>
> **Scope of this document:** phase 1 only. Phases 2 to 12 are listed in `docs/design.md`,
> section 15, and get their own roadmap when they start.

---

## 1. Goal

**Goal:** a deployed, tenant-isolated, bilingual OpenID provider in the lab, at
`identity.lab.aleogr.dev`, in which a person creates an account, signs in with a password or a
passkey, and an application signs that person in through OpenID Connect — with every operation
audited, the OpenID Foundation's Basic and Config suites green in CI, and a pipeline that refuses to
merge anything that breaks it.

**Architecture:** the layers of design §1 — `spec`, a `core` without I/O, adapters, surfaces, and the
server as the root module. Every external provider sits behind a port with a real adapter and a fake
(design §1.3). Asynchronous work goes through the queue and scheduler ports: PostgreSQL and
in-process by default, Cloud Tasks and Cloud Scheduler in the lab (design §11). Tenant isolation is
enforced twice: in the application and by PostgreSQL row-level security (design §3).

**Tech stack:** Go (the toolchain pinned in `go.mod`, 1.27.x at the time of writing), `pgx` v5,
`goose`, templ + HTMX (Alpine.js only in declared components), `go-webauthn/webauthn`, PostgreSQL 16
on the shared Cloud SQL instance, Cloud Run, Cloud KMS, Secret Manager, Cloud Tasks, Cloud Scheduler,
Artifact Registry, Terraform, GitHub Actions, pytest + Playwright (Python), the OpenID Foundation's
conformance suite (Docker, in CI).

**Methodology:** every delivery goes through the Superpowers cycle (§3): a spec in
`docs/superpowers/specs/`, approved by the owner; a plan in `docs/superpowers/plans/`, approved by the
owner; implementation with test-driven development; verification before completion; a pull request.

## 2. Global constraints

Every delivery's requirements implicitly include this section.

- **Repository language:** everything versioned is in English; translation files are the only
  exception (§3).
- **Definition of done:** every user-facing text in **both en-US and pt-BR** through translation keys,
  tests pass, and verification evidence is attached (§3).
- **The repository is public. No secret may ever be committed.** Model identifiers appear only in the
  attribution trailer of commits and pull request descriptions.
- **Every tenant table** carries `tenant_id` and is covered by row-level security (design §3).
- **Every external identifier** is stored as the pair `(provider, external_id)`; timestamps in UTC.
- **Tests use fakes of external providers, never real services.**
- **Security defaults are floors** (§15.2): no delivery adds a setting that can be relaxed below its
  floor.
- **Cryptography only from the standard library and `golang.org/x/crypto`**; secrets compared in
  constant time.
- **The core has no I/O** and imports no adapter; `depguard` enforces it from F1.
- **A delivery that adds a trust boundary, a parser, an outbound call, a secret or an extension point
  updates `docs/threat-model.md`** in the same pull request.
- **Checks that must pass before a push**, kept current in `CLAUDE.md`: `make check`, `make test`,
  `make integration`, `make e2e`, and `make tf` once Terraform exists.

## 3. What "done" means for one delivery

A delivery is finished when all of the following are true. The pull request description states each
one with its evidence; a delivery that cannot show the evidence is not finished.

1. The pipeline is green on the pull request: build, `go vet`, `staticcheck`, `golangci-lint`
   (including `depguard`), `gosec`, `govulncheck`, `gitleaks`, unit tests with `-race`, integration
   tests against a real PostgreSQL, end-to-end tests, fuzzing for the delivery's parsers, the
   conformance suites once F16 exists, and the image scanned.
2. Every user-facing string exists in `en-US` and `pt-BR`, and the catalogue-parity check passes.
3. The verification listed in the delivery was executed and its output or screenshots are attached.
4. The threats the delivery names have their verification in place.
5. Any requirement or design decision the delivery changed was updated in the same pull request.
6. Any manual step the owner performed is recorded in `docs/infrastructure.md`.

## 4. Working rules for this phase

- **Manual steps are given one at a time**, as command blocks for Cloud Shell or console clicks; the
  owner executes, reports, and only then receives the next one (§3).
- **The owner has no local environment**, and a session has no `gcloud` and no running Docker daemon
  (appendix). Terraform runs in CI only.
- **A delivery whose owner step is not ready still merges** behind its fake adapter; the real adapter
  is enabled by a Terraform variable afterwards. F11 is written that way.
- **The lab database sleeps** Monday to Thursday, 22:00 to 07:30 local time. Live checks against the
  lab are made inside the awake window; CI never depends on the lab.

## 5. Deliveries

Ordered. Each one is a single pull request unless its plan splits it. `Depends on` lists only hard
dependencies; section 6 shows what can run in parallel.

---

### F1 — Repository skeleton, modules, pipeline and build identifier

- [ ] **Objective:** every later pull request is checked by a pipeline that runs the full command list
  of `CLAUDE.md`, the module boundaries are enforced from the first line of code, and the binary
  starts, announces which build it is, and answers a health check.

**Depends on:** nothing (the governance files of the inaugural session are already merged).

**What is needed from the owner:**
1. Enable branch protection on `main`: require a pull request and require the CI checks to pass.
2. Enable Dependabot alerts and security updates.

**Scope:**
- `go.work` and the phase-1 modules of design §1.2: the root module (`cmd/identity`), `spec`, `core`,
  `adapters/postgres`, `adapters/gcp`, `adapters/mail`, `conformance`; each with its `LICENSE` and
  SPDX headers; a CI check that every Go file carries one.
- Lint and security tools pinned as `tool` directives; `.golangci.yml` with `depguard` rules
  enforcing design §1.2's dependency direction and a rule against `math/rand` in security packages.
- `cmd/identity serve`: HTTP server, `/health`, graceful shutdown on `SIGTERM`.
- Configuration loading that **refuses to start** on an unknown or invalid value, naming it.
- `log/slog` JSON logging with Cloud Logging's field names; typed secret wrappers that redact (B4-04).
- Build identifier from `git describe`, logged at start.
- `Makefile`: `check`, `test`, `integration`, `e2e`, `e2e-deps`, `build`, `run`.
- `.github/workflows/ci.yml` with actions pinned by commit SHA and least-privilege permissions
  (B7-02); `gitleaks`; the image built and scanned by Trivy.
- `e2e/` with pinned `playwright` (the version matching the environment's Chromium, appendix) and a
  smoke test.

**Verification:**
- `make check && make test && make e2e` green, output attached; the pipeline green on this pull
  request.
- A test that a boolean variable with a typo makes the process exit non-zero naming it.
- A deliberate `core` → adapter import rejected by `depguard` (shown in the pull request, then
  removed).

**Not in this delivery:** any GCP resource, any database.

---

### F2 — GCP bootstrap and the Terraform foundation

- [ ] **Objective:** the lab project exists, Terraform's state lives in it, and CI can plan and apply
  without stored keys.

**Depends on:** F1.

**What is needed from the owner:**
1. Create the project `aleogr-identity-lab-<suffix>` on the paid billing account (commands given).
2. Run the bootstrap block in Cloud Shell: state bucket, Terraform service account, Workload Identity
   Federation restricted to this repository.
3. Create the budget alert.
4. Set the repository variables the workflows read.

**Scope:** `infra/terraform/` with the provider pinned, remote state, the enabled APIs, the service
accounts of design §12.1; `.github/workflows/terraform.yml` (plan on pull requests, apply on merge);
`make tf` and `make terraform-deps`; `docs/infrastructure.md` recording every manual step.

**Verification:** a plan on the pull request and an apply on merge, logs attached; `make tf` green in
the session.

**Not in this delivery:** Cloud Run, the database.

---

### F3 — Cloud Run in the lab and continuous deployment on merge

- [ ] **Objective:** every merge into `main` deploys the binary to the lab automatically, and the
  platform answers at its host.

**Depends on:** F2.

**What is needed from the owner:** the Cloudflare CNAME `identity.lab.aleogr.dev` →
`ghs.googlehosted.com`, "DNS only", when this delivery merges.

**Scope:** Artifact Registry with a cleanup policy; the Cloud Run service (scale to zero, maximum 2
instances); the domain mapping; the deploy workflow (build once, push, deploy, smoke test against the
`run.app` address); the build identifier on an operator-only version endpoint.

**Verification:** the deploy workflow's log; the health check answering on the `run.app` address and,
once the certificate exists, on `identity.lab.aleogr.dev`.

**Not in this delivery:** the database.

---

### F4 — The database, migrations and the database layer

- [ ] **Objective:** the service reaches its own database in the shared instance, with isolation from
  the other laboratories and a connection budget proved from the code.

**Depends on:** F3.

**What is needed from the owner:**
1. In `aleogr/lab`, grant this project's Terraform identity the tenant role on the shared instance.
2. After the apply, run the isolation grants through the Cloud SQL Auth Proxy (commands given):
   `CONNECT` revoked from `PUBLIC`, granted to this project's two users, `CONNECTION LIMIT 8`.

**Scope:** Terraform for the `identity` database, `identity_migrator` (password in Secret Manager) and
the service's IAM user; the Cloud SQL connector; `pgx` pools sized to the budget of design §12.2;
`identity migrate` as a Cloud Run Job run before each deployment; `goose` migrations; the integration
test harness that starts its own PostgreSQL 16 in a session and uses a service container in CI; the
two database roles (B4-03).

**Verification:** the migration job's log; a throwaway role refused `CONNECT`; the peak connection
count computed in the pull request against the ceiling; integration tests green.

**Not in this delivery:** tenant tables.

---

### F5 — Tenancy, host resolution, row-level security and bootstrap

- [ ] **Objective:** every request belongs to a tenant resolved from its host, no query can cross
  tenants, and a fresh deployment bootstraps its first tenant safely.

**Depends on:** F4.

**Scope:** `tenant` and `tenant_host`, with a data region column; hosts declared in `.tfvars` and
seeded by the migration job; resolution through the owning-role function; the transaction's tenant
session variable; policies on every table and **a CI check that fails when a table has no policy**
(B6-01); cache keys always including the tenant (B6-05); the bootstrap tenant and the single-use
bootstrap token printed once; multi-tenant creation off by default.

**Verification:** cross-tenant tests for every store; the CI policy check shown failing on a table
without a policy; an unknown host answering the neutral page.

**Threats:** B6-01, B6-05.

---

### F6 — Request pipeline, outbound client and internationalisation

- [ ] **Objective:** every request passes the pipeline of design §2, every outbound call goes through
  one SSRF-safe client, and every page speaks both languages.

**Depends on:** F5.

**Scope:** security headers, CSP with nonces, `frame-ancestors`; CSRF protection; rate limiting with
PostgreSQL sliding windows shared by all instances; the trusted-proxy count for the client address;
**the outbound HTTP client** (HTTPS only, private and metadata ranges refused after resolution and on
every redirect, size and time limits); language resolution by URL prefix, cookie and header; the
translation catalogues with a parity check and a check that templates contain no literal text.

**Verification:** header tests; CSRF tests; the limiter across two server processes sharing one
database; SSRF tests with a resolver returning private addresses and a redirect chain; the parity
check.

**Threats:** B1-05, B1-06, B1-07 (headers), B3-01.

---

### F7 — The settings registry: defaults and floors

- [ ] **Objective:** every security-relevant setting is declared once with its default and floor, and
  no facade can cross a floor.

**Depends on:** F5.

**Scope:** the registry of design §7 (type, default, floor or ceiling, weakening flag, scope,
translation keys); tenant settings storage; reads and writes only through the registry; every change
audited once F10 lands (the hook is here, the audit write in F10); the operator's stricter floors.

**Verification:** a table-driven test that every floor refuses a value below it; a test that a
non-weakening setting (composition rules) has no floor.

**Threats:** X-05.

---

### F8 — Keys: the KMS, the hierarchy, signing keys and JWKS

- [ ] **Objective:** each tenant has its own data key and signing keys, unwrapped lazily, rotated on
  schedule, published in its JWKS, and never used from the database in plaintext.

**Depends on:** F5.

**Scope:** the key-encryption port with the Cloud KMS adapter, a local-key adapter and a fake; the
hierarchy of design §4.1; lazy unwrap with a bounded cache; signing keys (ES256 default, EdDSA,
PS256, RS256) with rotation that publishes the next key before use; the JWKS endpoint; the `Signer`
interface with the in-memory implementation; the refused algorithms of §17.

**Verification:** tests that the database holds no plaintext key (schema scan, B4-02); rotation tests;
a KMS failure test (B4-05); a CI log line from the lab showing the unwrap on first request.

**Threats:** B4-02, B4-05, X-02.

---

### F9 — Outbox, queue and scheduler

- [ ] **Objective:** domain events are written with their data and delivered at least once, both on a
  self-hosted PostgreSQL-only deployment and on Cloud Run scaled to zero.

**Depends on:** F5.

**Scope:** `outbox_event`; the queue port with the PostgreSQL (`SKIP LOCKED`) and Cloud Tasks adapters;
the scheduler port with the in-process and Cloud Scheduler adapters; the internal endpoint
authenticated by the invoker's OIDC token; job locks; scheduled work inside the database's awake
window; Terraform for the queue and the jobs.

**Verification:** each adapter passes the port's shared suite; a lab run showing an event dispatched by
the scheduled tick.

---

### F10 — The audit log

- [ ] **Objective:** every change is recorded in a tamper-evident log whose personal fields can be
  erased by destroying a key.

**Depends on:** F7, F8, F9.

**Scope:** `audit_entry` with the per-tenant hash chain; per-user audit keys under the tenant data key;
the audit write wired into the settings registry and into every store's mutating operations; the
scheduled chain verification with an alert on a break.

**Verification:** a test that tampering with one entry is detected; a crypto-shredding test; the
verification job's lab log.

**Threats:** B2-14, and the audit column of every later threat.

---

### F11 — E-mail

- [ ] **Objective:** the platform sends transactional e-mail in both languages through a pluggable
  provider, and the lab sends real mail from `identity@lab.aleogr.dev`.

**Depends on:** F6, F9.

**What is needed from the owner:** a Brevo API key for this project, added to its Secret Manager
(command given). The delivery merges with the fake adapter if the key is not ready.

**Scope:** the mail port with SMTP, Brevo and the file-writing fake; templates in `.txt` and `.html`
per language; delivery through the queue; a probe command that sends one message by hand.

**Verification:** template tests in both languages; the probe's delivery to the owner's inbox, not
spam.

---

### F12 — Users, identifiers and passwords

- [ ] **Objective:** a user exists with verified identifiers and a password hashed and checked under
  the policy of §6.1.

**Depends on:** F7, F10, F11.

**Scope:** `app_user`, the identifier registry (e-mail, phone, username) with normalisation;
e-mail verification; argon2id with parameters from a benchmark recorded for the lab's CPU; the
password policy with its floors from F7; the breached-password port (Pwned Passwords by k-anonymity,
offline dataset, fake) failing open with a recorded skip; **the legacy verifier interface and the
rehash path**, exercised by a test-only verifier.

**Verification:** policy tests at every floor; the benchmark recorded in `docs/infrastructure.md`; the
rehash test.

**Threats:** B1-01 (blocklist part), B4-02.

---

### F13 — The flow engine, hosted login and sessions

- [ ] **Objective:** a person signs up, signs in and signs out through the hosted login, in either
  language, with or without JavaScript.

**Depends on:** F6, F12.

**Scope:** `core/flow` state machines for sign-up, sign-in and sign-out; the templ + HTMX renderer;
server-side sessions with `__Host-` cookies, hashes stored, idle and absolute timeouts, rotation on
every authentication change; enumeration-resistant responses with constant work; rate limits and the
progressive delay on sign-in.

**Verification:** end-to-end tests of the three journeys in both languages, with and without
JavaScript, screenshots attached; the enumeration timing test; session rotation tests.

**Threats:** B1-01, B1-02, B1-05, B1-08, B1-10.

---

### F14 — Passkeys

- [ ] **Objective:** a person registers a passkey and signs in with it, including through autofill, and
  a password user is offered an automatic upgrade.

**Depends on:** F13.

**Scope:** WebAuthn Level 3 registration and authentication over `go-webauthn/webauthn` behind the
project's interface; the relying party ID per tenant host; conditional mediation and conditional
create; the one embedded script; counter regression flagging.

**Verification:** end-to-end tests with Chromium's virtual authenticator; negative tests for origin,
RP ID and user verification.

**Threats:** B1-04, B1-11, B1-12.

---

### F15 — The OpenID provider

- [ ] **Objective:** an application signs a user in through OpenID Connect, and a service obtains a
  token with client credentials.

**Depends on:** F8, F13.

**Scope:** the authorisation endpoint (code flow, PKCE `S256` required, exact redirect matching,
`iss` response parameter, consent); the token endpoint (authorisation code, refresh with rotation and
family revocation, client credentials; `client_secret_basic`, `client_secret_post`,
`private_key_jwt`); ID tokens; userinfo; discovery; the JWT access token profile; the level-1 hook
interfaces `PostAuthentication` and `TokenClaims` invoked with their contract harness; the refusals
of B2-04.

**Verification:** protocol tests including every negative case of B2-01 to B2-06; a sample client
signing in against the lab; `zitadel/oidc` as an interoperating relying party in integration tests.

**Threats:** B2-01, B2-02, B2-03, B2-04, B2-05 (rotation), B2-06, B2-07.

---

### F16 — Conformance

- [ ] **Objective:** "the OpenID provider works" is a green check, not a judgement.

**Depends on:** F15.

**Scope:** `conformance/` as a black-box HTTP suite over the `Target` interface, with the server as
its first target; a CI job that runs the OpenID Foundation's conformance suite with Docker against the
binary for the **Basic** and **Config** profiles, keeps the reports as artefacts, and blocks the merge
on any failure.

**Verification:** the suite's reports attached; the job shown failing on a deliberately broken build
(in the pull request, then reverted).

---

### F17 — Administrator bootstrap and the minimal console

- [ ] **Objective:** the first administrator claims the deployment with the bootstrap token and
  registers clients, reads the audit log and changes settings from the console.

**Depends on:** F10, F15.

**What is needed from the owner:** run the bootstrap in the lab with the printed token.

**Scope:** the bootstrap claim flow, with mandatory TOTP or passkey for administrators; the console
shell in both languages; client registration and editing; settings through the registry; the audit
log viewer.

**Verification:** end-to-end tests of bootstrap, client registration and a floor refused in the
console; screenshots attached.

---

### F18 — Operational readiness and phase closure

- [ ] **Objective:** the phase is left operable by one person, and its promises are checked.

**Depends on:** all of the above.

**What is needed from the owner:** the address for alerts; one restore test of the database into a
temporary instance, if the shared instance allows it, or recording why it is deferred.

**Scope:** alerts on 5xx rates, failed jobs and audit chain breaks; runbooks for key rotation, a
compromised client, a compromised administrator and restoring the database; a walk of
`docs/threat-model.md` confirming every threat marked for phase 1 has its verification; the phase's
entry in `docs/design.md` marked done.

**Verification:** an alert fired and received; the threat walk attached to the pull request.

## 6. Order and parallelism

Hard dependencies only.

| Delivery | Depends on | Can start as soon as |
|---|---|---|
| F1 | — | now |
| F2 | F1 | F1 merges |
| F3 | F2 | F2 merges |
| F4 | F3 | F3 merges |
| F5 | F4 | F4 merges |
| F6 | F5 | F5 merges |
| F7 | F5 | F5 merges |
| F8 | F5 | F5 merges |
| F9 | F5 | F5 merges |
| F10 | F7, F8, F9 | the last of them merges |
| F11 | F6, F9 | the later of them merges |
| F12 | F7, F10, F11 | the last of them merges |
| F13 | F6, F12 | F12 merges |
| F14 | F13 | F13 merges |
| F15 | F8, F13 | F13 merges |
| F16 | F15 | F15 merges |
| F17 | F10, F15 | F15 merges |
| F18 | all | F17 merges |

- **F1 to F5 are strictly sequential.**
- **F6, F7, F8 and F9 are independent of each other** once F5 merges.
- **F14 and F15 are independent of each other**, as are **F16 and F17**, once their dependencies merge.
- **F11 has an owner-supplied key** and merges behind the fake if it is not ready.

## 7. External dependencies of the phase, with the recommended moment

| Dependency | Needed by | Recommended moment | Blocks phase 1? |
|---|---|---|---|
| The environment setup script with the Superpowers plugin | Every session | **Now** | No, but sessions start without the plugin |
| Branch protection and Dependabot | F1 | Immediately | No, but the pipeline is advisory until set |
| GCP project on the paid billing account, budget alert | F2 | When F1 merges | Yes, from F2 |
| Tenant role in `aleogr/lab` for this project's Terraform identity | F4 | When F3 merges | Yes, from F4 |
| Cloudflare CNAME for `identity.lab.aleogr.dev` | F3 | When F3 merges; the certificate can take 24 hours | Only F3's live check |
| Brevo API key for this project | F11 | When F10 merges | No — F11 merges with the fake |
| Bootstrap of the lab deployment | F17 | When F17 merges | Yes for F17's verification |
| Alert address | F18 | At F18 | Yes for F18 |
| Brand, CLA, paid certification, audit, legal review | Later phases and the commercial launch | See §23 | No |

## 8. Explicitly out of phase 1

**Belongs to a later phase** (`docs/design.md`, section 15): MFA, recovery, proof-of-work, security
notifications, the device list, data export and account deletion, logout profiles, the Dynamic and
Form Post profiles (phase 2); the library and SQLite (3); organisations, RBAC, multi-tenant mode,
the management API, CLI, declarative configuration and Terraform provider (4); PAR, JAR, JARM, DPoP,
mTLS, RAR, token exchange, device flow, introspection, opaque tokens, CIMD and dynamic registration,
FAPI 2.0 (5); federation and SAML (6); SCIM, Shared Signals and webhooks for events (7); WebAssembly
and webhook hooks (8); agents and MCP (9); the legacy hash catalogue and importers (10); SDKs other
than what tests need, and React components (11); production (12).

**Deliberately deferred inside topics this phase touches:**
- **Releases.** The release workflow with GoReleaser, the digest copy to `ghcr.io` and the
  reproducibility check arrive with the first release the owner asks for; phase 1 builds the image
  reproducibly so that check can pass.
- **Opaque access tokens** — JWT only in phase 1; opaque tokens need introspection (phase 5).
- **The console beyond clients, settings and audit.**

## 9. Risks specific to this phase

- **The phase is long** (18 deliveries). Each delivery leaves `main` deployable and green; the phase's
  usable result appears only at F15, which is why F1 to F14 are ordered to reach it by the shortest
  path.
- **Certificate issuance for the domain mapping takes up to 24 hours**; only F3's live check waits.
- **The shared database sleeps at night** and has a ceiling of 8 connections; F4 proves the budget
  from the code, and live checks are made in the awake window.
- **The OpenID Foundation suite runs only in CI** (Docker), never in a session; a failure there is
  diagnosed from its report artefact.
- **Protocol code is security-critical**; each protocol delivery lists its threats and their negative
  tests, and F16 turns conformance into a blocking check.

## Appendix — environment facts verified in this session (2026-09-26)

| Fact | Value | Consequence |
|---|---|---|
| Go | `go1.24.7` on the path, `GOTOOLCHAIN=auto` | `go.mod` names the exact toolchain (1.27.x), which is fetched automatically |
| PostgreSQL server | 16 installed (`/usr/lib/postgresql/16`) | Integration tests start a cluster of their own in a session |
| Docker | Installed, **daemon not running** | No testcontainers; the OpenID Foundation suite runs only in CI |
| Chromium | Revision 1194 in `/opt/pw-browsers` | The end-to-end suite pins the Playwright release built against 1194 (the marketplace's `1.56.0`), installed by `make e2e-deps` |
| Python Playwright | **Not installed** in this session's `python3` | `make e2e-deps` installs it; the environment setup script runs it once the Makefile exists |
| `pytest` on the path | A separate `uv` tool without Playwright | The suite runs as `python3 -m pytest` |
| `gcloud`, `terraform` | Not installed | GCP steps are written for Cloud Shell; Terraform is installed by `make terraform-deps` and applied only in CI |
