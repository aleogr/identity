# Research: the customer identity market

> **Status:** research note written in the inaugural design session of 2026-09-25. It describes the
> market the product enters and the open-source building blocks available in Go, as they stood on
> that date. `docs/requirements.md` and `docs/design.md` cite it; they, not this note, are the
> sources of truth for decisions.
>
> **Method:** web research on 2026-09-25. Licences were checked against the projects' raw `LICENSE`
> files and module versions against `proxy.golang.org`; vendors' announcements are linked. Facts
> that rest on memory or on a single weak source are marked **[unverified]**.

---

## 1. The five categories, and what separates them

The briefing organised the market in five categories. They hold up, with one observation that
drives the architecture: **the categories differ by buyer, packaging and deployment, not by
authentication logic.**

| Category | Representative products | Buyer | What they sell beyond the protocols |
|---|---|---|---|
| Full CIAM | Auth0 (Okta Customer Identity), Microsoft Entra External ID | Product and security teams at larger companies | Breadth of protocols, extensibility (actions, hooks), compliance attestations, B2B and B2C in one tenant |
| Developer-first | Clerk, Stytch, Descope, Kinde | Application developers, start-ups | UI components, per-framework SDKs, sign-up to working login in minutes |
| Enterprise B2B layer | WorkOS | SaaS companies selling to enterprises | SAML/OIDC SSO, SCIM directory sync, a white-label admin portal for the customer's IT team, audit logs; runs beside another identity provider |
| Embedded in a platform | Amazon Cognito, Firebase Authentication / Identity Platform, Supabase Auth | Teams already on that platform | Integration with the platform's database, functions and billing |
| Open source and self-hosted, auth as a library | Keycloak, Zitadel, Ory, FusionAuth, authentik, Logto, SuperTokens, Better Auth | Teams that must or want to run it themselves | Control, data residency, no per-user price |

The shared part is large: users, credentials, sessions, the OAuth/OIDC authorisation server, SAML,
WebAuthn, MFA, organisations, provisioning, events. What does *not* share is not authentication
logic either: it is UI components and per-framework SDKs (developer-first), the operation of a
multi-tenant service with billing and self-service sign-up (every hosted category), and the
in-process packaging of a library. That is the argument for three **forms of execution** — library,
server, cloud control plane — with the categories as editions and profiles of the server
(`docs/research/architecture-decisions.md`, decision 1).

Out of scope, by the owner's decision: workforce IAM (the classic Okta), privileged access
management, identity governance and administration, mobile device management.

## 2. Competitors, 2025–2026

### 2.1 Hosted, proprietary

