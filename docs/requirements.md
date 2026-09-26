# Identity Platform: Requirements

> **Status:** living document and source of truth for product requirements, written in the inaugural
> design session of 2026-09-25/26. Changes to this file go through a pull request.
>
> **Conventions used in this document:**
> - Items without a marker are decisions made by the owner, directly or by approving a proposal in the
>   design session.
> - Items marked **(proposed)** were written from the session's reasoning without having been
>   discussed one by one; the owner confirms or changes them when reviewing this document.
> - "Beyond 1.0.0" items are scheduled, not cut: the owner's rule is that **nothing is pruned**
>   (section 2.4).
> - Items listed under [Open topics](#24-open-topics-for-design) need a spike or an external input
>   before they can be decided.
>
> **Related documents:** `docs/design.md` (how the product is built: architecture, data model,
> phases, lab cost), `docs/threat-model.md`, `docs/roadmap.md` (the ordered deliveries of the phase
> under way), and `docs/research/`: `standards.md` (state of every standard implemented),
> `market.md` (competitors and Go building blocks), `architecture-decisions.md` (where and why the
> design departs from the initial briefing), `licensing.md` (licence, contribution governance,
> external costs). Decisions below that came from those notes cite them.

---

## 1. Vision

- An **identity platform**: authentication, authorisation, user and organisation management,
  provisioning, migration, and identity for AI agents, built in **Go** in **one monorepo**
  (`github.com/aleogr/identity`), with products that compose a shared core.
- It covers the customer identity market's five categories — full CIAM, developer-first, the
  enterprise B2B layer, auth embedded in a platform, and open-source self-hosted / auth as a library —
  plus identity for AI agents and MCP, **from one core**. The categories differ by buyer, packaging
  and deployment, not by authentication logic (`docs/research/market.md`, section 1).
- It is **open source and public from the first commit**, and is meant to generate revenue through a
  managed cloud, commercial licences and long-term support, never by withholding security features
  (section 22).
- **Global first.** Differentiators for the Brazilian market (data residency in Brazil, CPF/CNPJ as
  identifiers, gov.br, WhatsApp as a channel, native LGPD workflows, the Open Finance Brasil profile)
  come after 1.0.0 — but the **extension points** that make each of them an adapter rather than a
  refactor exist from phase 1 (section 2.5).
- The owner's own projects will be the first clients. That imposes **no requirement** on this
  product: they consume it through its public interfaces and adapt to it. Nothing from other
  repositories is imported here.

### 1.1 Principles that govern every choice

1. **Maximum robustness.** Between options, the more robust, flexible, comprehensive, scalable and
   complete one wins, whatever the implementation cost. There is no rush: the product is released
   when complete.
2. **Secure by default, not configurable downwards.** In cryptography and security defaults, "more
   complete" never means "more options": few algorithms, all strong, and weak ones refused even when
   a customer asks. Maximum breadth of function, deployment and integration; minimum surface of
   insecure choices. The mechanism is **defaults and floors** (section 15.2).
3. **Robust in design, frugal in the lab.** The design allows the most robust deployment by
   configuration (highly available database, replicas, multi-region, keys in an HSM-backed KMS,
   minimum instances above zero). The lab runs the minimum, and **no code may depend on a resource
   that exists only in the expensive deployment**. Lowest possible recurring cost.
4. **Phases that end usable.** The owner works alone and the road to 1.0.0 is long; every phase
   leaves the system complete and usable within its scope.
5. **Standards are the product.** Where identity has consolidated practice and RFCs, the product
   follows them; any departure is stated and justified.

## 2. Scope

### 2.1 Forms of execution

The product is delivered in **three forms of execution** (`docs/research/architecture-decisions.md`,
decision 1):

- **Library** — the core in-process inside another Go application, with an implicit tenant. For Go
  backends only; TypeScript and other ecosystems are served by the server and SDKs (decision 4).
- **Server** — one binary: the authorisation server, the hosted login, the admin console and every
  API. Single-tenant or multi-tenant by configuration.
- **Cloud control plane** — what operating the server as a commercial multi-tenant service needs:
  self-service tenant sign-up, billing, domain provisioning, fleet operations. **Beyond 1.0.0**.

The market categories are **editions and profiles of the server** — presets of enabled capabilities,
the SDKs and UI kits a customer starts from, and their guides — not separate code.

### 2.2 What version 1.0.0 includes

Approved by the owner on 2026-09-26.

**Forms and surfaces**
- The server, single-tenant and multi-tenant; the library in Go, over PostgreSQL and SQLite.
- Hosted login; admin console; command-line interface; declarative configuration applied at start
  and reconciled; Terraform provider; SDKs in TypeScript, Go and Python; React UI components.

**Authentication**
- Passwords (argon2id; PBKDF2 in the FIPS profile), passkeys (WebAuthn Level 3), multi-factor
  authentication (TOTP, WebAuthn, e-mail codes), magic links, recovery with a security delay and a
  "this wasn't me" path, step-up authentication.

**Users, organisations and authorisation**
- Tenants, organisations, memberships, invitations, roles and permissions, role-based access control
  scoped to organisations, per-organisation single sign-on and domain verification.

**Protocols**
- OAuth 2.0 per RFC 9700, following the OAuth 2.1 profile; OpenID Connect provider passing the
  Basic, Config, Dynamic and Form Post profiles and the RP-Initiated, Front-Channel and Back-Channel
  logout profiles of the OpenID Foundation's suites.
- PAR, JAR, JARM, DPoP, mutual TLS, Rich Authorization Requests, token exchange, device
  authorisation grant, introspection, revocation, client ID metadata documents, dynamic client
  registration and its management, authorisation server and protected resource metadata.
- **FAPI 2.0** Security Profile and Message Signing.
- **SAML 2.0** identity provider, service provider and brokering.
- **SCIM 2.0** server and client, with the SCIM events profile (RFC 9967).
- **Shared Signals Framework** with CAEP and RISC, as transmitter and receiver.

**Tokens and keys**
- Opaque and JWT access tokens, per-tenant JWKS with automatic rotation, refresh token rotation with
  reuse detection, sender-constrained tokens, short lifetimes by default.

**Federation**
- Social login and enterprise SSO through brokering: Google, Microsoft, Apple, GitHub, Okta, and any
  OIDC or SAML provider.

**Agents and MCP**
- Authorisation server for remote MCP servers per the MCP authorisation specification of 2026-07-28,
  including the Enterprise-Managed Authorization extension (ID-JAG) in both roles; token vault for
  third-party tokens held on users' behalf; delegation through token exchange.

**Extensibility and events**
- Hooks at three levels — Go interfaces, WebAssembly plugins, webhooks — under one contract; domain
  events through an outbox, delivered by webhook.

**Audit, privacy and migration**
- Append-only, hash-chained audit log with per-user encryption keys and crypto-shredding.
- Self-service data export and account deletion.
- Legacy password hash verifiers with transparent rehash; importers for Auth0, Firebase, Keycloak and
  declarative CSV/JSON; lazy migration against Cognito; coexistence and rollback; complete export in
  an open format.

**Operation and supply chain**
- en-US and pt-BR throughout; the FIPS deployment profile; signed releases, SBOM, SLSA provenance,
  reproducible builds.

**Quality bar for 1.0.0** (section 23): every conformance suite green in CI, the threat model
current, a complete internal security review, the load test of section 18 passed.

### 2.3 Foundations that are expensive to retrofit

Designed from phase 1 even where delivered later: multi-tenancy with row-level security on every
table, per-tenant keys, the hook contract, the event schema, the audit log, internationalisation,
the "everything is API" rule, the defaults-and-floors configuration model, the extension points of
section 2.5, and the supply-chain practices of section 21.

### 2.4 Beyond 1.0.0

The owner's rule: **nothing is pruned.** Every item left out of 1.0.0 is recorded here with its
target, the extension point that exists for it from phase 1, and why it is not in 1.0.0. Targets are
indicative; `docs/design.md` places them in phases.

| Item | Target | Extension point ready from phase 1 | Why not in 1.0.0 |
|---|---|---|---|
| Cloud control plane: self-service tenant sign-up, billing, run-time domain provisioning, fleet operations, SLA | First post-1.0.0 phase; precedes the commercial launch of the managed cloud | Multi-tenancy in the server; `HostProvisioner` port; tenant lifecycle API | Operating a commercial service is a business launch, not product completeness; it has no user until there are external customers |
| Relationship-based authorisation (Zanzibar-style, OpenFGA modelling language) | 1.x | `Authorizer` interface through which every permission check passes | A product in its own right; RBAC by organisation covers 1.0.0's users (`architecture-decisions.md`, decision 10) |
| Third-party management API compatibility layers (`compat/`) | 1.x, driven by migration demand | Management API defined in OpenAPI over a stable service layer | Moving target, trademark exposure; standards already give protocol-level compatibility (decision 3) |
| Native mobile SDKs (Swift, Kotlin) | 1.x | OpenAPI from which SDKs are generated; standard OAuth native-app flows (RFC 8252) | The TypeScript, Go and Python SDKs cover 1.0.0's clients |
| Remaining importers (Cognito export, Okta, Entra External ID, Supabase Auth, Clerk, WorkOS, Better Auth, FusionAuth, Zitadel, LDAP/AD) and lazy connectors for the same | 1.x | Importer and lazy-migration interfaces proved by 1.0.0's set | 1.0.0's set proves every shape of source; the rest are repetitions |
| Other relational databases (MySQL, SQL Server) | 1.x | Per-aggregate storage interfaces and the shared adapter test suite | PostgreSQL covers the server; SQLite the library |
| OpenID Federation 1.0/1.1 | 1.x | Client registration interface with pluggable trust establishment | Relevant to specific federations (education, government) |
| OpenID for Verifiable Credentials (OID4VCI, OID4VP, HAIP) | 1.x or 2.0 | Credential and authentication-method model open to new credential types | A distinct product surface (wallets) |
| OIDC Session Management (iframe-based) | 1.x | Session model already exposes the state it needs | Unreliable under current browser cookie restrictions; the three logout profiles in 1.0.0 cover the need |
| Transaction tokens, AAuth, WIMSE workload credentials, identity chaining | As each becomes final | Token exchange policies; pluggable workload credential verifiers | Still drafts (`docs/research/standards.md`, section 6) |
| SMS and push as delivery channels and second factors | 1.x | `adapters/delivery` port | SMS is a weak factor (SIM swap) with per-message cost; push needs mobile SDKs |
| Brazilian differentiators (section 2.5) | After 1.0.0 | Section 2.5 | Global first |
| OpenID Connect Implicit and Hybrid certification profiles | 1.x, if a customer needs them | Response types are data in the authorisation endpoint | RFC 9700 discourages the implicit grant; hybrid is a legacy flow |

### 2.5 Extension points for the Brazilian differentiators

Each exists in phase 1 as an interface with one generic implementation, so the Brazilian version is
an adapter:

- **Identifier types** — a registry of identifier kinds (e-mail, phone, username) with per-kind
  normalisation and validation. CPF and CNPJ become two more kinds, with check-digit validation.
- **Data residency** — every tenant carries a data region; storage and key management resolve
  through it. Brazil becomes a region.
- **Identity providers** — gov.br is an OpenID Connect provider with its own assurance levels; it is
  an adapter of the federation port with a mapping of its levels to `acr` values.
- **Delivery channels** — WhatsApp is an adapter of the delivery port.
- **Privacy regime** — consent records, legal bases and retention policies are data per tenant; LGPD
  becomes a preset of them.
- **Open Finance Brasil** — a FAPI profile assembled from 1.0.0's building blocks (PAR, JAR, JARM,
  mTLS, DPoP, dynamic registration with software statements, CIBA added then).

