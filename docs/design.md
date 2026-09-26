# Identity Platform: Design

> **Status:** design document produced in the inaugural design session of 2026-09-26. It decides
> *how* the product of `docs/requirements.md` is built: the shape of the code, the main mechanisms,
> the lab's infrastructure and cost, and the phases. It is the input for `docs/roadmap.md`.
>
> **Sources of truth:** `docs/requirements.md` decides *what*; this document decides *how*;
> `docs/research/` explains *why*. Section numbers in the form §N refer to `docs/requirements.md`;
> "decision N" refers to `docs/research/architecture-decisions.md`.
>
> **Status of each part:** the phases (section 15) and the lab's e-mail arrangement (section 12.6)
> were approved by the owner in the session. Everything else here is the engineering proposal that
> follows from the requirements and the research, and is confirmed or changed when the owner reviews
> this document.

---

## 1. Shape of the code

### 1.1 Layers

| Layer | Contents | Rule |
|---|---|---|
| **Contracts** — `spec/` | The domain model, the glossary, token formats, the event schema, the hook contract and its per-extension-point mutation rules, the OpenAPI document of the management and self-service APIs | Changes here are API changes and follow the stability policy (§19) |
| **Core** — `core/` | Pure Go libraries: protocol state machines, domain services, policies, the flow engine. **No I/O**: every effect goes through an interface (a *port*) the core defines | Imports only the standard library, `golang.org/x/crypto` and a short allowlist of pure libraries, enforced by lint |
| **Adapters** — `adapters/` | Implementations of the ports: PostgreSQL, SQLite, GCP (KMS, Cloud Tasks, Secret Manager), mail, delivery, identity providers, directories | Each adapter has a fake twin used by tests, and passes the port's shared test suite |
| **Surfaces** | The HTTP handlers of every protocol and API, the hosted login, the admin console, the CLI, SDKs, UI components | Surfaces call the core's services; they never reach storage directly |
| **Forms of execution** | The server (root module), the library (`lib/`), later the cloud control plane | Compose the layers; hold no domain logic |

### 1.2 Repository and modules

A Go workspace (`go.work`) with **few modules** in phase 1 (decision 8). A package becomes its own
module when an outside consumer needs to version it independently.

```
/                       module github.com/aleogr/identity        AGPL-3.0   the server and CLI
  cmd/identity/         the one binary: `identity serve`, `identity migrate`, `identity admin ...`
  internal/             HTTP surfaces, hosted login, console, wiring
  ui/                   templ templates, static assets, (phase 11) React components
spec/                   module .../identity/spec                 Apache-2.0
core/                   module .../identity/core                 Apache-2.0
adapters/postgres/      module .../identity/adapters/postgres    Apache-2.0
adapters/gcp/           module .../identity/adapters/gcp         Apache-2.0
adapters/mail/          module .../identity/adapters/mail        Apache-2.0
conformance/            module .../identity/conformance          Apache-2.0
lib/                    module .../identity/lib                  Apache-2.0   (phase 3)
adapters/sqlite/        module .../identity/adapters/sqlite      Apache-2.0   (phase 3)
sdk/go/, sdk/ts/, sdk/python/                                    Apache-2.0   (phases 3 and 11)
e2e/                    pytest + Playwright
tools/<tool>/           one module per pinned tool, outside go.work             AGPL-3.0
tools/checks/           the project's own checkers (spdxcheck, trivyignorecheck) AGPL-3.0
infra/terraform/        the lab
docs/
```

- **Why the server is the root module.** A plain `vX.Y.Z` tag belongs to the root module; making the
  root the product lets the open-source GoReleaser release on plain tags, while libraries are
  released by path-prefixed tags (`core/v0.4.0`) that Go resolves natively (§19).
- **Why GCP has its own adapter module.** The Google Cloud client libraries are large; a library user
  on another cloud must not download them.
- **Dependency direction** is enforced by `depguard` in `golangci-lint`: `core` imports no adapter,
  no surface and no GCP package; adapters import `core` and `spec`; the root module imports
  everything.
- Every module carries its own `LICENSE`; every source file an SPDX identifier.
- **Why tools sit outside the workspace, one module per tool.** In the workspace they would raise
  dependency versions `core` shares with them; in a single module, minimal version selection breaks
  the builds of some tools (found in F1).

### 1.3 The ports of phase 1

