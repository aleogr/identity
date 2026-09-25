# Research: architecture decisions and the reasoning behind them

> **Status:** research note written in the inaugural design session of 2026-09-25. The project began
> from a briefing prepared in another conversation, whose architecture was explicitly a set of
> *suggestions*. This note records, for each point where the design departs from or sharpens that
> briefing, the alternatives, the reasoning and the owner's decision. `docs/design.md` states the
> resulting design; this note explains why.
>
> **How to read the status line of each decision:**
> - **Accepted by the owner** — discussed and agreed in the session; not to be reopened without a
>   new fact.
> - **Proposed** — the recommendation presented in the session without objection, carried into
>   `docs/design.md`, and confirmed or changed when the owner reviews that document.
>
> Sources for market and standards facts are in `docs/research/market.md` and
> `docs/research/standards.md`.

---

## Governing rules

Three rules set by the owner shape every decision below:

1. **Maximum robustness, no rush.** When options differ in completeness, the more complete one wins;
   the product launches only when complete.
2. **Security defaults are not configurable downwards.** Completeness means breadth of function and
   deployment, never a wider menu of weak choices.
3. **Nothing is pruned.** Whatever is left out of `1.0.0` is recorded in the "beyond 1.0.0" register
   of `docs/requirements.md`, with its target, the extension point that exists for it from phase 1,
   and the reason it is not in `1.0.0`. A deferral is a schedule, not a cut.

## 1. Three forms of execution instead of seven products

**Status:** accepted by the owner.

**Briefing:** seven thin product directories (`lib-auth`, `selfhosted`, `enterprise-layer`,
`devfirst`, `ciam`, `embedded`, `agent-gateway`) composing a shared core.

**Problem:** the five market categories differ by buyer and packaging, but the code only changes
with the *form of execution* (`docs/research/market.md`, section 1). Seven product directories
would mean seven builds, seven configuration surfaces, seven runs of the conformance suites and seven
places where a security behaviour can diverge — for products whose server-side logic is the same.

**Decision:** three forms of execution:
- **Library** — the core in-process in someone else's Go application.
- **Server** — one binary that is the authorisation server, the login, the admin console and the
  APIs; single-tenant or multi-tenant by configuration.
- **Cloud control plane** — what operating the server as a multi-tenant commercial service needs:
  tenant sign-up, billing, domain provisioning, fleet operations.

The market categories become **editions and profiles of the server**, documented as such: which
capabilities are on by default, which SDKs and UI kits and which guides a customer of that category
starts from. "Enterprise layer", "developer-first", "CIAM", "embedded" and "agent gateway" are
product pages and presets, not code directories.

## 2. The server before the library

**Status:** accepted by the owner.

**Briefing:** `lib-auth` first, as the thinnest shell and something usable early; then `selfhosted`.

**Reasons to invert:**
- Every infrastructure decision already taken — Cloud Run, the `identity.lab.aleogr.dev` host, the
  database in the shared lab instance — is infrastructure for a server. A library has nothing to
  deploy there.
- The conformance that matters, the OpenID Foundation's suites, tests a provider reachable over the
  network.
- The owner's own projects, the first clients, will consume the product over OIDC, not by embedding
  a library that shares their database.
- Publishing the library's Go API first would freeze a public surface before the server's
  generalisation (deployment, migrations, multi-tenancy, administration) had tested it.

**Decision:** phase 1 ends with a **single-tenant server deployed in the lab** that passes the OIDC
Basic and Config profiles of the official suite, with sign-up, sign-in with password and passkey,
and sessions. The server reaches the core **only through the same public API the library will
expose**, which keeps the core free of I/O from the first commit. Phase 2 packages that API as the
library, with the implicit tenant, the SQLite adapter and examples.

## 3. Third-party API compatibility layers after 1.0.0

**Status:** accepted by the owner (under the "nothing is pruned" rule: scheduled, not dropped).

