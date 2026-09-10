# Auth Gate interaction ownership

This contract records the existing shared controls and the approved access-log extension. It does not require unrelated pages to be rewritten.

| Capability | Canonical owner | Source of truth | Allowed variants | Verification |
| --- | --- | --- | --- | --- |
| Select/Listbox | `src/components/ui/Select.tsx` | Existing accessible authored Select | Single selection | Component tests; browser popup check when a browser is connected |
| Date | `AccessLogTimeRange.tsx`, `src/lib/access-log-time.ts` | Current local timezone, RFC3339 API instants | Typed `YYYY-MM-DD HH:mm`; relative presets | Impossible dates, range ordering, locale and round-trip tests |
| Form | `src/components/ui/Input.tsx` | Application validation and localized errors | Inline validation with labelled fields | Component tests |
| Scrollbar | `src/styles/base.css` | Existing global CSS | Tables scroll horizontally inside their own wrapper | Existing token styles; narrow browser checks |
| Toast | `src/components/ui/Alert.tsx` | Existing inline feedback | Alert for errors, status for successes | Accessible feedback tests |
| CRUD | `AccessLogCleanup.tsx` with `ui/Modal.tsx` | Admin-only cleanup API | Filtered, expired, all | Preview/confirm, no duplication, cutoff preservation, failure and restart tests |

## Dataset navigation

- List, summary, charts and grouping use identical committed filters and fixed time bounds.
- Use 20 records per page. Reset pages after filter changes and deletion; clamp an out-of-range page after refreshing.
- Preserve filters, grouping, view and pages in the existing hash route's query string.
- Debounce text queries by 300 ms, support IME composition, and reject obsolete responses or pending drafts after reset/navigation.
- Relative presets refresh their time window when Refresh is pressed. Custom ranges retain their exact bounds. Display the applied range separately from an uncommitted custom draft.
- Empty datasets, failed statistics and failed list/group queries remain distinguishable. Statistics failures do not hide successfully loaded log records.

## Cleanup

- Both settings and the log page use the same application dialog. Only administrators get the action; the server independently enforces authorization.
- Opening or changing the scope performs a read-only preview. Display filters, total matching count across all pages, and the cleanup cutoff.
- Disable the log-page cleanup entry while a text filter is being composed or waiting to commit, so its scope always matches the visible filters.
- Cancel receives initial focus. A separate explicit delete button performs the irreversible operation and prevents repeated submission.
- Preserve the preview cutoff during execution. Expired cleanup uses the saved retention policy, never an unsaved settings draft. Changes to retention cannot broaden a previously previewed scope.
- Persistence errors do not report success. Keep the dialog and allow retry. On success, refresh the page and show the actual removed count.

## Verification

Use automated component and live HTTP tests for behavior. These do not replace screenshot, narrow-screen and real popup checks when a browser is available.
