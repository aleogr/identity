# Research: the state of the standards

> **Status:** research note written in the inaugural design session of 2026-09-25. It records where
> each standard the product implements stood on that date and what that means for the product.
> `docs/requirements.md` and `docs/design.md` cite it; they, not this note, are the sources of
> truth for decisions.
>
> **Method:** web research on 2026-09-25. Some standards bodies' sites (IETF datatracker, RFC
> Editor, openid.net) were unreachable from the research session, so several facts come from
> search-result snippets or from the specifications' mirrors on GitHub. Facts that could not be
> confirmed from a primary source are marked **[unverified]** and must be re-checked before a
> decision rests on them. Dates of long-published RFCs are given from their RFC numbers and were
> not re-fetched.

---

## 1. Why this note exists

Identity is a domain where the standard *is* the product: a client written against Keycloak must
work against this server by changing the issuer, and an enterprise's SAML IdP must federate without
code. The briefing that started the project listed several standards from memory; this note checks
each one, because a specification that became final, was deprecated or was superseded changes what
the first version must implement.

## 2. OAuth 2.0 and OAuth 2.1

| Specification | State on 2026-09-25 | Consequence |
|---|---|---|
| OAuth 2.1 (`draft-ietf-oauth-v2-1`) | **Still an Internet-Draft.** Latest confirmed revision -15, 2026-03-02 ([draft](https://www.ietf.org/archive/id/draft-ietf-oauth-v2-1-15.html)); a -16 appears in one search title **[unverified]**. No evidence of IESG approval. | "OAuth 2.1" is a profile the server follows, not a normative citation. The normative base is RFC 6749 plus RFC 9700. |
| RFC 9700, OAuth 2.0 Security Best Current Practice (BCP 240) | Published January 2025 ([RFC 9700](https://www.rfc-editor.org/rfc/rfc9700)) | The server's defaults are RFC 9700's MUSTs, not options: PKCE with `S256` for every client, no implicit grant, no resource owner password grant, exact redirect URI matching (with RFC 8252's loopback port exception), refresh tokens of public clients sender-constrained or rotated, mix-up defence through the `iss` response parameter. |
| RFC 10017, OAuth 2.0 for Browser-Based Applications (BCP 212) | Published; exact month **[unverified]** ([RFC 10017](https://www.rfc-editor.org/info/rfc10017/)) | The TypeScript SDK and the documentation for single-page applications follow it, including the backend-for-frontend pattern as the recommended architecture. |
| RFC 9449, DPoP | Proposed Standard, September 2023 | Sender-constrained tokens for public clients; the default for refresh tokens of public clients alongside rotation. |
| RFC 8705, mutual-TLS client authentication and certificate-bound tokens | Proposed Standard, February 2020 | Needed by FAPI 2.0 and by confidential clients in regulated deployments. On Cloud Run the TLS connection terminates at Google's front end, so mTLS needs either a load balancer that forwards the client certificate or a deployment that terminates TLS itself; the lab cannot exercise it (see `docs/design.md`). |
| RFC 9126, Pushed Authorization Requests | Proposed Standard, September 2021 | Mandatory in FAPI 2.0; offered to every client. |
| RFC 9101, JWT-Secured Authorization Request | Proposed Standard, August 2021 | Signed request objects; required by FAPI 2.0 Message Signing. |
| RFC 9396, Rich Authorization Requests | Proposed Standard, May 2023 | `authorization_details`, the structured consent that fine-grained agent authorisation needs. |
| RFC 9207, authorization server issuer identification | Proposed Standard, March 2022 | Always returned. The MCP specification of 2026-07-28 makes its validation mandatory for clients (section 6). |
| RFC 8693, Token Exchange | Proposed Standard, January 2020 | The delegation primitive: impersonation and delegation, agents acting on behalf of users, the ID-JAG flow of section 6. |
| RFC 7662, Token Introspection; RFC 7009, Token Revocation | Proposed Standards (2015, 2013) | Opaque tokens are introspected; every token type is revocable. |
| RFC 9068, JWT profile for access tokens | Proposed Standard, October 2021 | The JWT access token format. |
| RFC 9470, step-up authentication challenge | Proposed Standard, September 2023 | Resource servers ask for a stronger authentication through `WWW-Authenticate`; the server honours `acr_values` and `max_age`. |
| RFC 8628, Device Authorization Grant | Proposed Standard, August 2019 | Televisions, command-line tools, agents without a browser. |
| RFC 7591 / 7592, Dynamic Client Registration and its management | Proposed Standards, July 2015 | Still required by OpenID's "Dynamic" certification profile; demoted by MCP in favour of client ID metadata documents (section 6). |
| RFC 8414, authorization server metadata | Proposed Standard, June 2018 | Published next to OpenID discovery. |
| RFC 8707, resource indicators | Proposed Standard, February 2020 | Audience binding; mandatory for MCP clients. |
| RFC 9728, OAuth 2.0 Protected Resource Metadata | Proposed Standard, April 2025 | The server publishes it for its own APIs and helps customers publish it for theirs (MCP servers). |
| `draft-ietf-oauth-client-id-metadata-document` (CIMD) | Internet-Draft -02, expires 2027-01-07 ([draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-02)) | A client identified by the HTTPS URL of its own metadata document. Preferred by MCP since 2026-07-28. |
| `draft-ietf-oauth-identity-assertion-authz-grant` (ID-JAG) | Internet-Draft -04, 2026-05-21 | Enterprise-managed authorisation for agents; the basis of MCP's stable extension (section 6). |
| `draft-ietf-oauth-identity-chaining` | Internet-Draft -17, 2026-07-19; reportedly approved by the IESG **[unverified]** | Cross-domain chaining of token exchange and JWT bearer grants. |
| `draft-ietf-oauth-transaction-tokens` | Internet-Draft -08, 2026-03-02 ([draft](https://datatracker.ietf.org/doc/draft-ietf-oauth-transaction-tokens/08/)) | Short-lived, call-chain-scoped tokens inside a customer's workload graph. A later phase. |

## 3. OpenID Connect and the OpenID Foundation's specifications

| Specification | State | Consequence |
|---|---|---|
| OpenID Connect Core, Discovery, Dynamic Registration, Form Post response mode | Final (2014–2015), with errata sets | The certification profiles of section 8. RS256 is **mandatory to implement** for an OpenID Provider (Core §15.1), which shapes the algorithm set in `docs/research/architecture-decisions.md`. |
| RP-Initiated Logout, Front-Channel Logout, Back-Channel Logout, Session Management | Final (2022) | All three logout profiles are v1 scope; Session Management (the iframe-based one) is not, because third-party cookie restrictions in browsers make it unreliable. It stays in the "beyond 1.0.0" register rather than being dropped. |
| FAPI 2.0 Security Profile and Attacker Model | Final, February 2025 ([announcement](https://openid.net/fapi-2-security-profile-attacker-model-final-specifications-approved/)) | A v1 profile, not a "planned" one: PAR, PKCE, sender-constrained tokens (DPoP or mTLS), `iss`, short-lived codes. |
| FAPI 2.0 Message Signing | Final, vote 2025-09-10 to 24 ([announcement](https://openid.net/fapi-2-message-signing-final-specification-approved/)) | JAR, JARM, signed introspection responses. v1. |
| Shared Signals Framework 1.0, CAEP 1.0, RISC 1.0 | **Final, 2025-09-02** ([announcement](https://openid.net/three-shared-signals-final-specifications-approved/)) | The server is an SSF transmitter (session revoked, credential changed, assurance level changed, account disabled) and a receiver (a customer's IdP or device manager revokes our sessions). v1. |
| OpenID Federation 1.0 | Final, 2026-02-17 ([announcement](https://openid.net/openid-federation-1-0-final-specification-approved/)); 1.1 final after a vote of 2026-04-21 to 05-05 | Trust chains instead of bilateral registration; relevant for education and government federations. Beyond 1.0.0, with the client-registration interface ready for it. |
| OpenID for Verifiable Presentations 1.0 / Verifiable Credential Issuance 1.0 | Final, July and September 2025 ([OID4VP](https://openid.net/openid-for-verifiable-presentations-1-0-final-specification-approved/), [OID4VCI](https://openid.net/openid-for-verifiable-credential-issuance-1-final-specification-approved/)); self-certification open since 2026-02-26 | Wallet-based credentials (EU digital identity wallet, mobile driving licences). Beyond 1.0.0; the credential model does not preclude it. |

## 4. SAML

SAML 2.0 is not deprecated by OASIS, and enterprise single sign-on still runs on it; the trend is
"keep SAML for what exists, build new integrations on OIDC". The product implements both roles
(identity provider and service provider, and brokering between them) because the enterprise layer
is useless without it.

The risk is not the protocol but XML signature processing: signature wrapping, comment injection
into canonicalisation, external entities. The Go ecosystem's SAML libraries are
`crewjam/saml` (BSD-2-Clause, last release 2025-04, effectively in maintenance) and
`russellhaering/gosaml2` with `goxmldsig` (Apache-2.0, maintained, release 2026-08). The decision
on how this project handles XML signatures is in `docs/research/architecture-decisions.md`.

## 5. SCIM

SCIM 2.0 (RFC 7642–7644) is the provisioning standard. **SCIM Events** was published as
**RFC 9967** in May 2026 ([RFC 9967](https://datatracker.ietf.org/doc/rfc9967/)), a profile for
security event tokens that updates RFC 7643 and 7644: provisioning changes can be pushed as events
rather than only polled. The product implements a SCIM server (customers' directories push users in)
and a SCIM client (the product pushes users to downstream applications), and the event profile, in
v1.

## 6. Authorisation for AI agents and MCP

The Model Context Protocol's authorisation specification moved twice since the briefing's sources
were written. The current revision is **2026-07-28**
([changelog](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/changelog.mdx),
[announcement](https://blog.modelcontextprotocol.io/posts/2026-07-28/)).

What it requires of an authorisation server serving MCP servers:
- OAuth 2.1 with PKCE; RFC 8414 or OpenID discovery; audience validation through RFC 8707 resource
  indicators; the MCP server publishes RFC 9728 protected resource metadata.
- **Client ID Metadata Documents are the preferred registration method; RFC 7591 dynamic client
  registration is deprecated** and kept for backward compatibility.
- The authorisation server **should** return `iss` (RFC 9207) and clients **must** validate it; the
  change log anticipates making it a MUST for servers. This server always returns it.
- Clients must send `application_type` in dynamic registration; client credentials are bound to the
  issuing authorisation server and never reused across servers.
- A formal extensions framework, whose first stable extension is **Enterprise-Managed
  Authorization**: the client exchanges the user's identity assertion at the enterprise IdP for an
  **ID-JAG** (RFC 8693 token exchange), then presents it with a JWT bearer grant to the MCP server's
  authorisation server ([extension](https://github.com/modelcontextprotocol/ext-auth/blob/main/specification/stable/enterprise-managed-authorization.mdx)).
  This server must play both roles: the enterprise IdP that issues ID-JAGs and the authorisation
  server that accepts them.

Other work on agent identity, none of it final:
- **AAuth** (`draft-hardt-oauth-aauth-protocol-10`, 2026-08-06), an individual draft for agent
  identity through HTTP Message Signatures and proof of possession.
- **WIMSE** (workload identity in multi-system environments): architecture, service-to-service
  protocol, workload credentials; every document still a draft
  ([workload credentials](https://datatracker.ietf.org/doc/draft-ietf-wimse-workload-creds/)).
- The OpenID Foundation's **AI Identity Management community group** published a white paper
  (October 2025) and answered NIST's request for information on agent security (March 2026). A
  community group produces no specifications.

Consequence: the agent surface of v1 is built on what is final or stable — token exchange, RAR,
DPoP, CIMD, the MCP authorisation specification and its enterprise-managed extension — and the
drafts above are tracked in the "beyond 1.0.0" register with the interfaces (token exchange
policies, workload credential verifiers) that let them arrive as additions.

## 7. Passkeys and WebAuthn

- **WebAuthn Level 3 became a W3C Recommendation on 2026-08-25**
  ([W3C news](https://www.w3.org/news/2026/web-authentication-an-api-for-accessing-public-key-credentials-level-3-is-now-a-w3c-recommendation/)).
  It normatively covers conditional create (upgrading a password sign-in to a passkey silently),
  the Signal API (telling the authenticator that a credential was deleted or a user renamed),
  Related Origin Requests (one passkey across several domains of the same party) and the PRF
  extension. All four are v1 scope.
- **FIDO Credential Exchange Format** is a Proposed Standard (v1.0, 2026-03-09); the **Credential
  Exchange Protocol** is still a working draft. Credential managers use them to move passkeys
  between each other; a relying party has nothing to implement, but the documentation must not
  promise that a passkey is tied to one device.
- **Related Origin Requests** matter to the hosting model: a tenant with several domains can share
  passkeys across them through a `/.well-known/webauthn` document the server publishes per tenant.
- The Go library `go-webauthn/webauthn` (BSD-3-Clause) is active (v0.18.2, 2026-09-19).

## 8. Certification

The OpenID Foundation runs the conformance suite as open source
([conformance suite](https://gitlab.com/openid/conformance-suite)); anyone may run it, locally with
Docker or against the hosted instance. Certification is the separate act of submitting results and
paying a fee ([fees](https://openid.net/certification/fees/)):

| Programme | Non-member | Member |
|---|---|---|
| OpenID Connect (any number of profiles, per deployment, per calendar year) | US$ 3,500 | US$ 700 |
| FAPI 1 or FAPI 2 (per deployment; provider and relying party separately) | US$ 5,000 | US$ 1,000 |

Profiles offered for providers: Basic, Implicit, Hybrid, Config, Dynamic, Form Post; logout:
RP-Initiated, Session Management, Front-Channel, Back-Channel; FAPI 1 and 2, FAPI-CIBA, Shared
Signals, Federation, OID4VCI/VP and HAIP. The FAQ describes an "online deployment" as the thing
certified; whether results from a privately hosted suite are accepted is **[unverified]**.

Certification is sometimes a regulatory requirement rather than a signal. **Open Finance Brasil**
requires every participant's authorisation server to be certified by the OpenID Foundation against
its own profile, "FAPI-BR v2", which is FAPI 1 Advanced plus Brazilian requirements (PAR, PKCE,
dynamic registration with directory software statements, a `consent:{id}` scope, CIBA for payments)
([certification listing](https://openid.net/certification/certified-fapi1-rp-brazil-open-finance-fapi-br-v2/)).
No timeline for moving it to FAPI 2.0 was found **[unverified]**. It belongs to the Brazilian
differentiators after 1.0.0; the FAPI 2.0 building blocks of v1 (PAR, JAR, JARM, DPoP, mTLS) are
what that profile is assembled from.

Decision recorded with the owner on 2026-09-25: **the suites run in CI from phase 1 and must be
green for v1.0.0; paid certification waits for the commercial launch**
(`docs/research/licensing.md`, section 6).

## 9. Cryptography in Go

- The current Go release is **1.27** (2026-08-19); the latest patch releases on 2026-09-01 are
  `go1.27.1` and `go1.26.8` ([tags](https://github.com/golang/go/tags)).
- The **Go Cryptographic Module v1.0.0** (Go 1.24+) is FIPS 140-3 validated, CMVP certificate
  **#5247**. Module **v1.26.0** (adds ML-DSA) is "pending review" since 2026-04-28 and is selected
  with `GOFIPS140=v1.26.0` ([Go FIPS 140](https://go.dev/doc/security/fips140)). No module snapshot
  tied to 1.27 was found **[unverified]**.
- **argon2id is not a FIPS-approved algorithm.** A deployment in FIPS mode must hash passwords with
  PBKDF2 (HMAC-SHA-512). The consequence is a FIPS *profile* that changes the password hash, not a
  switch that leaves argon2id in place (`docs/research/architecture-decisions.md`).

## 10. What changed relative to the briefing

- OAuth 2.1 is still a draft; RFC 9700 is the normative anchor.
- FAPI 2.0 is final and certifiable, so it is a v1 profile rather than a planned one.
- Shared Signals, CAEP and RISC are final; the product transmits and receives them in v1.
- MCP now prefers client ID metadata documents over dynamic registration, requires `iss` validation
  by clients, and has a stable enterprise-managed extension built on ID-JAG.
- WebAuthn Level 3 is a Recommendation, with conditional create, the Signal API and Related Origin
  Requests normative.
- SCIM has a published event profile, RFC 9967.
- OpenID Federation and the OpenID for Verifiable Credentials family are final; both are beyond
  1.0.0 with their extension points ready.