**Briefing:** `compat/`, versioned and pinned layers emulating competitors' management APIs.

**Reasons not to put it in 1.0.0:**
- It chases a moving target: undocumented behaviour of vendors that change without notice, and their
  bugs.
- The compatibility customers need most already comes from **standards**: an application on OIDC,
  SAML or SCIM changes identity provider by changing configuration.
- The value of migration is in the password hash verifiers, importers, lazy migration, coexistence
  and export, all of which are in 1.0.0 (decision 12).
- Trademarks: product names of third parties cannot name modules or products; "compatible with X"
  wording is the only acceptable form.

**Decision:** `compat/` is in the "beyond 1.0.0" register. Its extension point in phase 1 is the
management API being defined as an OpenAPI document with a stable internal service layer beneath,
so that a compatibility facade is another adapter over the same services.

## 4. The library is for Go; TypeScript is served by the server

**Status:** accepted by the owner.

**Briefing:** asked whether `lib-auth` should have a first-class TypeScript surface close to Better
Auth's.

**Problem:** a Go library does not run inside Node. A TypeScript `lib-auth` would mean
reimplementing the core in TypeScript, which contradicts the closed decision that Go is the only
core language.

**Decision:** the library is a product for **Go backends**. TypeScript applications are served by
the server plus a TypeScript SDK and UI components; the SDK's ergonomics follow what the TypeScript
ecosystem expects (Better Auth's is the reference), talking to the server over its APIs. This is
recorded rather than deferred: the need is met, without a second implementation of the core.

## 5. The issuer is host-based

**Status:** proposed.

**Briefing:** asked whether a tenant's issuer should be host-based (`https://acme.identity.example`)
or path-based (`https://identity.example/t/acme`), noting that path-based eases single-tenant
self-hosting.

**The deciding argument is security, not convention.** Two browser security boundaries are per host:
- The **WebAuthn relying party ID** is a registrable domain suffix of the origin. Tenants sharing one
  host share one relying party ID, so the browser's passkey autofill offers every tenant's
  credentials on every tenant's login page — a cross-customer privacy leak — and a passkey cannot be
  scoped to one customer.
- **Cookies** (`__Host-` prefixed, which this product uses) are scoped to the host. Tenants sharing a
  host share a cookie jar, which opens session confusion across tenants.

Path-based issuers bring no benefit to single-tenant self-hosting either: that deployment's issuer is
its own host, with no path.

**Decision:** each tenant's issuer is a configured absolute URL **without a path**: a host under the
platform's domain (`<tenant>.identity.lab.aleogr.dev` in the lab) or a customer's own domain. The
server resolves the tenant from the host of the request, as the marketplace resolves its
marketplaces. Passkeys for a tenant with several domains use WebAuthn Related Origin Requests
(`docs/research/standards.md`, section 7).

## 6. Hosts without a wildcard

**Status:** proposed.

**Facts** (`docs/research/standards.md` and the infrastructure research of 2026-09-25):
- Cloud Run domain mapping is still **Preview**, not production-ready, and supports **no wildcard**.
  Certificate issuance is limited to 50 per top-level domain per week.
- The production alternatives:
  - **A global external Application Load Balancer with Certificate Manager.** About US$ 18 per month
    for the first five forwarding rules; the first 100 certificates free, then US$ 0.20 per
    certificate per month. Wildcards with DNS authorisation.
  - **Cloudflare for SaaS.** 100 custom hostnames free on every plan, then US$ 0.10 each. But
    rewriting the `Host` header towards a `run.app` origin is Enterprise-only, so on the free plan it
    needs a Worker in front of Cloud Run, or the load balancer above as origin.

**Decision:**
- **The lab** declares its hosts as a list in Terraform, exactly as the marketplace does: one Cloud
  Run domain mapping, one certificate and one Cloudflare record in "DNS only" mode per host.
- **The code** depends on a `HostProvisioner` port, never on how a host came to reach it. The lab's
  adapter is "declared in Terraform" (a no-op that reports the host as ready once it answers).
- **The cloud control plane** needs an adapter that provisions hosts at run time through an API,
  because a hosted customer adding a domain cannot wait for a pull request. The choice between
  Certificate Manager behind a load balancer and Cloudflare for SaaS is a spike before the control
  plane's phase, recorded as an open item.

## 7. No event sourcing in the core

**Status:** proposed.

**Briefing:** asked whether the core should be event-sourced or whether an append-only audit log
suffices.

**Reasons against event sourcing:**
- Identity state is small and the hot path is reads (token validation, session lookup); event
  sourcing adds projection lag and rebuild cost to exactly that path.
- Erasure under privacy law is harder when the source of truth is an immutable event stream;
  crypto-shredding helps with personal fields but not with the structure of the stream.
- Schema evolution of events is a permanent cost, and row-level security and administrative queries
  are simpler over current-state tables.

**Decision:** current-state tables are the source of truth. Every change writes, in the same
transaction, **an entry in the append-only, hash-chained audit log** (personal fields encrypted
with a per-user key, so erasure is key destruction) and **the domain events to an outbox**. The
event schema is part of `spec/` and is a public contract; events are published, not replayed to
rebuild state.

## 8. Few Go modules in phase 1

**Status:** proposed.

**Briefing:** asked whether each `core/*` package should be its own module in `go.work` from the
start.

**Decision:** the closed decision stands — one monorepo, Go workspaces, modules tagged individually
(`core/v0.3.0`). What changes is the granularity in phase 1: **a handful of modules**, not one per
package. A module per package multiplies tag coordination before any API is stable, and makes every
cross-package change a multi-module release. The rule: a package becomes its own module when an
outside consumer needs to version it independently. The phase 1 layout is in `docs/design.md`.

## 9. Storage

**Status:** proposed.

**Decision:**
- The core defines **one interface per aggregate** (`UserStore`, `SessionStore`, `ClientStore`...),
  in domain types, never in SQL, plus a unit of work: `Tx(ctx, func(Stores) error) error`.
- **One test suite** that every storage adapter must pass, run against each adapter in CI.
- **PostgreSQL** is the system of record for the server. Every table carries `tenant_id`; row-level
  security covers every table (closed decision).
- **SQLite** is supported for the library and for a single-node, single-tenant server. **The server
  refuses to start in multi-tenant mode on SQLite**, because SQLite has no row-level security and
  the closed decision requires it wherever tenants share storage.
- **Redis** or any similar store is allowed only for ephemeral data (rate-limit counters, caches),
  never as the source of truth.
- **DynamoDB and other non-relational stores** are not planned: their consistency model would leak
  into the core. Zitadel's decision to drop CockroachDB and support PostgreSQL only
  (`docs/research/market.md`) supports this direction. Other relational databases (MySQL, SQL
  Server) are in the "beyond 1.0.0" register, reachable through the same interfaces and suite.

## 10. Authorisation

**Status:** proposed; the owner agreed that relationship-based authorisation comes after 1.0.0.

**Options weighed** (`docs/research/market.md`, section 3):
- **Embed OpenFGA.** Mature, Apache-2.0, CNCF Incubating, embeddable. But it brings its own tuple
  storage and schema, outside this project's `tenant_id`, row-level security and per-tenant keys.
- **Cedar (cedar-go).** A policy language (attribute-based, with entity hierarchies), not a
  relationship graph; schema validation still experimental in Go.
- **SpiceDB.** Built to run as a separate service; in-process use is not a supported API.
- **Own engine.** More work; full control of storage, isolation and consistency.

**Decision:**
- **1.0.0:** role-based access control scoped to organisations — roles, permissions, membership
  roles, custom roles per tenant — carried in tokens and checkable through the API.
- **Beyond 1.0.0:** relationship-based authorisation through **an engine of this project's own that
  accepts OpenFGA's modelling language**, stored in the project's schema under row-level security.
  OpenFGA's own test suite serves as the compatibility oracle; Cedar is a candidate for evaluating
  conditions attached to relationships. The extension point from phase 1 is the `Authorizer`
  interface through which every permission check already passes.

## 11. Plugins: wazero with the project's own ABI

**Status:** proposed.

**Briefing:** WebAssembly plugins through wazero, and evaluate Extism on top.

**Facts:** wazero is active (v1.12.0, May 2026) and supports WASI preview 1, not the Component Model.
Extism's Go SDK is built on wazero, pins an old wazero release, and has had no release since March
2025. Go 1.24 added `//go:wasmexport` and reactor-style `wasip1` modules, so plugins can be written
in Go.

**Decision:** wazero directly, with a **minimal ABI of this project's own** — JSON in, JSON out, host
functions through a declared allowlist — versioned inside the product's SemVer. Extism's main
attraction, plugin development kits in many languages, does not outweigh tying the product's
versioned ABI to a dependency whose release cadence has stalled. The project publishes thin plugin
kits for Go, Rust and TypeScript. The Component Model is tracked; the ABI's version field lets a
second ABI arrive beside the first.

## 12. Migration: the minimum set that proves the architecture

**Status:** proposed.

**Decision:** one importer per *shape* of source, plus one lazy migration and one rollback:
- **Auth0** — JSON export with bcrypt hashes.
- **Firebase** — its modified scrypt with per-project parameters.
- **Keycloak** — realm export with PBKDF2 and federated credentials.
- **Declarative CSV/JSON mapping** — the generic path.
- **Lazy migration against Cognito** — a source that exports no hashes.
- **Rollback** to one of the above, exporting in its format.

The remaining importers and connectors of the briefing are built on the same interfaces and are in
1.0.0 scope if they fit, otherwise in the "beyond 1.0.0" register; none is dropped.

Legacy hash algorithms sit in apparent tension with the rule that weak algorithms are refused. The
resolution is written into the design: **legacy algorithms are verifiers only.** A legacy hash is
checked once, at the user's next sign-in, and immediately replaced by argon2id (or PBKDF2 in the FIPS
profile); no code path writes a new hash in a legacy algorithm.

