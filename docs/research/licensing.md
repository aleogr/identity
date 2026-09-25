# Research: licensing, monetisation and contribution governance

> **Status:** research note written in the inaugural design session of 2026-09-25, recording the
> owner's decision on licensing and the reasoning presented. **This is not legal advice**; the texts
> of the licences, the contributor licence agreement and the commercial licence are reviewed by a
> lawyer before they are relied on commercially (section 6).
>
> Market facts are sourced in `docs/research/market.md`, section 2.3.

---

## 1. The problem

The project is public from its first commit and is meant to generate revenue. Its licence must at
the same time:
- let anyone **embed** the library and the SDKs without obligations that would stop adoption;
- stop a third party from **taking the server and selling it as a service** without contributing
  back or paying;
- keep the owner able to **sell a commercial licence** to those who cannot accept the copyleft
  terms, which requires holding the rights to every line, including contributors';
- never give buyers of a security product a reason to distrust it.

## 2. Options considered

| Option | Precedent in identity | For | Against |
|---|---|---|---|
| (a) AGPL-3.0 everywhere, plus a commercial licence | Zitadel since v3 (2025); Hanko's backend | Strong protection against hosted resale | Kills embedding: an application that links an AGPL library in a network service inherits the obligation. The library and SDKs lose their market |
| (b) Open core: Apache-2.0 core, proprietary enterprise features | Ory (Ory Enterprise License), SuperTokens (`ee/`), authentik (`enterprise/`), Keycloak via Red Hat | Familiar; easy to embed | The line lands on features, often security ones (the "SSO tax"). Ory's enterprise builds receive CVE fixes first, which buyers read as the open-source build being less safe |
| (c) FSL or BSL, converting to Apache/MIT after a delay | Sentry (FSL, 2023); no prominent identity server found | Protection against competing hosted use for a period | Not an OSI licence during the period; identity buyers weigh lock-in and longevity heavily |
| (d) Hybrid: Apache-2.0 for what is embedded, AGPL-3.0 plus a commercial licence for what is run as a service | Zitadel's own split is close (Apache `proto/`, MIT login and clients, AGPL server) | Embedding stays free of obligations; hosted resale is covered; no feature is withheld | Two licences to explain; requires contributor rights to be clean (section 4) |

## 3. The decision

**Status:** accepted by the owner, option **(d)**.

| Part of the repository | Licence |
|---|---|
| `spec/`, `core/`, `adapters/`, `conformance/`, the SDKs, the plugin development kits, the library | **Apache-2.0** |
| The server, `ui/` (hosted login, admin console, UI components), the cloud control plane | **AGPL-3.0-only**, or a **commercial licence** for those who do not accept the AGPL |

Apache-2.0 code may be combined into an AGPL-3.0 work, so the server can use the core and adapters
freely. Each module carries its own `LICENSE` file, and every source file an SPDX identifier, so a
consumer can tell from any file which licence governs it.

### 3.1 Consequences for each kind of user

- **Someone who embeds the library or uses an SDK:** no obligation beyond Apache-2.0's (keep the
  notice), with Apache-2.0's patent grant. Their application's licence is unaffected.
- **Someone who runs the unmodified server, for themselves or their customers:** free. The AGPL
  obliges them only to offer the source of what they run, which is this repository.
- **Someone who modifies the server and offers it over a network:** must publish those modifications
  under the AGPL, or buy the commercial licence.
- **Someone who builds an identity service on the Apache-2.0 core with their own server:** allowed.
  The moat is the complete server, UI, operations and certification, not the core's protocols. This
  is accepted consciously: a permissive core is what makes the library and the SDKs adoptable.

### 3.2 The line between free and paid

- **Free, for everyone:** every capability of the product. **No security feature is paid** (single
  sign-on, MFA, passkeys, audit logs, SCIM included), and **no security fix reaches paying customers
  before the public repository.**
- **Paid:**
  1. the **managed cloud** — the product operated as a service;
  2. the **commercial licence** — an exemption from the AGPL for the server and UI;
  3. **long-term support** builds with a support agreement and service levels.

