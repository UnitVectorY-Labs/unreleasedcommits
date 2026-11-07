# UVY Design v0.1.0 adoption

Source release: https://github.com/UnitVectorY-Labs/uvy-design/releases/tag/v0.1.0

Pinned source commit: `396a65021c2bcab8dbcb3d1bb4cb498c807a4396`.

Canonical tokens, optional element styling and theme controller are copied verbatim into `templates/uvy-design/`, embedded in the binary and emitted locally in every generated output. The existing `templates/style.css` is the app-specific adapter. `templates/shared.html` contains inline official Tabler paths copied from the release; the MIT icon notice is retained. No HTMX or external runtime dependency is introduced.

## Preserved behavior and scope

The existing summary cards, sortable table, repository/detail links, merge disclosures and three-metric compact mobile geometry remain. Header identity is the standalone project name; footer links to this project and its license. System fonts and standard density map to the approved variables, with light/dark available through the top-right binary cookie toggle.

Green–yellow–red backgrounds, data normalization and metric values are unchanged. Only foreground selection is improved using luminance so text remains readable throughout the original ramp. The 0/0.5/1 anchor colors and all sampled foreground contrast pairs have tests.

## Verification and screenshots

Generate and test with `go test ./...`, `go vet ./...`, `go build .` and `go run . -generate`. Review both themes, OS defaults/cookie reload, desktop/mobile sorting, 320px fit, repository links, existing merge expansion, unchanged data/spectrum and zero external runtime requests. The PR embeds the complete before/after desktop/mobile light/dark major-page evidence matrix. Baseline has no dark mode; both OS preferences faithfully retain its original appearance. A separate documented no-commits fixture covers the existing empty-state branch because all production repositories currently have commits. No-release repositories are intentionally skipped by the existing crawler.

No screenshots are committed to this application. Evidence is hosted in the design repository's dedicated evidence branch and linked immutably from the PR.

## Update procedure

Choose a reviewed future release, verify archive checksums and file hashes, then replace only canonical copied files. Review source diffs and update this record; keep app-specific adapter changes local. Rebuild and repeat workflow/evidence checks before updating consumers. Never fetch design assets at browser runtime or from a mutable branch.

## Copied-file SHA-256

- `LICENSE`: `00320f7ebea7d48c92ddabd067541e3a8d3c033deafb50ada6bf7a3a56f640dc`
- `TABLER-ICONS-LICENSE`: `b740a1d46122672da62833e97f7e7c8a13fa85cbc7445b584b297cc00dde93db`
- `uvy-elements.css`: `b2f92292dd9317bb770be1a4c745c9a00b81b6cef39cbcac54d2fe192135b092`
- `uvy-theme.js`: `b105c0ddec8c19b6a380a33ffa5f9f558ca49baa477921a1dee9596d77bc61ed`
- `uvy-tokens.css`: `0e97baaece886e0015c4238eaf7dbed9ea68f330c0f50729311c43b4a5a98533`