## 13. The cryptographic algorithm set

**Status:** proposed.

**Constraint:** OpenID Connect Core requires every OpenID Provider to support **RS256**
(`docs/research/standards.md`, section 3), and FAPI 2.0 requires **PS256** or **ES256**. "Few, strong
algorithms" must therefore include RSA.

**Decision:**
- **Token and request-object signatures:** ES256, EdDSA (Ed25519), PS256, RS256. RSA keys of at
  least 2048 bits, 3072 by default. The default per tenant is ES256.
- **Refused, never configurable:** `none`; HMAC algorithms for tokens this server issues; RSA below
  2048 bits; SHA-1 in any signature, SAML included; RSA PKCS#1 v1.5 *encryption*.
- **Passwords:** argon2id, with parameters set by a benchmark recorded for the deployment's CPU;
  PBKDF2-HMAC-SHA-512 in the FIPS profile.
- **Cryptography only from the Go standard library and `golang.org/x/crypto`.** Comparisons of
  secrets in constant time, enforced by a lint rule.

## 14. Signing keys and the KMS

**Status:** proposed.

**Briefing:** signing never calls the KMS per token; the KMS wraps each tenant's signing keys,
signing happens in memory, rotation re-wraps.

**Decision:** that is the default, and it is right for latency and cost. Two additions:
- A **`Signer` interface** with a second implementation that **signs inside Cloud KMS or an HSM**, per
  tenant, for customers whose regulation requires non-exportable keys. Its latency and cost are
  documented; it is a deployment option, never required by the code.