| Port | Real adapters | Fake |
|---|---|---|
| Storage (per-aggregate stores and a unit of work) | PostgreSQL; SQLite from phase 3 | In-memory, used by core unit tests |
| Key encryption (wrap and unwrap data keys) | Cloud KMS; a local key file for self-hosting without a KMS | In-memory |
| Secrets | Environment and files; Secret Manager | Map |
| Mail | SMTP; Brevo; later Amazon SES, Postmark, SendGrid | File writer |
| Queue (at-least-once jobs with idempotency keys) | **PostgreSQL** (`SKIP LOCKED` workers, the default everywhere); **Cloud Tasks** (for scale-to-zero on Cloud Run) | Synchronous |
| Scheduler (periodic jobs) | **In-process** (the default everywhere); **Cloud Scheduler** calling an internal endpoint | Manual clock |
| Clock and randomness | System | Controlled, for deterministic tests |
| Breached-password check | Pwned Passwords by k-anonymity; offline dataset | Fixed list |
| Host provisioning | "Declared in Terraform" (reports ready once the host answers) | Immediate |

The queue and scheduler ports exist because of principle 3 of §1.1: Cloud Tasks and Cloud Scheduler
are what make scale-to-zero work in the lab, but a self-hosted deployment must run with nothing but
PostgreSQL. Both adapters are production-quality; the lab chooses the GCP ones.

## 2. Request pipeline

Every request passes, in order:

1. **Tenant resolution by host.** The host is looked up in a routing table read through one function
   that runs as the owning role, because it is the one query that runs before a tenant is known. An
   unknown host gets the platform's neutral page. The resolved tenant carries its issuer, its data
   region and its settings.
2. **Security headers**, strict content security policy with a nonce, and the `noindex` header on
   every page except those a tenant marks public.
3. **Language** from the URL prefix (`/en-US/`, `/pt-BR/`), a cookie, or `Accept-Language`, for
   pages; APIs use `Accept-Language` only.
4. **Session**, from a `__Host-` cookie holding an opaque token whose hash is looked up.
5. **Tenant context:** the transaction sets the tenant as a database session variable; row-level
   security refuses rows of other tenants even when a query forgets the filter, and a transaction
   that named no tenant reads nothing.
6. **Rate limiting** on sensitive endpoints, **CSRF** protection on state-changing forms, and the
   origin data auditing needs.
7. The handler.

Behind Cloud Run, the client address is the last entry of `X-Forwarded-For` appended by Google's
front end; the number of trusted proxies is a setting.

## 3. Tenancy and isolation

- Every table carries `tenant_id`; row-level security on every table; the application also scopes
  every query. Integration tests run as the application role under row-level security.
- **Two database roles:** a migrator that owns the schema, and the service role, subject to
  row-level security. The operator's cross-tenant work runs through explicit, audited functions,
  not through a role that bypasses the policies.
- **Hosts, issuer, cookies, relying party ID** are per host (decision 5). A tenant with several
  domains lists them; WebAuthn Related Origin Requests are published per tenant.
- **Bootstrap:** at first start the server creates one tenant and prints a single-use bootstrap token;
  creating more tenants requires the multi-tenant capability, off by default.
- **The library** binds its single tenant in a middleware; one code path (decision 17).
- **Data region** is a column of the tenant from phase 1, with one value in the lab. Storage and key
  resolution take it as a parameter, which is what makes Brazilian residency an adapter later (§2.5).

## 4. Keys and cryptography

### 4.1 Key hierarchy

```
KMS key-encryption key (one per deployment and data region; never leaves the KMS)
  └─ tenant data key (AES-256-GCM, wrapped by the KMS key, stored in the database)
       ├─ tenant signing keys (ES256 default; EdDSA, PS256, RS256 available), encrypted by the data key
       ├─ tenant secrets (client secrets are hashed, not encrypted; federation secrets, webhook keys)
       └─ per-user audit keys, encrypted by the data key; destroyed on erasure (crypto-shredding)
```

- The KMS is called to **unwrap a tenant's data key, lazily, on the tenant's first request after a
  start**, then the key is cached in memory with a bounded lifetime. It is never called per token.
- **The KMS cost is constant**: one key version per deployment and region, whatever the number of
  tenants.
- **Rotation:** signing keys rotate on a schedule, the next key published in the JWKS before use and
  the retired key kept until every token it signed has expired. Data keys rotate by re-wrapping.
  The KMS key rotates in the KMS; old versions stay enabled for unwrapping.
