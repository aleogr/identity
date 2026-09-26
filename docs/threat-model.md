# Identity Platform: Threat Model

> **Status:** written in the inaugural design session of 2026-09-26. It is a living document: every
> phase's roadmap names the threats its deliveries mitigate, and a delivery that adds a trust
> boundary, a parser, an external call or a secret updates this file in the same pull request.
>
> **Sources:** `docs/requirements.md` (§N), `docs/design.md` ("design §N"),
> `docs/research/architecture-decisions.md` ("decision N"), RFC 9700 (OAuth 2.0 Security Best
> Current Practice), the OpenID Connect, FAPI 2.0 and WebAuthn security considerations.
>
> **How to use it:** each threat has an identifier, the mitigation and where it is designed, the
> **verification** that proves it (a test, a conformance suite, a lint rule, a CI check), and the
> phase that delivers it. A threat without a verification is not mitigated, only intended.

---

## 1. Scope and trust assumptions

Approved by the owner on 2026-09-26.

| Adversary | Treatment |
|---|---|
| Anonymous external attacker (credential stuffing, phishing, bots, parser exploitation) | In scope |
| Malicious end user | In scope |
| **Malicious tenant against other tenants**, including its administrators | In scope — the central threat of multi-tenant mode |
| Malicious organisation administrator, against their own or other organisations | In scope |
| Compromised or malicious client application (dynamically registered or CIMD clients, forged redirects, mix-up) | In scope |
| Malicious WebAssembly plugin or webhook endpoint | In scope |
| Manipulated AI agent (prompt injection leading to misuse of delegated tokens) | In scope, mitigated in authorisation (least privilege, consent, audit, revocation), not in the model |
| Supply chain: dependencies, CI, images | In scope |
| Compromised **operator** account | Partially: detection and traceability, mandatory strong MFA; the operator has broad legitimate power and no complete defence exists |
| Compromised cloud provider, KMS or hardware; physical access | Out of scope, a declared assumption |
| Volumetric denial of service | Out of scope for the code (the edge's job); **application-level** denial of service is in scope |

## 2. Assets

| Asset | Why it matters | Where it lives |
|---|---|---|
| **Tenant signing keys** | Forge any token for any user of the tenant | Encrypted by the tenant data key in the database; plaintext only in process memory (design §4.1) |
| **Tenant data keys** | Unlock every tenant secret and audit key | Wrapped by the KMS key in the database; cached in memory |
| **KMS key-encryption key** | Unwraps every data key | Inside Cloud KMS; never exported |
| **Password hashes and credential material** | Offline cracking; passkey public keys are not secret but their binding is | Database, argon2id |
| **Sessions, authorisation codes, access and refresh tokens** | Account takeover | Hashes in the database; bearer values only in transit and in clients |
| **Token vault contents** | Access to users' third-party accounts | Encrypted with the tenant data key; released only through audited token exchange |
| **Recovery channel** (e-mail) and recovery state | Account takeover through recovery | Database; the user's mailbox |
| **Client secrets and private-key JWT keys** | Impersonate a client | Secrets hashed; public keys registered |
| **Personal data** | Privacy, law | Database; audit fields encrypted per user |
| **Audit log integrity** | Undetected tampering hides every other attack | Hash-chained entries |
| **Configuration and floors** | Weakening security silently | Settings registry, audited |
| **Build and release pipeline** | Ship a backdoor to every self-hosted user | GitHub Actions, Workload Identity Federation, signing identity |

## 3. Trust boundaries

```
 [browser / user agent] ──B1── [server] ──B4── [PostgreSQL]   [Cloud KMS / Secret Manager]
 [OAuth / SAML / SCIM client] ──B2──┘   │  └────B4────────────────────┘
 [federated IdP, CIMD host, JWKS host, webhook endpoint] ──B3── (outbound from server)
 [WebAssembly plugin] ──B5── [plugin host inside the server]
 [tenant A] ──B6── [tenant B]          (logical boundary inside one deployment)
 [GitHub, CI, registries] ──B7── [images, releases, GCP]
```

## 4. Threats by boundary

STRIDE category: **S**poofing, **T**ampering, **R**epudiation, **I**nformation disclosure, **D**enial of
service, **E**levation of privilege.

### B1 — Browser and user agent ↔ server

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B1-01 | S | Credential stuffing and password spraying | Per-IP, per-identifier and per-tenant rate limits with progressive delay; breached-password blocklist; proof-of-work under risk (§6.6) | Integration tests of the limiter across two instances sharing PostgreSQL; e2e of the challenge | 1, 2 |
| B1-02 | I | Account enumeration through responses or timing | Identical responses and constant-work paths for existing and missing accounts (a dummy hash is computed); floor, cannot be disabled | Test comparing response bodies and a statistical timing test in CI | 1 |
| B1-03 | D | Account lockout used as denial of service | No hard lockout; progressive delay | Test that a flood against one identifier does not block a correct sign-in from a clean origin after the delay | 2 |
| B1-04 | S | Phishing of passwords and one-time codes | Passkeys first-class and origin-bound; conditional create to upgrade users; codes bound to the requesting browser where possible | e2e with Chromium's virtual authenticator; test that a code is refused from another browser session | 1, 2 |
| B1-05 | T | Cross-site request forgery on state-changing forms | Per-session CSRF tokens; `SameSite=Lax` `__Host-` cookies; `Origin` checks | Handler tests without token, with a foreign origin | 1 |
| B1-06 | T | Clickjacking of login and consent | `frame-ancestors 'none'` except configured trusted origins | Header tests | 1 |
| B1-07 | I, E | Cross-site scripting in hosted pages, including tenant branding | templ contextual escaping; strict CSP with nonces; branding limited to a whitelist of CSS properties and uploaded images re-encoded | CSP test on every page; fuzzed branding inputs | 1, 4 |
| B1-08 | S | Session fixation and hijacking | New session identifier at every authentication level change; opaque token, hash stored; idle and absolute timeouts; revocation on credential change (floor) | Tests of rotation on sign-in and step-up; revocation tests | 1, 2 |
| B1-09 | E | Account takeover through recovery | Security delay, notification on every channel, "this wasn't me" cancellation, revocation of all sessions; administrators recovered only by another administrator | e2e of recovery including cancellation | 2 |
| B1-10 | S | Open redirect through `return_to`, `post_logout_redirect_uri`, `redirect_uri` | Exact matching against registered values; relative internal paths only for `return_to` | Table-driven tests with known bypass shapes (`//evil`, `\evil`, encoded, userinfo) | 1 |
| B1-11 | I | Cross-tenant passkey or cookie leakage through a shared host | Host-based issuers; `__Host-` cookies; relying party ID per tenant (decision 5) | Test that a cookie of tenant A is not sent to tenant B's host; WebAuthn tests with mismatched RP IDs | 1, 4 |
| B1-12 | E | WebAuthn ceremony flaws: wrong origin or RP ID, missing user verification, cloned authenticator | Library verification behind our interface; origin and RP ID checked against the tenant; UV required per policy; signature counter regressions flagged | Negative tests per field; counter regression test | 1 |

### B2 — Clients (OAuth, OIDC, SAML, SCIM) ↔ server

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B2-01 | S, E | Authorisation code interception or injection | PKCE `S256` required for every client (floor); codes single-use, 60 seconds, bound to client, redirect URI and challenge; reuse revokes issued tokens | OIDF suite; negative tests for replay and wrong verifier | 1 |
| B2-02 | S | Mix-up attacks between authorisation servers | `iss` authorisation response parameter always returned (RFC 9207); distinct issuer per tenant | OIDF suite; test of the parameter on every response mode | 1 |
| B2-03 | E | Redirect URI manipulation | Exact string matching; loopback port exception only for native clients (RFC 8252); no wildcards | Table-driven tests | 1 |
| B2-04 | E | Downgrade to weak grants or flows | Implicit grant and resource-owner password grant not implemented; `none` algorithm refused; PKCE `plain` refused | Tests that each is refused; lint rule against the constants | 1 |
| B2-05 | S | Token theft and replay | Short-lived access tokens; DPoP or mTLS sender-constraining; refresh rotation with family revocation on reuse | Replay tests; DPoP proof `jti` replay cache test; FAPI suite | 1, 5 |
| B2-06 | S | Algorithm confusion and key confusion in JWTs we verify (client assertions, request objects, DPoP proofs, federated ID tokens) | Algorithm fixed per key and per context, never read from the token alone; `kid` resolved only within the expected key set; HMAC never accepted where an asymmetric key is registered | Tests with `alg` swapped, `kid` pointing elsewhere, embedded `jwk` rejected where not allowed; fuzzing of the JWT parser | 1 |
| B2-07 | E | Scope or audience escalation | Scopes intersected with the client's allowed set and the user's consent; resource indicators bind audience; tokens rejected by resources outside their audience | Unit and integration tests; MCP audience tests | 1, 5, 9 |
| B2-08 | E | Token exchange privilege escalation (delegation chains, impersonation) | Exchange policies per tenant: who may act for whom, maximum chain depth, scope narrowing only, `act` claims recorded | Policy tests including chain-depth and widening attempts | 5, 9 |
| B2-09 | S | Malicious dynamically registered or CIMD client impersonating a known brand on the consent screen | Consent shows the verified host of the metadata document, never only the self-asserted name; registration policies per tenant (initial access tokens, allowed hosts) | e2e screenshot of consent; policy tests | 5, 9 |
| B2-10 | S | Device authorisation grant phishing | User code shown with the client's verified identity and the requesting location; short lifetime; rate-limited polling | Tests of expiry and polling limits | 5 |
| B2-11 | T | Tampering with authorisation request parameters | PAR and JAR available to every client and required under FAPI; `request_uri` single-use | FAPI suite | 5 |
| B2-12 | S, E | SAML signature wrapping, comment injection, XXE, assertion replay | Strict pre-validation (no DTD, no external entities, exactly one assertion, the signed element is the one read); `goxmldsig` behind our interface; `InResponseTo`, audience, recipient, time window and one-time-use checks | Test corpus of known XSW variants; continuous fuzzing; interoperability tests | 6 |
| B2-13 | T, E | SCIM filter or patch injection, provisioning into the wrong tenant | Hand-written filter parser to an AST, compiled to parameterised queries; SCIM tokens scoped to one tenant and organisation | Fuzzing of the filter parser; cross-tenant provisioning test | 7 |
| B2-14 | R | A client or administrator denies an action | Every grant, token issuance class, consent, administrative change and hook execution audited with the actor | Audit completeness tests per endpoint | 1 |

### B3 — Server → outbound calls (federated IdPs, CIMD documents, JWKS, webhooks, breached-password service)

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B3-01 | I, E | **Server-side request forgery** through URLs supplied by tenants or clients (CIMD `client_id`, `jwks_uri`, `request_uri`, `sector_identifier_uri`, logos, webhook targets, federation metadata) | One outbound HTTP client for all of them: HTTPS only, resolved addresses checked against private, loopback, link-local and metadata ranges **after** DNS resolution and on every redirect, size and time limits, no credentials | Tests against a resolver returning private addresses, redirect chains, DNS rebinding simulation | 1, 5, 6 |
| B3-02 | S | A federated IdP asserts an identity it does not own (e.g. an unverified e-mail used to link accounts) | Account linking only on verified claims and per-connection rules; `email_verified` honoured; per-organisation domain verification before home-realm routing | Linking tests with unverified e-mails | 6 |
| B3-03 | D | Slow or hostile endpoints stall request handling | Timeouts, circuit breakers, asynchronous delivery for webhooks and events | Tests with a stalled endpoint | 1, 7, 8 |
| B3-04 | I | Webhook payloads leak personal data to a misconfigured endpoint | Per-subscription event and field selection; signed payloads; delivery log visible to the tenant | Tests of field filtering | 7 |

### B4 — Server ↔ database, KMS and secrets

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B4-01 | I, T | SQL injection | Parameterised queries only; no string-built SQL; `gosec` | Lint; fuzzing of inputs reaching storage | 1 |
| B4-02 | I | Database dump exposes secrets | Hashes for passwords, sessions, codes, tokens and client secrets; everything else secret encrypted under the tenant data key; the KMS key never in the database | Test that no plaintext secret column exists (schema scan in CI) | 1 |
| B4-03 | E | The service's database role exceeds its needs | Two roles; the service is subject to row-level security and owns nothing; migrations run as the migrator | Integration tests run as the service role; a test that the service cannot alter the schema | 1 |
| B4-04 | I | Keys or tokens written to logs | Structured logging with typed secret wrappers that redact; a lint rule against logging request bodies and headers wholesale | Log-scanning test in e2e for known token prefixes | 1 |
| B4-05 | D | The KMS is unavailable | Data keys cached with a bounded lifetime; a cold instance without KMS access fails closed with a clear error | Test with a failing KMS fake | 1 |

### B5 — WebAssembly plugins ↔ host

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B5-01 | E | Sandbox escape | wazero, no WASI filesystem or network; host functions only from the manifest's allowlist | Tests that undeclared host functions fail to link | 8 |
| B5-02 | D | Resource exhaustion | Memory limit, per-call deadline, instance per call | Tests with a looping and a memory-hungry plugin | 8 |
| B5-03 | E | A hook grants more than its extension point allows | Mutation rules per extension point enforced by the host on the returned JSON | Contract harness tests for each point | 1, 8 |
| B5-04 | I | Exfiltration through `fetch` | `fetch` only to domains declared in the manifest, through the SSRF-safe client (B3-01) | Tests | 8 |
| B5-05 | R | A tenant disputes what a plugin did | Each execution audited with module hash and contract version | Audit tests | 8 |

### B6 — Tenant ↔ tenant

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B6-01 | I, T | Reading or writing another tenant's rows through a missing filter | Row-level security on every table; a transaction without a tenant reads nothing; application scoping as a second layer | A CI check that every table has a policy; cross-tenant tests for every store | 1 |
| B6-02 | S | Tokens of tenant A accepted by tenant B | Distinct issuer and keys per tenant; audience and issuer validated | Cross-tenant token tests | 1, 4 |
| B6-03 | E | A tenant administrator affects the deployment | Tenant settings cannot cross the deployment's floors; operator-only endpoints on a separate authorisation path | Tests per operator endpoint | 4 |
| B6-04 | D | Noisy neighbour: one tenant exhausts shared capacity | Per-tenant rate limits and quotas; hash cost bounded per tenant | Load test in phase 12 | 4, 12 |
| B6-05 | I | Cross-tenant leakage through caches | Cache keys always include the tenant | Unit tests of cache key construction | 1 |

### B7 — Supply chain and CI

| ID | STRIDE | Threat | Mitigation | Verification | Phase |
|---|---|---|---|---|---|
| B7-01 | T | Malicious or vulnerable dependency | Minimal dependency policy for `core`; `govulncheck`, Dependabot, Trivy; reviewed updates | CI | 1 |
| B7-02 | T | Compromised workflow or third-party action | Actions pinned by commit SHA; least-privilege `permissions:`; no secrets in pull requests from forks; Workload Identity Federation restricted to this repository and branch | Workflow lint in CI | 1 |
| B7-03 | T | Tampered release artefacts | Reproducible builds with a digest check; Sigstore signatures; SLSA provenance; SBOM | Release workflow verification step | 1, 12 |
| B7-04 | I | Secret committed to the public repository | `gitleaks` in CI; secrets only in Secret Manager and repository secrets | CI | 1 |

## 5. Cross-cutting threats

| ID | Threat | Mitigation | Verification |
|---|---|---|---|
| X-01 | Timing side channels on secret comparison | Constant-time comparison, enforced by a lint rule | Lint |
| X-02 | Weak or misused cryptography | Algorithm set fixed in code with refusals (§17); keys generated only by the crypto packages; nonces never reused (random 96-bit for AES-GCM with rotation well below the bound) | Tests; lint against `math/rand` in security packages |
| X-03 | Parser bugs (JWT, JSON, CBOR, XML, SCIM filters, URLs) | Fuzzing of every parser, corpus committed, run on every pull request and nightly | CI fuzz jobs |
| X-04 | Application denial of service through expensive operations | argon2id cost bounded and concurrency-limited; request size limits; XML and JSON depth limits; regular expressions only from RE2 | Tests with oversized and deep inputs |
| X-05 | Misconfiguration weakens security | Floors in code; invalid configuration refuses to start; recommendations shown in the console | Tests per floor through every facade |
| X-06 | Prompt-injected agents misuse delegated tokens | Least-privilege scopes and RAR, per-grant consent, short lifetimes, token vault releases audited and revocable, CAEP revocation | Tests of scope narrowing and revocation propagation |

## 6. The operator

The operator can read configuration, create tenants and see audit logs; a compromised operator
account is therefore severe. Mitigations: mandatory TOTP or WebAuthn for operator and tenant
administrator accounts (never e-mail codes); every operator action audited into the hash chain; alerts
on operator sign-in from a new device; no operator path that reads tenant secrets in plaintext;
break-glass access documented and audited. Residual risk: an operator with database and KMS access can
do anything, which the trust assumptions accept.

## 7. Accepted residual risks

- **Cloud provider and KMS compromise** — out of scope (section 1).
- **Volumetric DoS** — delegated to the edge; the lab has no edge protection.
- **mTLS in the lab** is exercised only in CI against the binary (design §12.4).
- **Cloud Run domain mapping** is a Preview feature; certificate issuance may lag.
- **E-mail as the recovery channel** is as strong as the user's mailbox; the security delay, the
  "this wasn't me" path and passkeys reduce but do not remove this.

## 8. Keeping the model current

- Every pull request that adds a trust boundary, a parser, an outbound call, a secret or an extension
  point updates this file; the identity security review checklist asks for it.
- Every phase's roadmap lists the threat identifiers its deliveries mitigate, and its closing delivery
  checks that each has its verification in place.
- Before 1.0.0, the complete internal security review walks this document line by line against the
  code (§23).