### 2.6 Out of scope

Workforce IAM (the classic Okta), privileged access management, identity governance and
administration, mobile device management.

## 3. Working agreements

- **Language:** Claude Code always talks to the owner in **Portuguese**. All versioned content is
  written in **English**: code, comments, commit messages, pull request descriptions and
  documentation. The only exception is translation files for other languages.
- **One topic at a time.** Before writing a document or a design, Claude Code asks the questions
  whose answers change it, one at a time, and does not presume.
- **Methodology:** the **Superpowers** plugin. The project documents (`requirements.md`, `design.md`,
  `threat-model.md`, `roadmap.md`) are written through its brainstorming process; every roadmap
  delivery goes through a written spec in `docs/superpowers/specs/`, approved by the owner, then a
  written plan in `docs/superpowers/plans/`, approved by the owner, then implementation with test-driven
  development and verification before completion.
- **Branches and pull requests:** each topic is developed in its own branch and merged through a pull
  request, always opened by Claude Code.
- **Pull request follow-through:** Claude Code watches every pull request it opens until it is merged
  or closed, and drives it to a green, mergeable state without the owner having to report failures. A
  failing check is diagnosed from the job's own log before any fix is proposed. A fix is validated
  with the repository's checks before it is pushed; when a session cannot run a check, the pull
  request says so rather than claiming a verification that did not happen.