- **`Signer` interface** with a second implementation that signs inside Cloud KMS or an HSM for a
  tenant that requires non-exportable keys (decision 14).
- Self-hosting without a KMS uses a local key-encryption key from a file or secret, with a warning at
  start in multi-tenant mode.

### 4.2 Algorithms and the FIPS profile

As §17 and decisions 13 and 15. The FIPS profile is a named setting checked at start: PBKDF2 for
passwords, the approved algorithm set, Go's FIPS 140-3 module (`GOFIPS140`), and refusal to start on
any contradiction.

### 4.3 Password hashing

argon2id parameters come from a benchmark run on the deployment's own CPU and recorded (the
marketplace's practice), and are stored with every hash so parameters can rise without invalidating
existing hashes; a stronger parameter set rehashes at the next sign-in. Legacy verifiers from phase 10
use the same rehash path.

## 5. Tokens and sessions

- **Authorisation codes** single-use, bound to the client, the redirect URI, the PKCE challenge and
  (when present) the DPoP key, valid for 60 seconds.
- **Access tokens:** JWT (RFC 9068) signed with the tenant's key, or opaque (a random handle whose
  hash maps to the grant) with introspection; five to fifteen minutes by default.
- **Refresh tokens:** opaque, rotated on every use; a reused token revokes its whole family and emits a
  security event.
- **Grants** are the durable record behind tokens (client, user, scopes, `authorization_details`,
  resource, sender constraint), so revoking a grant revokes everything issued under it.
- **Sessions:** server-side, opaque token in a `__Host-` cookie, hash in the database, idle and
  absolute timeouts, device record, revocable individually or all at once.
- **Token validation for the server's own APIs** reads signing keys from memory and grants from a
  short cache; introspection results for opaque tokens are cached for less than their remaining life.

## 6. Storage

- **Interfaces per aggregate** in `core`, domain types only, a unit of work `Tx(ctx, func(Stores)
  error) error` (decision 9). One shared test suite every adapter passes.
- **PostgreSQL** through `pgx` v5; migrations with `goose`, written in SQL, applied by
  `identity migrate` (a Cloud Run Job in the lab) before the service is deployed. Migrations are
  forwards-only; each is checked for statements that would hide rows from row-level security.
- **Naming:** tables in the singular (`tenant`, `app_user` because `user` is reserved,
  `organization`, `membership`, `client`, `grant`, `session`, `credential`, `audit_entry`,
  `outbox_event`, ...); every external identifier stored as the pair `(provider, external_id)`;
  timestamps in UTC.
- **SQLite** (phase 3) through a pure-Go driver, single-tenant only; the server refuses multi-tenant
  mode on it.
- **Rate-limit counters** live in PostgreSQL (a sliding window per key) so every instance shares them;
  a Redis adapter is optional for high-volume deployments.

## 7. Configuration: settings, defaults and floors

- A **settings registry** in `core` declares every setting once: its type, its default, its floor or
  ceiling, whether a change is a weakening, its scope (deployment, tenant, organisation, client) and
  the translation keys that explain it.
- Every facade — console, API, CLI, declarative file, Terraform provider — reads and writes through
  the registry, so a floor holds everywhere at once (§15.2).
- **Deployment configuration** comes from the environment and files, is parsed strictly, and
  **refuses to start** on an unknown or invalid value, naming the variable and quoting the value.