- **Keys are unwrapped lazily per tenant**, on first use after start, and cached. Cloud Run scales to
  zero; unwrapping every tenant's keys at start would make cold-start latency grow with the number of
  tenants.

## 15. A FIPS profile, not a FIPS switch

**Status:** proposed.

argon2id is not FIPS-approved (`docs/research/standards.md`, section 9). A deployment that turns on
Go's FIPS 140-3 mode must therefore hash passwords with PBKDF2 and restrict signatures to the
approved set. The design makes this a named **deployment profile** that changes those choices
together and refuses to start if any setting contradicts it, rather than a flag that leaves argon2id
running under a FIPS label.

## 16. Conformance: one black-box suite, two targets

**Status:** proposed.

**Decision:** `conformance/` is a black-box suite over HTTP. It runs against a `Target` interface:
the base URL plus an administrative client that registers clients and users for the test. The
library is a target through an in-process `httptest.Server`; the server is a target as a process.
The OpenID Foundation's official suite (Java, run with Docker) runs **in CI** against the server
binary. It cannot run in a Claude Code session, where the Docker daemon does not run.

## 17. The library hides the tenant without a second code path

**Status:** proposed.

Every core operation reads the tenant from the request context. The library's constructor binds the
single tenant it created at bootstrap and injects it with a middleware; an embedding application never
sees a `tenant_id`. There is one code path: the library is a server with the tenant fixed, not a
variant of the core.