## 4. Contribution governance

**Status:** accepted by the owner: **a contributor licence agreement from the first external
contributor**, one rule for the whole repository.

**Why a CLA and not only the DCO.** Selling a commercial licence for the AGPL parts requires the
right to relicense every contribution. The Developer Certificate of Origin certifies that the
contributor may submit the code under the project's licence; it grants nothing beyond that licence,
so an AGPL contribution could not be sold under a commercial licence.

**The alternative considered: inbound Apache-2.0** (Zitadel's model). Contributions are accepted
under Apache-2.0, whose permissions let the project redistribute them under the AGPL or a commercial
licence, with no CLA to sign. It is simpler for contributors. It was not chosen because it lacks two
things a CLA provides: an explicit representation that the contributor's employer allows the
contribution, and a grant broad enough to cover future licensing decisions without returning to every
contributor.

**The form of the agreement.** A **licence grant, not a copyright assignment**: the contributor keeps
the copyright and grants the project a perpetual, irrevocable, worldwide licence to use, modify and
sublicense, plus a patent licence. The common base texts are the Apache Individual CLA v2.2 and the
Harmony HA-CLA-I; the text is chosen with the lawyer (section 6).

**Tooling, checked on 2026-09-25.** CLA Assistant (SAP's hosted service) still runs, but its
repository has had no commit since October 2023; the CLA Assistant Lite GitHub Action was archived on
2026-03-23; the Linux Foundation's EasyCLA serves mainly projects it hosts. The choice of tool is made
when the first external pull request arrives. Until then, `CONTRIBUTING.md` states that external
contributions cannot be merged before the agreement exists, so no contribution enters with unclear
rights.

## 5. Brand and trademark

A licence covers code, not the name. When the brand is chosen (`docs/research/market.md`, section 5)
the project publishes a short trademark policy: forks may say "based on" but may not use the name or
logo for their own product. Third parties' product names appear only in "compatible with" or "import
from" wording, never in module, package or product names.

## 6. External costs and their moment

**Status:** accepted by the owner. **None of these is needed while the owner's own projects are the
only clients.** They exist to convince third parties or to allow selling and receiving contributions.
The implementation is built so that each is ready to be paid for, never blocked on, and they are paid
when the owner decides to launch commercially.

| Item | What it is | Price found on 2026-09-25 | Moment |
|---|---|---|---|
| OpenID certification (Connect, all profiles) | Submitting the suite's results to the OpenID Foundation for the public "OpenID Certified" listing. Running the suite is free and happens in CI | US$ 3,500 per deployment per calendar year for non-members; US$ 700 for members | Commercial launch |
| FAPI 2.0 certification | The same for FAPI | US$ 5,000 non-member; US$ 1,000 member | Commercial launch, or earlier if a regulated customer asks |
| OpenID Foundation membership | Lowers certification fees to 20% and gives a vote on specifications | Current amounts **[unverified]**; sustaining members pay US$ 50,000 a year, lower tiers exist | Only if the discount pays for itself at certification time |
| External security audit | An independent firm reviews the code and attacks the deployed system, and publishes a report | Not disclosed by published identity audits; OSTIF puts first audits at US$ 30k–200k | Before the first external customer. Sponsored audits (OSTIF, Alpha-Omega) are worth applying for once the project has users |
| Legal review | The CLA text, the commercial licence, the terms of the managed cloud | To be quoted | The CLA before the first external contribution is merged; the rest before the first sale |

The milestone structure that follows from this:

- **1.0.0** is the complete product. Every conformance suite (OIDC, logout, FAPI 2.0, SAML
  interoperability, WebAuthn) is green in CI; the threat model is current; a full internal security
  review is done; the owner's projects may go to production on it. **No external cost.**
- The **commercial launch** adds the paid items above, when the owner funds them.

One caution is recorded with the decision: the owner's projects will have real users, whose data
depends on this product even without a paying customer. The internal review before 1.0.0 is therefore
not a formality: threat model, continuous fuzzing of every parser, a security review on every pull
request, static analysis and dependency scanning in CI.