- **Declarative configuration** (a tenant's clients, connections, policies, branding, hooks) is a
  versioned YAML document applied at start and reconciled on change; drift between the document and
  the database is reported, and `identity admin apply --plan` shows what would change.
- Every change to a setting writes an audit entry with the previous and the new value.

## 8. Protocol engines in the core

| Package | Phase | Responsibility |
|---|---|---|
| `core/oauth` | 1, 5 | Authorisation, token, revocation, introspection endpoints' logic; grant types; PKCE; DPoP; PAR; JAR/JARM; RAR; token exchange; device flow; client authentication; CIMD and dynamic registration |
| `core/oidc` | 1, 2 | ID tokens, userinfo, discovery, claims, `acr`/`amr`, logout profiles |
| `core/webauthn` | 1 | Ceremonies over `go-webauthn/webauthn`, behind the project's interface |
| `core/password`, `core/mfa` | 1, 2 | Hashing and policy; TOTP, recovery codes, e-mail codes, step-up |
| `core/flow` | 1 | The flow state machine every user-facing journey runs on (section 10) |
| `core/tenancy`, `core/directory` | 1, 4 | Tenants, users, identifiers, organisations, memberships, invitations |
| `core/authz` | 4 | RBAC behind the `Authorizer` interface |
| `core/saml` | 6 | Requests, responses, bindings, metadata, brokering; XML signatures through `goxmldsig` behind a strict pre-validation (decision 19) |
| `core/federation` | 6 | Identity-provider adapters' common model: attribute mapping, linking, just-in-time provisioning, home-realm discovery |
| `core/scim` | 7 | Server and client, filter parser, events |
| `core/ssf` | 7 | Transmitter and receiver, security event tokens |
| `core/hooks` | 1, 8 | The extension points, the contract, the three levels |
| `core/agents` | 9 | MCP authorisation, ID-JAG issuance and acceptance, token vault |
| `core/migration` | 10 | Hash verifiers, importers, lazy migration, export |
| `core/audit`, `core/events` | 1 | Hash-chained audit log, outbox events |

Where an independent, maintained implementation exists (`zitadel/oidc`, `crewjam/saml`, OpenFGA),
it is used **as a test oracle** — a second party to interoperate with — not as a dependency of the
server (`docs/research/market.md`, section 3).

## 9. Extensibility

- **Level 1**, Go interfaces, from phase 1: every extension point is an interface called at a fixed
  place in the flow engine, with the contract of §12.1 checked by a shared test harness.
- **Level 2**, WebAssembly (phase 8): wazero, one compiled module cached per hash, a fresh instance
  per call with a memory limit and a deadline; the ABI is JSON in, JSON out, plus host functions
  declared in the plugin's manifest and checked at load (decision 11).
- **Level 3**, webhooks (phase 8): a level-1 implementation that POSTs the extension point's input,
  signed with HMAC-SHA-256 over the timestamp and body, and applies the response under the same
  mutation rules.
- Each execution writes an audit entry with the module's hash, the contract version, the duration and
  the decision.

## 10. User interface

- **The flow engine** owns every journey as a state machine: the current step, its fields, their
  validation errors, the available actions, the next transitions. The state is exposed as JSON in the
  self-service API (phase 11) and rendered by templ on the server.
- **Hosted login** (templ + HTMX): one route, one fragment per interaction; works without JavaScript
  except for the WebAuthn calls, which one small embedded script performs.
- **Admin console** (templ + HTMX, Alpine.js only in declared components) over the same service layer
  as the management API; the organisation admin portal is a scoped view of it.
- **React components** (phase 11) render the same flow state, so the logic is not repeated.
- **Branding** per tenant: logo, colours, texts per translation key, custom CSS within a restricted
  set of properties (no arbitrary CSS that could hide security indicators).
- Static assets are embedded and served with content hashes.

## 11. Audit log and events

- **Audit:** every change writes, in its transaction, an `audit_entry` with the actor, the action,
  the target, the origin, the previous and new values, and the hash of the previous entry of the same
  tenant. Personal fields are encrypted with the subject's audit key. A scheduled job walks the chain
  and alerts on a break.
- **Events:** domain events written to `outbox_event` in the same transaction; the queue port
  delivers them to subscribers (webhooks, SSF, SCIM events) with retries, idempotency keys and a
  dead-letter view in the console.
- **In the lab**, the outbox dispatcher runs from a Cloud Scheduler tick inside the database's awake
  window (section 12.2) and hands deliveries to Cloud Tasks, which calls an internal endpoint
  authenticated by an OIDC token of the invoker service account. Self-hosted deployments use the
  in-process scheduler and the PostgreSQL queue.

## 12. Infrastructure (lab)

Everything declarable is declared in Terraform in `infra/terraform/`; manual steps are recorded in
`docs/infrastructure.md`.

### 12.1 Project and services

- **GCP project** `aleogr-identity-lab-<suffix>` (the marketplace's naming: owner, product,
  environment, a random suffix because project ids are global and never reusable), region
  `us-central1`, on the paid billing account, with a budget alert.
- **Cloud Run:** one service, scale to zero, maximum instances bounded by the connection budget; one
  Cloud Run Job for migrations.
- **Artifact Registry:** one repository with a cleanup policy keeping the last images.
- **Secret Manager:** the migrator's password, the mail API key, internal tokens.
- **Cloud KMS:** one key ring and one key-encryption key.
- **Cloud Tasks and Cloud Scheduler** as in section 11.
- **Service accounts:** separate for the service, the migration job, the task and scheduler invoker,
  and CI through Workload Identity Federation restricted to this repository.

### 12.2 Database

- The database `identity` in the shared instance `lab-postgres` of project `aleogr-lab-shared-dacd`
  (§20). This repository's Terraform receives the project and the instance as variables and creates
  the database, a migrator user with a password in Secret Manager, and the service's IAM user; the
  shared repository grants this project's Terraform identity the custom role that allows exactly that.
- **Isolation inside the shared instance**, applied by hand once and recorded: `CONNECT` revoked from
  `PUBLIC` and granted to this project's two users, and a **connection limit of 8** on the database.
- **The connection budget is read from the code**, as the marketplace does: 2 connections per
  process × 2 instances for the service, plus 2 for the migration job during a deployment, is a peak
  of 6, under the ceiling of 8. `max_instance_count` is 2 in the lab.
- **The instance sleeps** Monday to Thursday, 22:00 to 07:30 local time. Scheduled jobs that touch the
  database run inside the awake window; nothing in the lab is expected to answer at night, and the
  login shows a clear error rather than hanging when the database is unreachable.
- Roles are named for the tenant of the instance (`identity_migrator`), because roles are
  cluster-wide.

### 12.3 Hosts

- `identity.lab.aleogr.dev` for the platform; hosted tenants' hosts declared in a list in the
  environment's `.tfvars`, which Terraform maps and the migration job seeds, exactly as the
  marketplace declares its marketplaces. Each host: a CNAME to `ghs.googlehosted.com` in Cloudflare,
  "DNS only", added by hand; a Cloud Run domain mapping; a certificate issued in up to 24 hours.
- The pipeline verifies deployments against the service's `run.app` address, never a mapped host.

### 12.4 mTLS in the lab

Cloud Run terminates TLS at Google's front end, so certificate-bound tokens (RFC 8705) cannot be
exercised in the lab as deployed. The code accepts the client certificate from a trusted header set
by a terminating proxy (as load balancers with mTLS do) and from a direct TLS listener when the
server terminates TLS itself. CI tests both paths against the binary. The production approach is part
of the spike of §24.

### 12.5 The deployment pipeline

- **Pull request:** every check of §21, plus `terraform fmt`, `validate` and a plan.
- **Merge into `main`:** build the image once, push it, run the migration job, deploy the revision,
  run the end-to-end suite against the `run.app` address.
- **Release** (only when the owner asks): the tag triggers GoReleaser (binaries, SBOM, signatures,
  provenance, GitHub Release), copies the lab's image **by digest** to `ghcr.io/aleogr/identity` and
  signs it, and rebuilds from the tag to check the digest is identical.
- Terraform is applied only in CI, by the Workload Identity Federation identity.

### 12.6 E-mail in the lab

Approved by the owner on 2026-09-26: the marketplace's Brevo account and its authenticated domain
`lab.aleogr.dev`, sender `identity@lab.aleogr.dev`, with **an API key of this project's own** in this
project's Secret Manager, so revoking one project's key never affects the other. The daily free quota
and the domain's reputation are shared, which is accepted for the lab; a project going to production
gets its own account and domain. Until the key exists, the lab runs the fake mail adapter.

## 13. Testing and conformance

- **Unit tests** of the core against in-memory fakes, with a controlled clock and randomness.
- **Integration tests** against a real PostgreSQL 16 with migrations applied and row-level security
  active, behind the `integration` build tag; the session starts a cluster of its own (the Docker
  daemon does not run in a session); CI uses a service container.
- **Port suites:** every storage, queue and mail adapter passes its port's shared suite.
- **End-to-end tests** in Python with Playwright against the binary started with fake providers,
  including Chromium's virtual authenticator for passkeys; screenshots kept as CI artefacts and as
  verification evidence.
- **Conformance** (decision 16): `conformance/` is a black-box HTTP suite over a `Target` interface;
  the server and, from phase 3, the library are targets. The OpenID Foundation's suite runs in CI
  with Docker against the binary for every profile in scope. SAML and SCIM interoperability tests run
  against local test instances and recorded metadata; OIDC interoperability against `zitadel/oidc`.
- **Fuzzing** of every parser, run on every pull request for a short budget and nightly for longer,
  with the corpus committed.
- **Load test** (phase 12) against the reference deployment, measuring §18's targets.

## 14. Lab cost

Estimated from public list prices on 2026-09-26; the budget alert (section 12.1) is what catches a
mistake. Several free tiers are **per billing account** and already partly used by the marketplace,
so the estimate assumes they are exhausted.

| Item | Monthly cost |
|---|---|
| Cloud SQL | **US$ 0 marginal** — a database in the shared `lab-postgres` |
| Cloud Run (scale to zero, a few requests) | ~US$ 0; within the free tier or cents |
| Cloud KMS (one software key version, a few thousand operations) | ~US$ 0.10 |
| Secret Manager (about four secret versions) | ~US$ 0.25 |
| Artifact Registry (under 1 GB with the cleanup policy) | ~US$ 0.10 |
| Cloud Scheduler (two or three jobs beyond the free three) | ~US$ 0.30 |
| Cloud Tasks, Cloud Logging, Cloud Monitoring | US$ 0 within free tiers |
| Cloudflare DNS, Brevo, GitHub Actions for a public repository | US$ 0 |
| **Total** | **Under US$ 1 per month** |

The robust deployment (dedicated highly available database, replicas, minimum instances, HSM-backed
keys, a load balancer) is reached **by configuration** of the same Terraform and code, never by code
that the lab lacks.

## 15. Phases

Approved by the owner on 2026-09-26. Each phase ends complete and usable in its scope. Phases 5 to 11
are independent of each other once phase 4 is done; the order is the recommended one and may change
with what the owner's projects need first.

| Phase | Scope | Usable at the end because… |
|---|---|---|
| **1. Foundation and essential OIDC** | Repository, CI, Terraform, Cloud Run and database in the lab; tenancy with row-level security (one tenant from bootstrap); settings registry with defaults and floors; audit log; i18n; outbox and jobs; e-mail; users, passwords and passkeys; sessions; hosted login on the flow engine; OpenID provider (authorisation code with PKCE, client credentials, refresh, discovery, JWKS, userinfo); administrator bootstrap; minimal console to register clients; level-1 hook interface; the legacy hash verifier *interface* | An application signs users in through OIDC at `identity.lab.aleogr.dev`; the OIDC Basic and Config suites are green in CI |
| **2. Account security** | MFA and step-up, recovery, abuse resistance (rate limits, proof-of-work, enumeration), security notifications, sessions and devices, data export and account deletion; the three logout profiles; the Dynamic and Form Post profiles | Sign-in has 1.0.0's full security posture |
| **3. The library** | `lib/`, SQLite, the shared storage suite, the Go SDK, examples | A Go backend embeds authentication |
| **4. Organisations, multi-tenancy and "everything is API"** | Organisations, invitations, RBAC; multi-tenant mode with per-tenant hosts; the complete console and the organisation portal; the management API (OpenAPI); the CLI; declarative configuration; the Terraform provider | Several tenants and B2B products are hosted; everything is managed as code |
| **5. Advanced OAuth and FAPI 2.0** | PAR, JAR, JARM, DPoP, mTLS, RAR, token exchange, device flow, introspection, revocation, CIMD and dynamic registration, opaque tokens; FAPI 2.0 suites green | Regulated, high-security clients are served |
| **6. Federation** | Social login, enterprise SSO over OIDC, SAML identity provider, service provider and brokering, per-organisation SSO, home-realm discovery | The enterprise SSO layer works |
| **7. Provisioning and signals** | SCIM server and client with events; SSF with CAEP and RISC; domain events by webhook | Real-time provisioning and revocation with external systems |
| **8. Extensibility** | Hooks at level 2 (WebAssembly) and level 3 (webhooks); plugin kits; the test harness | Customers adapt flows without forking |
| **9. Agents and MCP** | MCP authorisation server, ID-JAG in both roles, token vault | MCP servers and agents are authorised by the platform |
| **10. Migration** | Legacy hash catalogue, importers (Auth0, Firebase, Keycloak, CSV/JSON), lazy migration (Cognito), coexistence, rollback, complete export | A real customer migrates from a competitor |
| **11. SDKs and components** | Self-service flow API in JSON, React components, TypeScript and Python SDKs | The developer-first offering is complete |
| **12. Production and 1.0.0** | Production project and domains; FIPS profile; Helm chart, Compose and highly available reference deployments; load test; complete supply chain; internal security review | **1.0.0** |

**Why the legacy hash catalogue is not in phase 1**, although the briefing placed it there: without an
importer, verifiers have nothing to verify. Phase 1 delivers the verifier interface and the rehash
path, so phase 10 adds algorithms without touching sign-in.

**After 1.0.0**, each item of the "beyond 1.0.0" register (§2.4) becomes a phase of its own. The
first is the cloud control plane, which precedes the commercial launch.

Every phase gets its own roadmap when it starts; `docs/roadmap.md` holds the phase under way.

## 16. Decisions taken in this session

1. **Three forms of execution** (§2.1; decision 1).
2. **The server before the library** (decision 2).
3. **`compat/` beyond 1.0.0** (§2.4; decision 3); **the library for Go only** (decision 4).
4. **Host-based issuers** (§4; decision 5); **hosts declared in Terraform in the lab** behind a
   `HostProvisioner` port (section 12.3; decision 6).
5. **No event sourcing**: current-state tables, hash-chained audit, outbox (section 11; decision 7).
6. **Few modules; the server as the root module** (section 1.2; decision 8).
7. **Storage**: per-aggregate interfaces, PostgreSQL, SQLite single-tenant only (section 6;
   decision 9).
8. **RBAC in 1.0.0**, own relationship-based engine beyond (§7; decision 10).
9. **wazero with the project's own ABI** (section 9; decision 11).
10. **Queue and scheduler ports** with PostgreSQL/in-process defaults and GCP adapters for the lab
    (section 1.3).
11. **Key hierarchy** with one KMS key per deployment and region, lazy per-tenant unwrap, and an
    optional KMS/HSM signer (section 4; decision 14).
12. **Defaults and floors** through one settings registry (section 7; §15.2).
13. **Independent implementations as test oracles, not dependencies** (section 8).
14. **GoReleaser, digest promotion and the reproducibility check** (section 12.5; §19).
15. **The lab's e-mail** on the marketplace's Brevo account with its own key (section 12.6).
16. **The twelve phases** (section 15).

## 17. What remains open and what closes it

| Topic | What is needed |
|---|---|
| Run-time host provisioning for the control plane; production domains off Cloud Run domain mapping | A spike before the production phase (12) comparing Certificate Manager behind a load balancer with Cloudflare for SaaS plus a Worker (§24) |
| mTLS in production | The same spike; section 12.4 covers testing until then |
| The brand | Owner's choice before the first public npm package, image rename or Terraform provider (§23) |
| CLA text and tool | Before the first external contribution (`docs/research/licensing.md`) |

## 18. Actions that depend on the owner, with the recommended moment

| Action | Recommended moment |
|---|---|
| Branch protection and Dependabot on the repository | Phase 1, first delivery |
| Create the GCP lab project and bootstrap Terraform's state and federation | Phase 1, second delivery |
| Grant this project's Terraform identity the tenant role in `aleogr/lab`; create the `identity` database's isolation grants | Phase 1, when the database delivery starts |
| Cloudflare CNAME for `identity.lab.aleogr.dev` | When the Cloud Run delivery merges (the certificate takes up to 24 hours) |
| Brevo API key for this project | When the e-mail delivery starts; the lab runs the fake adapter until then |
| Add the environment setup script to the `identity` environment in Claude Code | Now; it installs the Superpowers plugin in every session |

## 19. Risks

- **The scope is very large for one person.** Mitigated by phases that each end usable, by the
  "nothing is pruned" register that keeps deferrals honest, and by the conformance suites that turn
  "done" into a green check rather than a judgement.
- **Protocol engines are security-critical code written in-house.** Mitigated by RFC 9700 defaults
  as floors, the OpenID Foundation's suites in CI, independent implementations as oracles, fuzzing,
  the threat model and a security review on every pull request.
- **XML signature handling in SAML** is the most attack-prone component (decision 19).
- **Cloud Run domain mapping is a Preview feature**, with a weekly certificate quota per domain; the
  lab accepts it, production does not depend on it.
- **The shared lab database** has a ceiling of 8 connections and sleeps at night; the design
  measures its own connection peak and schedules around the window.
- **Scale-to-zero means cold starts are sign-in latency**; the binary must start in under a second
  and unwrap tenant keys lazily.
- **Standards still moving** (OAuth 2.1, MCP, agent drafts): the product implements what is final or
  stable and tracks the rest in the register.