- **Green is never bought:** no test is skipped, disabled or quarantined, no job is made
  non-blocking, no severity threshold is lowered, no commit bypasses the checks. When the cause of a
  failure is ambiguous, or the fix would change a design decision or widen the pull request, Claude
  Code stops and asks the owner.
- **Merge authority:** Claude Code opens pull requests, pushes to them and answers review comments;
  it never approves and never merges. The owner merges.
- **Manual steps:** when the owner must act outside a session (GitHub, GCP Cloud Shell, Cloudflare,
  any provider), instructions are given **one at a time**, as command blocks or clicks. The owner
  executes, reports the result, and only then receives the next instruction.
- **Confirmation before acting:** when the owner asks to see something before an action, Claude Code
  shows it and waits for explicit confirmation.
- **Secrets:** the repository is public. No secret may ever be committed. Model identifiers appear
  only in the attribution trailer of commit messages and pull request descriptions.
- **Best practice:** follow the domain's consolidated practice and RFCs; when a choice departs from
  them, say so and why.
- **Definition of done:** every user-facing text exists in **both en-US and pt-BR** through
  translation keys, tests pass, and there is verification evidence (test output, screenshots or
  equivalent).
- **UI principles:** immediate visual feedback when an element is pressed; the operating system's
  reduced-motion preference is respected; animations only when they serve a purpose.
