# Contributing

Thank you for your interest. Please read this before opening an issue or a pull request.

## The state of the project

The project is at the start of implementation and is developed by one maintainer. Design decisions are
recorded in [`docs/`](docs/) and are not reopened in pull requests; if you think one is wrong, open an
issue that explains the new fact or argument.

## Contributor licence agreement

Parts of this repository are offered under the AGPL-3.0 **and** a commercial licence
([`LICENSING.md`](LICENSING.md)). To be able to do that, the project needs every contribution under a
**contributor licence agreement**: a licence grant, not a transfer of copyright. You keep the copyright
to your work.

**The agreement does not exist yet.** Until it is published and a signing process is in place,
**external pull requests cannot be merged**, however good they are. Issues, bug reports and design
discussions are welcome now.

## Reporting a vulnerability

Never in a public issue. Follow [`SECURITY.md`](SECURITY.md).

## How changes are made

- One topic per pull request, against `main`.
- Everything versioned is written in English. User-facing text goes through translation keys and exists
  in every supported language (en-US and pt-BR).
- The checks CI runs must pass: `make check`, `make test`, `make integration`, `make e2e`, and the
  image scan and CodeQL in CI. No test is skipped or disabled to make a pull request pass.
- Security defaults are floors: no change may add a way to configure the product below them.
- A change that adds a trust boundary, a parser, an outbound call, a secret or an extension point
  updates [`docs/threat-model.md`](docs/threat-model.md).
- Every new source file carries an SPDX licence identifier matching its directory.

## Conduct

Be respectful and constructive. Harassment of any kind is not tolerated, and the maintainer may remove
comments, pull requests or participants that do not meet that standard.