- **Auth0 (Okta).** Proprietary service; SDKs open source. *Auth0 for AI Agents* became generally
  available in November 2025, with **Token Vault** — storage and refresh of third-party OAuth tokens
  on the user's behalf, 35+ integrations
  ([announcement](https://auth0.com/blog/auth0-for-ai-agents-generally-available/)); Token Vault
  with organisation support went GA in June 2026
  ([announcement](https://auth0.com/blog/token-vault-with-organization-support-generally-available/)).
  Okta's **Cross App Access** became an official MCP authorisation extension, and **Okta Agent SSO**
  went GA on 2026-08-24
  ([press release](https://www.okta.com/newsroom/press-releases/okta-brings-first-class-identity-to-ai-agents-with-agent-sso/)).
- **WorkOS.** Proprietary service. AuthKit as an OAuth 2.1 authorisation server for MCP, with
  dynamic registration, launched as a preview on 2025-05-23
  ([changelog](https://workos.com/changelog/mcp-authorization-with-authkit)); whether it left
  preview is **[unverified]**. US$ 100M Series C at a US$ 2B valuation in March 2026
  ([announcement](https://workos.com/blog/series-c)).
- **Clerk.** Proprietary service. US$ 50M Series C on 2025-10-15, with "Agent Identity" as a new
  product pillar and an agent toolkit for the Vercel AI SDK and LangChain
  ([announcement](https://clerk.com/blog/series-c)).
- **Stytch.** Acquired by **Twilio**, announced 2025-10-30, closed 2025-11-14
  ([announcement](https://www.twilio.com/en-us/blog/company/news/twilio-to-acquire-stytch)).
  *Connected Apps* for remote MCP servers launched 2025-04-11.
- **Descope.** Proprietary, no-code flows. *Agentic Identity Hub* launched April 2025, 2.0 on
  2026-01-26, 2.5 in June 2026
  ([press release](https://www.descope.com/press-release/agentic-identity-hub-2.0)).
- **Kinde.** Proprietary service. In 2026 it added SSO and SCIM to its free B2B tier (February),
  passkeys (June) and e-mail codes (August) ([updates](https://updates.kinde.com/)). Funding and
  agent features **[unverified]**.

### 2.2 Embedded in a platform

- **Supabase Auth.** A fork of GoTrue, in Go, MIT. An OAuth 2.1 / OIDC server entered public beta
  in November 2025, conforming to the MCP authorisation specification and included in the
  self-hosted image ([docs](https://supabase.com/docs/guides/auth/oauth-server)).
- **Amazon Cognito, Firebase / Identity Platform.** Proprietary; relevant to this project mainly as
  migration sources (section 4).

### 2.3 Open source and self-hosted

| Project | Language | Licence and model | 2025–2026 |
|---|---|---|---|
| **Keycloak** | Java (Quarkus) | Apache-2.0; CNCF Incubating; Red Hat sells a supported build | 26.x line (26.4 in September 2025 to 26.7 in July 2026): MCP authorisation-server documentation, **experimental CIMD**, Token Exchange v2, zero-downtime updates ([26.6](https://www.keycloak.org/2026/04/keycloak-2660-released), [MCP](https://www.keycloak.org/securing-apps/mcp-authz-server)) |
| **Zitadel** | Go | **Moved from Apache-2.0 to AGPL-3.0-only with v3** (2025-03-31); `proto/` and docs stay Apache-2.0, the login app and client packages MIT; a commercial licence is sold. **No CLA**: contributions are accepted under Apache-2.0 ([blog](https://zitadel.com/blog/apache-to-agpl), [LICENSING.md](https://github.com/zitadel/zitadel/blob/main/LICENSING.md)) | **v3 removed CockroachDB and supports PostgreSQL only** ([PR #9444](https://github.com/zitadel/zitadel/pull/9444), merged 2025-03-18; [v3.0.0](https://github.com/zitadel/zitadel/releases/tag/v3.0.0), 2025-05-02) |
| **Ory** (Kratos, Hydra, Keto, Oathkeeper, Polis) | Go (Polis in TypeScript) | Apache-2.0 open source plus the **Ory Enterprise License**, whose README promises "guaranteed CVE fixes, current enterprise builds": the open-source builds lag. CLA through cla-assistant | Acquired BoxyHQ on 2025-05-29, now **Ory Polis** (SAML-to-OIDC bridge, directory sync) ([blog](https://www.ory.com/blog/introducing-ory-polis-for-enterprise-single-sign-on)). Module tags on the Go proxy are stale (Kratos v1.3.1 of October 2024, Hydra v2.3.0 of January 2025) while Ory's changelog announces v26.x releases |
| **FusionAuth** | Java | Core **proprietary**, a free "Community" edition that is not OSI open source; SDKs Apache-2.0 ([licence FAQ](https://fusionauth.io/license-faq)) | Nothing notable found **[unverified]** |
| **authentik** | Python (Django), Go outpost | MIT except `authentik/enterprise/` under an enterprise licence | Release 2026.8.2 (September 2026) |
| **Logto** | TypeScript | MPL-2.0 plus Logto Cloud | Co-maintains the "MCP Auth" library |
| **SuperTokens** | Java core; Node, Python, Go SDKs | Apache-2.0 core; `ee/` needs a commercial licence in production | Nothing notable found **[unverified]** |
| **Better Auth** | TypeScript | MIT | YC X25, US$ 5M seed in June 2025 ([TechCrunch](https://techcrunch.com/2025/06/25/this-self-taught-ethiopian-dev-built-an-authentication-tool-and-got-into-yc/)); an MCP plugin built on its OAuth provider plugin (OAuth 2.1, RFC 9728, CIMD) ([docs](https://better-auth.com/docs/plugins/mcp)) |
| **Authelia** | Go | Apache-2.0, community-run, no paid tier | v4.39.0 (2025-03-16) is **OpenID Certified** ([blog](https://www.authelia.com/blog/we-are-now-openid-certified/)) |
| **Casdoor** | Go (Beego), React | Apache-2.0 | Rebranded "agent-first", with an MCP gateway |
| **Janssen / Gluu** | Java; Cedarling policy engine in Rust/WASM | Apache-2.0 (Linux Foundation); Gluu Flex is the commercial distribution | — |

### 2.4 What the market tells this project

1. **Agent and MCP authorisation became table stakes within a year.** Every serious vendor shipped
   an MCP authorisation server, a token vault or both between April 2025 and August 2026. It
   belongs in v1, not in a later phase.
2. **The Go field is thin at the top.** Zitadel is the only complete Go identity server with
   momentum, and it is now AGPL; Ory's open-source builds lag its commercial ones; Supabase Auth
   and Authelia are narrower. A complete, permissively embeddable Go core has no incumbent.
3. **Licence changes are a trust event.** Zitadel's move to AGPL and Ory's lagging open-source
   builds are both discussed by buyers. Choosing the licence once, before the first external
   contributor, and never gating security fixes, is part of the product
   (`docs/research/licensing.md`).
4. **PostgreSQL-only is where the Go incumbent landed.** Zitadel dropped CockroachDB to focus on
   one database; that supports the storage decision in `docs/research/architecture-decisions.md`.
5. **Certification is a differentiator that costs little to earn technically.** Authelia, a
   community project, is OpenID Certified; most developer-first vendors are not.

## 3. Go building blocks

Versions and dates from `proxy.golang.org`; licences from the raw `LICENSE` files.

| Module | Licence | Latest | Status | Use in this project |
|---|---|---|---|---|
| `github.com/zitadel/oidc/v3` | Apache-2.0 | v3.51.6, 2026-09-25 | Very active; certified relying-party and provider library | **Test oracle** (an independent client and provider to interoperate with), not a dependency of the server |
| `github.com/ory/fosite` | Apache-2.0 | v0.49.0, 2024-12-12 | Stale as a standalone module; development appears to have moved inside Ory's v26 line **[unverified]** | Not used; reference reading only |
| `github.com/go-webauthn/webauthn` | BSD-3-Clause | v0.18.2, 2026-09-19 | Active | Candidate for the WebAuthn ceremony verification, behind the project's own interface |
| `github.com/crewjam/saml` | BSD-2-Clause | v0.5.1, 2025-04-14 | Maintenance mode | Test oracle only |
| `github.com/russellhaering/gosaml2` / `goxmldsig` | Apache-2.0 | v0.12.0, 2026-08-05 | Maintained | Candidate for XML signature verification; see the architecture decisions |
| `github.com/openfga/openfga` | Apache-2.0 | v1.21.0, 2026-09-20 | Active; CNCF Incubating since 2025-10-28; embeddable as `pkg/server` | Compatibility oracle for the relationship-based engine beyond 1.0.0 |
| `github.com/cedar-policy/cedar-go` | Apache-2.0 | v1.8.0, 2026-06-01 | Active; Cedar is CNCF Sandbox since 2025-10-08; schema validation and batch evaluation still under `x/exp` | Not the main engine; candidate for policy conditions |
| `github.com/casbin/casbin` | Apache-2.0 | v3.10.0, 2026-01 | Active | Not used |
| `github.com/authzed/spicedb` | Apache-2.0 | v1.56.2, 2026-09-11 | Active; built to run as a gRPC service; in-process embedding is not a supported public API **[unverified]** | Not used |
| `github.com/wazero/wazero` | Apache-2.0 | v1.12.0, 2026-05-29 | Active; WASI preview 1 only; the Component Model is not implemented and the project waits for it to be standardised | The WebAssembly runtime for plugins |
| `github.com/extism/go-sdk` | BSD-3-Clause **[unverified]** | v1.7.1, 2025-03-19 | Built on wazero (pins v1.9.0); no release in 18 months | Not used; see the architecture decisions |

## 4. Migration sources

The migration capability (hash verifiers, importers, lazy migration, coexistence, export) is sized by
where customers come from. The sources the briefing listed are all still relevant; the market moves
of 2025–2026 add two reasons to treat migration as a product rather than a script:
- **Acquisitions push customers to move.** Stytch's customers now depend on Twilio's roadmap;
  BoxyHQ's on Ory's.
- **Licence changes push self-hosters to move.** Zitadel's AGPL move and Ory's lagging open-source
  builds both create a population evaluating alternatives.

The minimum set that proves the architecture generalises is in
`docs/research/architecture-decisions.md`, decision 12.

## 5. Names

The Go module path is fixed (`github.com/aleogr/identity`) and needs no brand. A brand is needed at
the first public npm package, public container image and Terraform provider. Prominent collisions in
the identity space found during this research, to avoid when the brand is chosen:
- **Severe:** Keystone (OpenStack's identity service), Warden (the Ruby authentication middleware
  under Devise), Heimdall (a Go identity-aware proxy), Aegis (a popular TOTP authenticator app).
- **Moderate and crowded:** Sigil, Portcullis (several MCP gateways use it), Janus, Tessera (Google's
  Go transparency-log library).
- **Clean in identity:** Ostium.

The Keystone, Warden, Aegis and Janus notes are from memory **[unverified]**; the choice itself is
recorded as an open item in `docs/requirements.md`, with a formal collision check on GitHub,
`pkg.go.dev`, npm and trademark registers before the first public package.