- **Environment:** the owner has no local development environment. Claude Code works exclusively
  through Claude Code on the web; the environment's setup script installs the Superpowers plugin and,
  once the repository has them, the end-to-end and Terraform dependencies.

## 4. Tenancy and vocabulary

- **Tenant** — a customer of the platform: the unit of isolation, keys, configuration and billing.
- **Organization** — a customer of the customer: a grouping of users inside the tenant's
  application, with members, roles and its own SSO. A user belongs to zero or more organisations.
- `spec/` fixes the domain model (Tenant, Organization, User, Identity, Credential, Session,
  Membership, Client, Grant, Token) and keeps a **glossary** with competitors' synonyms (Keycloak's
  realm, Auth0's tenant, Zitadel's instance and organization, Cognito's user pool).
- **The data model is always multi-tenant.** Every table carries `tenant_id`; row-level security
  covers every table; keys are per tenant; one schema serves every product.
- The **self-hosted server** starts with one tenant created at bootstrap; creating further tenants is
  a capability that is **off by default**. The **library** uses an implicit tenant, through the same
  code path (`architecture-decisions.md`, decision 17).
- **Issuer and hosts:** each tenant's issuer is an absolute URL **without a path** — a host under the
  platform's domain or a customer's own domain. The server resolves the tenant from the request's
  host. Cookies and the WebAuthn relying party ID are per host, which is why path-based issuers are
  not offered (decision 5).
- **Lab hosts:** the platform answers at `identity.lab.aleogr.dev`; hosted tenants at
  `<tenant>.identity.lab.aleogr.dev` or a customer domain. Cloud Run domain mapping accepts no
  wildcard, so each host is a mapping and a certificate (up to 24 hours to issue), declared in
  Terraform from a list. Run-time host provisioning belongs to the control plane (section 2.4).

## 5. Actors and administration (proposed)

| Actor | What they do |
|---|---|
| **Operator** | Runs a deployment: its configuration, its floors above the product's, tenant creation when enabled. In the lab and the managed cloud, the owner |
| **Tenant administrator** | Configures a tenant through the console, API, CLI, declarative file or Terraform: applications, connections, policies, branding, hooks. Roles within the tenant: owner, administrator, and read-only auditor, plus custom roles |
| **Organisation administrator** | Manages one organisation inside a tenant: members, invitations, organisation roles, its SSO and SCIM connection, its verified domains — through a delegated, white-label admin portal |
| **End user** | Signs up, signs in, manages their own credentials, sessions, devices, data export and deletion |
| **Developer** | Registers clients, integrates SDKs and components, writes hooks and plugins |
| **Agent** | Software acting for a user or on its own behalf, holding delegated tokens under explicit, auditable grants |

The first operator and tenant owner are created by a **single-use bootstrap token**, printed once at
first start; the server refuses to serve administration before bootstrap completes.

## 6. Authentication

### 6.1 Passwords

Defaults follow **NIST SP 800-63B-4**; floors are fixed in code (section 15.2). Approved by the owner
on 2026-09-26.

