# Licensing

This repository is released under two licences, chosen by what each part is for. The reasoning is in
[`docs/research/licensing.md`](docs/research/licensing.md). This file is a summary, not legal advice.

| Part | Licence | Text |
|---|---|---|
| `spec/`, `core/`, `adapters/`, `conformance/`, `lib/`, `sdk/`, plugin kits | **Apache-2.0** | [`LICENSES/Apache-2.0.txt`](LICENSES/Apache-2.0.txt) |
| Everything else — the server (the root Go module, `cmd/`, `internal/`), `ui/`, the cloud control plane — and any file not covered above | **AGPL-3.0-only**, or a commercial licence | [`LICENSE`](LICENSE), identical to [`LICENSES/AGPL-3.0-only.txt`](LICENSES/AGPL-3.0-only.txt) |
| Documentation in `docs/` | **Apache-2.0** | [`LICENSES/Apache-2.0.txt`](LICENSES/Apache-2.0.txt) |
| Skills copied into `.claude/skills/` from third parties | Their own licence, in each skill's directory | — |

Rules that make the split unambiguous:

- Each Go module carries its own `LICENSE` file with the text that governs it.
- Every source file starts with an SPDX identifier (`// SPDX-License-Identifier: Apache-2.0` or
  `// SPDX-License-Identifier: AGPL-3.0-only`); CI rejects a file without one.
- Code under Apache-2.0 never imports code under AGPL-3.0-only; the module boundaries and a lint rule
  enforce it.

**What this means for you:**

- **Embedding the library or using an SDK** carries only Apache-2.0's obligations, with its patent
  grant. Your application's licence is unaffected.
- **Running the unmodified server**, for yourself or your customers, is free.
- **Modifying the server and offering it over a network** requires publishing those modifications under
  the AGPL-3.0, or a commercial licence.

**No capability is paid, and security fixes are never released to paying customers first.** Commercial
licences, the managed cloud and long-term support are how the project is funded.
