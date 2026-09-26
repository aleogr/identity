# Security policy

This project is an identity platform; its security is its purpose. Thank you for helping keep it and
its users safe.

## Reporting a vulnerability

**Report privately through GitHub:** on this repository, open the **Security** tab and choose
**Report a vulnerability**. The report is visible only to the maintainer.

**Never** report a vulnerability in a public issue, pull request or discussion.

Please include what you can of: the affected component and version or commit, a description of the
issue and its impact, steps to reproduce or a proof of concept, and any suggested fix.

## What to expect

| Step | Target |
|---|---|
| Acknowledgement of your report | Within 3 business days |
| Initial assessment (validity, severity) | Within 7 days |
| Fix, advisory and coordinated disclosure | Within 90 days of the report, sooner for severe issues |

Fixes are developed in a private GitHub security advisory. When the fix is released, the advisory is
published with a CVE identifier, and you are credited unless you prefer otherwise. If a fix cannot be
released within 90 days, we agree a new date with you.

## Supported versions

The project is before `1.0.0`. Only the **latest `0.y.z` release** and `main` receive security fixes.
After `1.0.0`, the supported versions and the long-term support line are listed here.

## Scope

In scope: the code in this repository, its release artefacts and its published container images. The
threats the project defends against are listed in [`docs/threat-model.md`](docs/threat-model.md).

Out of scope: the lab deployment at `*.lab.aleogr.dev` beyond what a report about this code requires
(please do not run automated scanners or load against it), social engineering, and physical attacks.

## Safe harbour

Good-faith research that follows this policy — no privacy violations, no destruction of data, no
degradation of service, and private reporting — will not be pursued legally by the maintainer.
