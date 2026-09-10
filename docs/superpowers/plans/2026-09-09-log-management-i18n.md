# Log management and localization implementation

User approved this scope in conversation on 2026-09-09. Implement in this checkout; do not deploy or operate on existing user logs.

## Contract

- Keep the Go/JSONL ring buffer and React/i18next architecture and existing visual system.
- Record the incoming request hostname in `host`; historical entries without it remain unknown.
- List, stats and aggregate accept the same host, client_ip, path, username, auth_result, route_id, status_code, start_time and end_time filters. Times use RFC3339, inclusive boundaries, and reject invalid/reversed ranges. Legacy duration queries continue to work.
- Stats add `auth_failures`; graph buckets must cover the selected period (bounded/adaptive resolution), not only the current hour/day. Aggregate also supports host.
- Admin-only `POST /access-logs/purge` accepts `{mode: 'expired'|'filtered'|'all', filters?, preview: boolean, cutoff_time?: string}`. Preview returns `{matched, cutoff_time, retention_days?}`. Execution requires the preview cutoff and returns `{removed, cutoff_time}`. Freeze the upper request-start timestamp at preview to protect requests started after the cutoff. Preview counts are current counts, not immutable row snapshots; late-finishing requests can change the final removed count. Expired mode uses the saved retention policy; zero means disabled. Filtered mode requires a real filter. Validate all filters and scope before any mutation.
- Keep the legacy retention API compatible. Serialize flush and purge; persist even an empty file, propagate errors, and preserve memory on persistence failure.
- Log UI: shared domain/time filters above summaries and both views; default 24 hours; presets 1h/6h/24h/7d/all and custom local date/time with explicit timezone. Reset paging on filters; retain filter scope when drilling down; reject stale results. Show full timestamp, host, no-data/error/loading states and retained-log capacity note.
- Reuse Input/Select/Modal/Button/Alert. Cleanup dialog previews scope/count, makes cancel safe, prevents duplicate submission, and refreshes queries after success. Preserve the existing settings entry.
- Bring all application-owned text, dates and numbers into the active en/zh-CN locale, including standalone gateway pages. Language preference cannot cross browser origins; gateway pages detect their own preference/browser locale.

## Independent work

1. Backend log capture, query, statistics, cleanup and tests. Own server files except standalone page localization templates/helper.
2. Non-log localization: missing route key, Layout, RouteAuth, Dashboard, Hosts accessibility, standalone gateway login/404/502 templates, relevant tests. Do not edit log UI/resources or backend log handlers/store/proxy.go.
3. Root: log API types/client, log page/filters/charts/detail/cleanup dialog and resources/tests; settings integration; design notes and final verification.

## Verification

- Reproduce the empty-file cleanup defect and add persistent restart, write failure, concurrent flush/delete, filter boundary/combination, original-host capture, statistics and administrator permission regression coverage.
- UI regression tests cover date conversion/validation, filtering across views, stale responses, locale switching, cleanup confirmation/count and disabled/permission/error states.
- Run Go tests, frontend tests, TypeScript and production build. Verify changed UI with isolated fixture data in a browser, including narrow viewport and both languages.

## Progress

- [x] Backend implementation and regression tests
- [x] Non-log localization and regression tests
- [x] Log UI, cleanup interaction and regression tests
- [x] Integration review and available verification

## Completion evidence

- Full frontend run: 35 files / 301 tests passed. A subsequently added log language-switch regression and expanded all-namespace reference scan passed in the focused 8-test run.
- `go test ./... -count=1` passed across the server module.
- `npx tsc --noEmit` and `npm run build` passed; Vite reports a main chunk over 500 kB, as before this work.
- Independent review found a debounce/reset race. A new regression first reproduced it, then passed after pending work was cancelled on external filter replacement. Scoped re-review confirmed the fix.
- Live smoke test: 41 assertions against a rebuilt server and disposable data on loopback ports 19480/19490. Verified all filters, aggregate/statistics consistency, admin guards, preview non-mutation, post-cutoff request retention, zero-byte persistence after clearing all, and empty results after restarting. QA server was stopped and temporary credentials removed.
- No connected browser was available; both browser discovery and in-app browser creation failed. Real screenshots, narrow-screen layout and popup checks remain unverified.
- Go race instrumentation was unavailable (CGO disabled, no C compiler). Concurrency regression tests passed without `-race`.
- The optional Python skill audit could not run: no Python runtime/launcher was installed, only the Windows application alias. Project-owned TypeScript reference checks and interaction tests were used; no static premium-audit compliance claim is made.
- No deployment, commit, or operation on existing user logs was performed.