| Rule | Default | Editable per tenant | Floor |
|---|---|---|---|
| Minimum length, password as the only factor | 15 | Yes | **12** |
| Minimum length, tenant requiring MFA from every user | 8 | Yes | **8** |
| Maximum accepted length | 64 or more | Upwards only | **64** |
| Composition rules | Off | Freely | — |
| Periodic change every N days | Off | Freely | — |
| History (no reuse of the last N) | Off | Yes | — |
| Blocklist: breached, common, contextual (application name, user's identifiers) | On | Tenant terms may be added | **Always on** |
| Forced change on evidence of compromise | On | No | **Always on** |

When composition rules or periodic change are turned on, the console shows NIST's recommendation
against them, without blocking. Breached passwords are checked by k-anonymity against Pwned Passwords
(only the first five characters of the hash leave the server) at sign-up and change; an offline
adapter with a local dataset serves deployments without internet access; when the service does not
answer, the check is skipped and the skip recorded, because it is an additional layer and never the
only one.

Password hashing: argon2id with parameters set by a benchmark recorded for the deployment's CPU;
PBKDF2-HMAC-SHA-512 in the FIPS profile. Legacy algorithms exist only as **verifiers** for migrated
users, followed by an immediate rehash; no code path writes a legacy hash (section 13).

### 6.2 Passkeys and WebAuthn

WebAuthn Level 3: registration and authentication ceremonies, conditional mediation (autofill) and
conditional create (silent upgrade from password), the Signal API, Related Origin Requests for
tenants with several domains, attestation verification where a tenant requires specific
authenticators. The relying party ID is the tenant's host or its registrable domain.

### 6.3 Multi-factor authentication and step-up

- Factors: TOTP, WebAuthn (security keys and platform authenticators), e-mail codes.
- Policy per tenant and per user type (for example: administrators only with TOTP or WebAuthn, never
  an e-mail code, because e-mail is the recovery channel); per application; and by risk.
- Recovery codes generated when TOTP or WebAuthn is enrolled.
- Step-up through `acr_values`, `max_age` and RFC 9470 challenges from resource servers.

### 6.4 Magic links and e-mail codes

Single-use, short-lived, bound to the browser that requested them where possible; the same
enumeration-resistant responses as every other path.

### 6.5 Recovery

By e-mail with a **security delay** (default 24 hours, editable, with a floor), notification on every
channel, and a **"this wasn't me"** link that cancels the recovery and revokes sessions. Revocation of
all sessions on completion. Administrator accounts are recovered only by another administrator
resetting the factor, audited.

### 6.6 Abuse resistance

All defaults editable in the console; floors where relaxing would open a hole:
- **Rate limits** per IP, per identifier and per tenant; **progressive delay** rather than account
  lockout, which would be a denial-of-service vector; detection of many identifiers tried from one
  origin (credential stuffing).
- **Bot protection:** a built-in **proof-of-work challenge** (no third party, no tracking) enabled by
  risk; adapters for Cloudflare Turnstile, hCaptcha and reCAPTCHA per tenant.
- **Account enumeration:** identical responses and timings for existing and non-existing accounts at
  sign-in, sign-up and recovery. **Floor: cannot be turned off.**

### 6.7 Sessions

Server-side sessions with opaque tokens, only their hash stored; idle and absolute timeouts per
tenant within ceilings; revocation of every session on password change, factor change and recovery
(**floor**); a list of active sessions and devices for the user, with revocation; back-channel and
front-channel logout; CAEP session-revoked events sent and received.

### 6.8 Security notifications

E-mail to the user on a new device, password change, e-mail change, factor added or removed, and
recovery, each with the "this wasn't me" link. Templates per tenant, in both languages.

## 7. Users, organisations and authorisation

- **Users:** profile with standard OIDC claims and tenant-defined custom attributes (typed, validated,
  with per-attribute visibility and editability); several identities per user (password, passkeys,
  social and enterprise federations) with account linking under explicit rules; blocking,
  impersonation by administrators (audited, visible to the user).
- **Identifiers:** a registry of identifier kinds with normalisation and validation per kind
  (section 2.5); e-mail and phone verification.
- **Organisations:** members, invitations, organisation roles, verified domains with automatic
  membership (just-in-time or by invitation), per-organisation SSO (SAML or OIDC) and SCIM, an
  organisation switcher in the hosted login and in tokens.
- **Authorisation (1.0.0):** roles and permissions per tenant and per organisation, custom roles,
  permissions carried in tokens and checkable through the API; every check passes through the
  `Authorizer` interface that relationship-based authorisation will implement later (section 2.4).

## 8. Protocols

- **OAuth 2.0 / OpenID Connect:** the full list of section 2.2. Defaults are RFC 9700's
  requirements, and they are floors: PKCE with `S256` for every client, no implicit grant, no
  resource owner password grant, exact redirect URI matching (loopback port exception per RFC 8252),
  refresh tokens of public clients sender-constrained or rotated, the `iss` authorisation response
  parameter always returned.
- **Conformance:** the OpenID Foundation's suites run in CI against the server from phase 1 and must
  be green for every profile in section 2.2 before 1.0.0.
- **SAML 2.0:** identity provider and service provider, HTTP-Redirect, HTTP-POST and Artifact
  bindings, signed and encrypted assertions, metadata import and publication, IdP-initiated sign-in
  off by default, brokering in both directions. Interoperability verified against Okta, Entra ID,
  ADFS and Shibboleth test instances before 1.0.0.
- **SCIM 2.0:** server (directories push in) and client (the product pushes to downstream
  applications), with filtering, bulk operations, and the events profile (RFC 9967).
- **Shared Signals:** transmitter and receiver for CAEP and RISC events.

## 9. Tokens and keys

- **Formats:** opaque tokens with introspection (RFC 7662) and JWT access tokens (RFC 9068), chosen
  per client or resource.
- **Signing keys** per tenant, with automatic rotation and publication of the next key before use.
  Signing happens **in memory**: the KMS wraps each tenant's keys and never signs per token; keys are
  unwrapped lazily per tenant and cached. A **KMS/HSM signer** is available per tenant for
  regulations that require non-exportable keys (`architecture-decisions.md`, decision 14).
- **Lifetimes** short by default, editable within ceilings; refresh token rotation with reuse
  detection that revokes the whole family.
- **Sender-constraining:** DPoP and mutual TLS certificate binding.
- **Revocation:** RFC 7009, back-channel logout, and CAEP/SSF events.
- **Delegation:** RFC 8693 token exchange is the primitive for impersonation, delegation and agents.

## 10. Federation and social login

- One **identity provider adapter** per provider, reused by social login and enterprise SSO: Google,
  Microsoft, Apple, GitHub, Okta, and generic OIDC and SAML.
- Attribute mapping declared per connection; just-in-time provisioning; account linking rules;
  home-realm discovery by e-mail domain.
- **Coexistence:** brokering to a legacy identity provider during migration (section 13).

## 11. Agents and MCP

- The server is an **authorisation server for remote MCP servers**: client ID metadata documents
  (preferred), dynamic client registration (kept for compatibility, with `application_type`), PKCE,
  resource indicators with audience validation, `iss` always returned, RFC 9728 metadata published
  for the product's own APIs and helpers for customers' MCP servers to publish theirs.
- **Enterprise-Managed Authorization:** the server issues ID-JAGs as an enterprise identity provider
  and accepts them as an MCP server's authorisation server.
- **Token vault:** third-party OAuth tokens obtained with the user's consent, stored encrypted,
  refreshed, and released to authorised agents through token exchange, with every release audited.
- **Fine-grained consent** through Rich Authorization Requests; agent grants visible and revocable
  by the user.

## 12. Extensibility and events

### 12.1 The hook contract

Fixed in `spec/`:
- A hook is a **pure function over versioned JSON**; the mutations allowed are declared per extension
  point; hooks have no database access.
- **Fail-open or fail-closed is a property of the extension point**, not a configuration.
- Execution order is deterministic per tenant; each execution is audited with the module's hash and
  the contract version.
- The ABI is versioned within SemVer; a test harness is published with the plugin kits.
- Only what must **decide** — deny, enrich, require MFA — is a synchronous hook. Everything else (CRM,
  analytics, notifications) consumes events.

Extension points include `PreRegistration`, `PostRegistration`, `PreAuthentication`,
`PostAuthentication`, `TokenClaims`, `PasswordPolicy`, `MFAPolicy`, `RiskDecision`, `PreUserUpdate`,
`PreTokenExchange`; the complete list and each point's allowed mutations are in `spec/`.

### 12.2 Three levels, one contract

1. **Go interfaces in-process** — the library's mechanism and every first-party extension.
2. **WebAssembly plugins** (wazero, with the project's own ABI): real sandbox, memory limit, timeout
   per call, host functions under an allowlist declared in a manifest (`fetch` only to declared
   domains, per-tenant `kv`, `log`, `secret.get`). The same `.wasm` runs in every hostable form.
   Plugin kits for Go, Rust and TypeScript.
3. **Webhooks** — a built-in level-1 plugin: signed POST (HMAC with timestamp and replay protection),
   short timeout. Never the only path.

### 12.3 Events

Domain events written to an outbox in the same transaction as the change; delivered by signed
webhooks with retries and a dead-letter view; the event schema is versioned in `spec/`.

## 13. Migration and export

- **Legacy hash verifiers**, each with real test vectors: bcrypt; scrypt, including Firebase's
  variant; PBKDF2 with SHA-1, SHA-256 and SHA-512; argon2i and argon2id; salted SHA in every order;
  MD5-crypt and SHA-crypt; Django; Rails/Devise; Laravel; ASP.NET Identity v2 and v3; Keycloak;
  phpass; LDAP `{SSHA}`. Verified once at the next sign-in, then rehashed.
- **Importers** — idempotent, with dry run, a rejection report and resumption: Auth0, Firebase,
  Keycloak and declarative CSV/JSON in 1.0.0; the rest in section 2.4.
- **Lazy migration** as a `PreAuthentication` hook, with a Cognito connector in 1.0.0, an expiry
  policy and a report of users still pending.
- **Coexistence:** federation with the old identity provider through brokering; migration by cohort
  (organisation, domain, percentage); **rollback** by exporting back in the old system's format.
- **Export** of everything — users with hashes, organisations, configuration, clients, protected
  keys, audit log — in an open, documented format, and in the formats of competitors that accept
  imports.

## 14. Audit and privacy

- **Audit log:** append-only, tamper-evident through a hash chain verified by a scheduled job;
  personal fields encrypted with a per-user key wrapped by the KMS; erasure by destroying the key
  (crypto-shredding). Every administrative change records who, when, the previous and the new value.
- **Retention:** audit log kept one year by default, editable per tenant upwards within the
  deployment's limit; expired sessions deleted after 30 days.
- **Self-service:** users export their own data and delete their account; deletion destroys the
  personal key.

## 15. Configuration and APIs

### 15.1 Everything is API

The console, the CLI, the declarative file (applied at start and reconciled) and the Terraform
provider are **four facades of the same API**. Nothing exists that only the console can do.
**GitOps for identity** is a first-class capability: a tenant's full configuration can live in a
repository and be applied by CI.

### 15.2 Defaults and floors

Approved by the owner on 2026-09-26. Every security-relevant setting has:
- a **default**, editable per tenant through every facade of section 15.1, with each change audited;
- where relaxing it would open a hole, a **floor fixed in code** that no configuration crosses;
- optionally, a **stricter floor** set by the deployment's operator for all its tenants.

Settings whose change is not a weakening (composition rules, periodic password change) have no floor.
The console explains recommendations (NIST, RFC 9700) where a tenant departs from them, without
blocking. **Invalid configuration refuses to start**, naming the setting and the value received.

## 16. User interface and internationalisation

- **Hosted login** (templ + HTMX): works without JavaScript except for passkeys; strict content
  security policy; per-tenant branding (logo, colours, texts, custom domain).
- **UI components** (React): renderers of the same server-side flow state as the hosted login, so
  logic lives once (`architecture-decisions.md`, decision 18).
- **Admin console** (templ + HTMX) over the same service layer as the management API; the delegated
  organisation admin portal is part of it.
- **Languages:** en-US and pt-BR for every user-facing text through translation keys; tenants may
  override any text per key and add languages.
- **Accessibility:** WCAG 2.2 AA for the hosted login, the components and the console, verified by
  automated checks and manual review.
- **Browsers:** the two latest versions of the major browsers.

## 17. Cryptography

- Only the Go standard library and `golang.org/x/crypto`.
- **Signatures:** ES256 (default), EdDSA (Ed25519), PS256, RS256 — RS256 because OpenID Connect
  requires it, PS256 and ES256 because FAPI 2.0 requires one of them; RSA keys of 3072 bits by
  default, never below 2048.
- **Refused, never configurable:** `none`; HMAC for tokens the server issues; RSA below 2048 bits;
  SHA-1 in any signature, SAML included; RSA PKCS#1 v1.5 encryption.
- Secrets compared in constant time, enforced by a lint rule.
- **FIPS profile:** a named deployment profile that switches password hashing to PBKDF2, restricts
  algorithms to the approved set, runs Go's FIPS 140-3 module, and refuses to start if any setting
  contradicts it.

## 18. Non-functional targets

Approved by the owner on 2026-09-26. Scale and latency are orders of magnitude used to size the
design and the load test, not contractual commitments.

| Topic | Target |
|---|---|
| Reference scale per deployment | 10,000 tenants; 10 million users |
| Throughput per instance | 1,000 sign-ins per second; 5,000 token validations per second |
| Latency, p99, warm instance, excluding password hashing | Token endpoint under 50 ms; introspection under 20 ms |
| Cold start | The binary serves requests within 1 second of starting |
| Availability | No SLA in 1.0.0; the design allows 99.99% by configuration (replicas, highly available database, active-passive multi-region) |
| Disaster recovery, robust deployment | RPO of 5 minutes or less through point-in-time recovery; RTO of 1 hour or less; restore test documented |
| Accessibility | WCAG 2.2 AA |

## 19. Deployment and distribution

- **Server targets:** static binaries for Linux and macOS on amd64 and arm64; a signed multi-arch OCI
  image; a reference Docker Compose file; a Helm chart for Kubernetes; reference Terraform modules
  for GCP. Cloud Run is the lab's platform, not a requirement of the product.
- **Releases** are built by the open-source **GoReleaser**: binaries, checksums, SBOM, Sigstore
  signatures, SLSA provenance, the GitHub Release. The public image in `ghcr.io` is **the same image,
  by digest, that ran in the lab**, copied and signed, never rebuilt; CI rebuilds from the tag and
  requires an identical digest, which proves the build reproducible.
- **Tags:** a plain `vX.Y.Z` tag is a **product release** (server, CLI, image), which is why the root
  module is the product's; library modules are tagged with their path prefix (`core/vX.Y.Z`) and
  released by the tag alone, as Go modules are. GoReleaser's paid monorepo feature is not needed.
- **Versioning:** SemVer, `0.y.z` during development, `1.0.0` for the complete product; a written API
  stability and deprecation policy; an LTS line for the self-hosted server after 1.0.0.
- **Flow:**
  - **Pull request:** CI runs every check; nothing is deployed; evidence lives in the pull request.
  - **Merge into `main`:** the image is built once, pushed, migrations run as a job, the lab is
    deployed, the end-to-end suite runs against it. Visible at `identity.lab.aleogr.dev` within
    minutes.
  - **Release, only when the owner asks:** Claude Code pushes the tag; the release workflow publishes
    the artefacts; once a production environment exists, it promotes the same digest there.
  - **Production** does not exist in the early phases; it is created in a later phase, before 1.0.0,
    when the owner's projects are to run on the product.
- **Build identifier** from `git describe`, in the start-up log, the console footer and an
  operator-only version endpoint.

## 20. Infrastructure and cost (lab)

- **GCP**, one project for this repository (Cloud Run, Secret Manager, Cloud KMS, Cloud Tasks and
  Scheduler, Artifact Registry), region `us-central1`.
- **Database:** a database named `identity` in the shared lab instance `lab-postgres` (PostgreSQL 16),
  in the shared project `aleogr-lab-shared-dacd` declared in `aleogr/lab`. This repository's Terraform
  receives the project and instance as variables and creates only the database, its roles and
  permissions; it never manages the instance. Connection across projects through IAM and the Cloud
  SQL connector.
- **Accepted lab limits** (owner, 2026-09-25): at most **8 connections** (the instance has 22 usable
  and the marketplace's ceiling is 14), and the instance **sleeps Monday to Thursday, 22:00 to 07:30
  local time**; scheduled work runs inside the awake window.
- **No load balancer** in the lab: Cloud Run domain mapping (a Preview feature) with Cloudflare's
  free plan in "DNS only" mode; one mapping and certificate per host.
- **Terraform is applied only by CI**; GitHub authenticates to GCP through Workload Identity
  Federation restricted to this repository.
- **Lowest possible recurring cost**; the lab's cost is itemised in `docs/design.md`.

## 21. Development workflow and quality

- **CI on every pull request:** build; `go vet`, `staticcheck`, `golangci-lint`, `gosec`,
  `govulncheck`; secret detection; unit tests with the race detector; integration tests against a real
  PostgreSQL; end-to-end tests with Playwright in Python; the conformance suites; image scanning.
- `make check`, `make test`, `make integration`, `make e2e`; Go tools pinned as `tool` directives;
  integration tests against a PostgreSQL started in the session (the Docker daemon does not run in a
  session); fakes of every external provider in tests.
- **Fuzzing** of every parser (JWT, SAML XML, WebAuthn CBOR, SCIM filters, CIMD documents)
  continuously in CI.
- **Supply chain:** Dependabot; signed releases; SBOM; SLSA provenance; reproducible builds.
- **Security review** on every pull request against the domain checklist; `docs/threat-model.md`
  kept current.
- **Manual steps** performed by the owner are recorded in `docs/infrastructure.md` so they can be
  repeated for another environment.

## 22. Licensing and contributions

Decided by the owner on 2026-09-25 (`docs/research/licensing.md`):
- **Apache-2.0:** `spec/`, `core/`, `adapters/`, `conformance/`, the SDKs, the plugin kits, the
  library.
- **AGPL-3.0-only or a commercial licence:** the server, `ui/`, the cloud control plane.
- **Free:** every capability; no security feature is paid; no security fix reaches paying customers
  first. **Paid:** the managed cloud, the commercial licence, long-term support.
- **A contributor licence agreement** (licence grant, not assignment) from the first external
  contributor; until it exists, external contributions are not merged.

## 23. Milestones and external dependencies

- **1.0.0** — the complete product of section 2.2; every conformance suite green in CI; the threat
  model current; a complete internal security review; the load test passed. No external cost.
- **Commercial launch** — when the owner decides and funds it (owner, 2026-09-26): paid OpenID
  certification (Connect and FAPI 2.0), an external security audit, legal review of the CLA,
  commercial licence and cloud terms, and optionally OpenID Foundation membership. None of these is
  needed while the owner's projects are the only clients.

| Dependency | Recommended moment |
|---|---|
| Brand, with a formal collision check (GitHub, pkg.go.dev, npm, trademarks) | Before the first public npm package, image name change or Terraform provider |
| CLA text and tooling | Before the first external contribution is merged |
| Production GCP project and domains | The production phase, before 1.0.0 |
| Paid certification, external audit, legal review of the commercial terms | Commercial launch |

## 24. Open topics for design

| Topic | What closes it |
|---|---|
| Run-time host provisioning for the control plane: Certificate Manager behind a load balancer (about US$ 18 per month plus certificates) or Cloudflare for SaaS (100 hostnames free, but a Worker needed to reach Cloud Run on the free plan) | A spike before the control plane's phase |
| Production domains and the move off Cloud Run domain mapping (Preview) | The same spike, before the production phase |
| mutual TLS on Cloud Run, where TLS terminates at Google's front end | A spike in the phase that delivers FAPI 2.0; the lab may be unable to exercise it, and the design says how it is tested instead |
| The brand | Owner's choice, with the collision check of section 23 |