## 18. One flow engine, several renderers

**Status:** proposed.

**Briefing:** templ + HTMX + Alpine.js for the hosted login and the admin console; React for UI
components; asked how to avoid duplicating logic.

**Decision:** the pattern of Ory Kratos's self-service flows. The server owns a **flow state machine**
for every user-facing journey (sign-up, sign-in, recovery, enrolment, consent) and exposes its current
state as data: the fields, their errors, the next actions. The hosted login (templ + HTMX, which works
without JavaScript and suits a strict content security policy) and the React components are both
**renderers** of that same state. Logic lives once, on the server. The admin console is templ + HTMX
over the same service layer as the public management API, so nothing is possible in the console that
the API cannot do.

## 19. SAML: own protocol logic, hardened signature verification

**Status:** proposed.

XML signature processing is the most attack-prone component in identity (signature wrapping, comment
injection in canonicalisation, external entities). The decision:
- The SAML protocol logic — requests, responses, bindings, metadata, brokering — is this project's
  own.
- XML signature verification uses a maintained, reviewed implementation (`goxmldsig`) **behind the
  project's own interface**, after a strict pre-validation that rejects document type declarations,
  external entities, more than one assertion, and any signed element other than the one the
  processing then reads.
- The parser and the verifier are fuzzed continuously; `crewjam/saml` and real IdPs' metadata serve
  as interoperability oracles.

## 20. The Terraform provider is mirrored to its own repository

**Status:** proposed.

Both the Terraform Registry and the OpenTofu registry publish a provider only from a **public
repository named `terraform-provider-<name>`**, with releases and `vX.Y.Z` tags in that repository
([Terraform](https://raw.githubusercontent.com/hashicorp/web-unified-docs/main/content/terraform-docs-common/docs/registry/providers/publishing.mdx),
[OpenTofu](https://raw.githubusercontent.com/opentofu/registry/main/.github/ISSUE_TEMPLATE/provider.yml)).
A monorepo cannot publish directly. The provider is developed in the monorepo; CI mirrors each
release to a dedicated repository that holds only releases. This is the one exception to the
single repository, and it needs a name, so it waits for the brand (`docs/research/market.md`,
section 5).

## 21. Two milestones: 1.0.0 and the commercial launch

**Status:** accepted by the owner.

Paid certification, the external security audit and legal review serve third parties — buyers,
regulators, contributors — and are not needed while the owner's own projects are the only clients.
**1.0.0** is the complete product: every conformance suite green in CI and a complete internal
security review. The **commercial launch** adds the paid items when the owner decides to fund them.
Details in `docs/research/licensing.md`, section 6.
