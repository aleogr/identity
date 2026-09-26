---
name: spec-catalog
description: Where to find, how to cite and how to track the standards this identity platform implements (IETF RFCs and drafts, OpenID Foundation specifications, W3C WebAuthn, FIDO, OASIS SAML, SCIM, MCP). Use when writing a spec, plan, code or test that implements or depends on a protocol rule, before stating what a standard requires.
---

# Standards catalogue

The product's behaviour is defined by standards, and a rule quoted from memory is a bug waiting to
happen. Use this skill whenever a spec, plan, test or piece of code depends on what a standard says.

## Rules

1. **Cite the section, not the document.** Write "RFC 9700 §2.1.1" or "OIDC Core §3.1.2.1", never just
   "the RFC". Tests that check a normative rule name the section in the test's name or a comment.
2. **Normative words decide.** MUST, MUST NOT, REQUIRED, SHALL are implemented as behaviour and tested,
   including the negative case. SHOULD is implemented unless `docs/requirements.md` records why not.
   MAY is implemented only when the requirements ask for it.
3. **This project's defaults are floors.** Where a standard allows a weaker option (the implicit grant,
   PKCE `plain`, HMAC-signed tokens), the product refuses it unless `docs/requirements.md` says
   otherwise.
4. **Verify the current version before relying on a draft.** Internet-Drafts change and expire. For a
   draft, record in the spec the revision number and date read. `docs/research/standards.md` holds the
   state on its date; if more than a few months have passed, check again and update it in the same pull
   request.
5. **Read the primary source.** Prefer the RFC Editor, the IETF datatracker, openid.net, w3.org,
   fidoalliance.org, oasis-open.org and the specification's own repository. When a site is unreachable
   from the session, use a mirror of the specification's source (for example its GitHub repository) and
   say so.

## Where each family lives

| Family | Primary source | In this product |
|---|---|---|
| OAuth 2.0 and extensions (RFC 6749, 7009, 7591/7592, 7636, 7662, 8252, 8414, 8628, 8693, 8705, 8707, 9068, 9101, 9126, 9207, 9396, 9449, 9470, 9700, 9728) and drafts (OAuth 2.1, CIMD, ID-JAG, transaction tokens, identity chaining) | `https://www.rfc-editor.org/rfc/rfcNNNN`, `https://datatracker.ietf.org/doc/<draft-name>/` | `core/oauth`, phases 1 and 5 |
| OpenID Connect Core, Discovery, Registration, Form Post, the logout specifications, FAPI 2.0, Shared Signals (SSF, CAEP, RISC), Federation, OID4VC | `https://openid.net/specs/` and `https://openid.net/wg/` | `core/oidc`, `core/ssf`; phases 1, 2, 5, 7 |
| OpenID conformance suite | `https://gitlab.com/openid/conformance-suite` | CI, from F16 |
| WebAuthn Level 3 | `https://www.w3.org/TR/webauthn-3/` | `core/webauthn`, phase 1 |
| FIDO (CTAP, metadata service, credential exchange) | `https://fidoalliance.org/specifications/` | attestation policies, phase 1 |
| SAML 2.0 | OASIS SAML v2.0 standard documents | `core/saml`, phase 6 |
| SCIM 2.0 (RFC 7643, 7644) and SCIM events (RFC 9967) | RFC Editor | `core/scim`, phase 7 |
| MCP authorisation | `https://modelcontextprotocol.io/specification/` and the `modelcontextprotocol` GitHub repository | `core/agents`, phase 9 |
| NIST SP 800-63-4 (digital identity guidelines) | `https://pages.nist.gov/800-63-4/` | password and authenticator policy |

## When writing a delivery's spec

List the specifications the delivery implements, each with its version or revision and the sections
that matter, in a "Standards" section of the spec. The plan's tests then trace to those sections.
