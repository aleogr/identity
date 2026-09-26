---
name: identity-security-review
description: Domain security checklist for pull requests in this identity platform. Use before opening or updating any pull request that touches authentication, sessions, tokens, keys, cryptography, protocol endpoints (OAuth, OIDC, SAML, SCIM, WebAuthn), parsers, outbound HTTP calls, tenancy or row-level security, settings and floors, hooks or plugins, or the audit log.
---

# Identity security review

Run this on the diff of the branch before opening or updating a pull request. It complements the
generic `security-review` skill with the rules of this domain and project. Report every item that fails
with the file and line, and fix it before pushing; an item that does not apply is skipped silently.

The sources are `docs/threat-model.md` (threat identifiers), `docs/requirements.md` sections 6, 8, 9,
15.2 and 17, and RFC 9700.

## 1. Protocol rules (RFC 9700 and OpenID Connect)

- PKCE with `S256` is required for every client; `plain` is refused (B2-01).
- Redirect URIs match **exactly**; the only exception is the loopback port for native clients
  (RFC 8252). No wildcard, prefix or substring matching anywhere (B2-03, B1-10).
- The `iss` authorisation response parameter is present on every response mode (B2-02).
- No implicit grant, no resource owner password grant, no `none` algorithm (B2-04).
- Authorisation codes: single use, short lifetime, bound to client, redirect URI and PKCE challenge;
  reuse revokes what was issued.
- Refresh tokens rotate; reuse revokes the family (B2-05).
- `state` and `nonce` are checked where the flow uses them.

## 2. JWT and signature handling (B2-06)

- The algorithm is fixed by the key and context, never taken from the token's header alone.
- `kid` is resolved only inside the expected key set; an embedded `jwk` or `jku` is not trusted unless
  the context explicitly allows it (DPoP proofs carry their `jwk`, which is then bound).
- HMAC is never accepted where an asymmetric key is registered.
- Only the algorithm set of requirements §17: ES256, EdDSA, PS256, RS256 (RSA at least 2048 bits).
- SAML: no SHA-1; the strict pre-validation of decision 19 runs before any signature check; the element
  read after verification is the element that was signed (B2-12).

## 3. Secrets, keys and cryptography

- Cryptography only from the standard library and `golang.org/x/crypto`. No `math/rand` in security
  code.
- Secrets compared with `crypto/subtle.ConstantTimeCompare` or `hmac.Equal` (X-01).
- Passwords, session tokens, codes, refresh tokens and client secrets are stored **hashed**; other
  secrets encrypted under the tenant data key; nothing secret in plaintext in the database (B4-02).
- No secret, token, code, password or key reaches a log, an error message or a metric; typed secret
  wrappers are used (B4-04).
- No secret in the repository, in a workflow's environment or in a Terraform variable.

## 4. Tenancy and isolation (B6)

- Every new table has `tenant_id` and a row-level security policy; the CI policy check covers it.
- Every query runs in a transaction that set the tenant; no code path uses a role that bypasses the
  policies except the declared, audited operator functions.
- Every cache key includes the tenant.
- Tokens are validated for issuer **and** audience; a token of one tenant is never accepted by another.

## 5. Web surface (B1)

- State-changing forms have CSRF protection; cookies are `__Host-`, `Secure`, `HttpOnly`,
  `SameSite=Lax` or stricter.
- Pages carry the CSP with a nonce and `frame-ancestors`; no inline script without the nonce; no
  user or tenant input rendered outside templ's contextual escaping.
- Sign-in, sign-up and recovery return identical responses and do constant work for existing and
  missing accounts (B1-02).
- Session identifiers rotate on every authentication level change (B1-08).

## 6. Outbound calls and parsers

- Every outbound HTTP request uses the SSRF-safe client: HTTPS, private and metadata ranges refused
  after resolution and on redirects, size and time limits (B3-01).
- Every new parser (JWT, JSON with depth limits, CBOR, XML, SCIM filters, URLs) has a fuzz target, and
  input sizes are bounded (X-03, X-04).
- Regular expressions are RE2 only (the standard library's `regexp`).

## 7. Settings and floors (X-05)

- A new security-relevant setting is declared in the settings registry with its default and, where
  relaxing it weakens security, a floor.
- No facade (console, API, CLI, declarative file, Terraform) writes a setting except through the
  registry.
- Invalid configuration refuses to start, naming the value.

## 8. Hooks, plugins and audit

- A new extension point declares its allowed mutations and whether it fails open or closed; the host
  enforces the mutations (B5-03).
- Every administrative change, grant, consent and hook execution writes an audit entry (B2-14).

## 9. The threat model and the documents

- If the diff adds a trust boundary, a parser, an outbound call, a secret or an extension point,
  `docs/threat-model.md` is updated in the same pull request, with the verification for each new
  threat.
- A threat marked mitigated has its test in the diff or already in `main`.

## Output

List the failures, each as: the checklist item, `file:line`, what is wrong, the fix. If there are none,
say so in one line in the pull request's verification section.
