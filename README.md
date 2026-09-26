# identity

An open-source identity platform in Go: authentication, authorisation, user and organisation
management, provisioning, migration, and identity for AI agents — as a library, as a server, and later
as a managed cloud, all from one core.

> **Status: design complete, implementation starting.** Nothing here is usable yet. The product is
> released as `1.0.0` only when it is complete; until then versions are `0.y.z` and interfaces change.

## What it will be

- **An OpenID Connect and OAuth 2.0 authorisation server** following RFC 9700 by default, with PAR,
  JAR, DPoP, mutual TLS, token exchange, device flow and the FAPI 2.0 profiles, conformance-tested
  against the OpenID Foundation's suites.
- **Passkeys first**, passwords under NIST SP 800-63B-4, multi-factor authentication, recovery with a
  security delay.
- **Organisations** with roles, their own SSO over SAML or OIDC, and SCIM provisioning.
- **Shared Signals** (CAEP and RISC) as transmitter and receiver.
- **Authorisation for MCP servers and AI agents**, including enterprise-managed authorisation and a
  token vault.
- **Extensibility** through Go interfaces, sandboxed WebAssembly plugins and signed webhooks.
- **Migration** from other identity providers, and complete export.
- **Everything is API:** the console, the CLI, a declarative file and a Terraform provider are facades of
  the same API.
- Multi-tenant from the data model up, with row-level security on every table.

## Documents

| Document | What it decides |
|---|---|
| [`docs/requirements.md`](docs/requirements.md) | What the product is, the scope of `1.0.0` and what comes after |
| [`docs/design.md`](docs/design.md) | How it is built, and the phases |
| [`docs/threat-model.md`](docs/threat-model.md) | The threats and how each is mitigated and verified |
| [`docs/roadmap.md`](docs/roadmap.md) | The deliveries of the phase under way |
| [`docs/research/`](docs/research/) | Why: standards, market, architecture decisions, licensing |

## Licence

Two licences, by what a part is for (see [`LICENSING.md`](LICENSING.md)):

- **Apache-2.0** for everything you embed: the specification, the core libraries, the adapters, the
  conformance suite, the SDKs, the plugin kits and the library.
- **AGPL-3.0-only**, or a commercial licence, for the server, its user interface and the cloud control
  plane.

## Security

Report vulnerabilities privately through GitHub; see [`SECURITY.md`](SECURITY.md).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). External contributions require a contributor licence
agreement, which does not exist yet; until it does, external pull requests cannot be merged.
