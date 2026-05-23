# Web I18n Design

## Summary

Add multilingual support to the existing `packages/web` control-plane application using `i18next` and `react-i18next`. The first release supports `en` and `zh-CN`, detects the initial language from the browser, persists a user-selected language in `localStorage`, and applies consistently across the control-plane UI, the main login page, and the route access login page.

This design keeps the current Vite + React + hash-router architecture intact. It does not put language into the URL, does not change API payloads or route behavior, and does not translate backend-provided error messages in this phase.

## Goals

- Support `en` and `zh-CN` across the web UI.
- Detect the initial language using:
  1. persisted local language selection
  2. browser language
  3. fallback to `en`
- Allow manual language switching without a page reload.
- Persist the selected language so it survives browser restarts.
- Cover both authenticated and unauthenticated entry points:
  - control-plane login page
  - route access login page
  - logged-in control-plane pages
- Use a mature third-party i18n library that can scale to more locales and namespaces later.

## Non-Goals

- No locale-specific routing or URL rewriting.
- No backend API changes.
- No translation of backend-returned error strings.
- No custom modal system to replace `window.confirm`.
- No expansion beyond `en` and `zh-CN` in this phase.

## Current State

The web app lives in `packages/web` and is a Vite + React application with hash-based routing managed inside `src/App.tsx`. Most user-facing strings are hardcoded directly inside:

- page components
- layout and navigation
- forms
- shared table and empty-state components
- loading and confirmation messages

There is no existing i18n runtime, no translation resource structure, and one page currently formats dates with a hardcoded `en-US` locale.

## Decision

Use `i18next` with `react-i18next`.

This is preferred over a custom translation layer because the project is expected to expand, and `i18next` provides:

- a stable runtime
- good React integration
- namespace support
- straightforward fallback behavior
- scalable resource organization for future locales

The design keeps the integration thin so the UI consumes `useTranslation(...)` directly rather than building a second abstraction on top of the library.

## Architecture

### 1. I18n Runtime

Add a dedicated i18n initialization module under `packages/web/src/lib/i18n/`.

Responsibilities:

- initialize `i18next`
- register `react-i18next`
- declare supported locales
- declare fallback locale
- provide translation resources
- handle language detection and persistence

The application entrypoint in `src/main.tsx` loads this module before rendering the app so all components can safely use translation hooks during first render.

### 2. Supported Locales

The supported locales are:

- `en`
- `zh-CN`

Fallback locale:

- `en`

Language normalization rules:

- if persisted language is one of the supported locales, use it
- otherwise inspect `navigator.languages` and `navigator.language`
- if the preferred browser language starts with `zh`, map it to `zh-CN`
- otherwise use `en`

This keeps detection predictable while remaining easy to extend later.

### 3. Persistence

Persist the selected locale in `localStorage`.

Behavior:

- first visit: follow browser language
- after a manual language change: persist the explicit choice
- subsequent visits: use the persisted choice even if browser language differs

The locale persistence key should be defined once in the i18n config module rather than duplicated in UI components.

### 4. Resource Organization

Store translations by locale and namespace, not as one global file.

Initial namespace split:

- `common`
- `layout`
- `login`
- `accessLogin`
- `routes`
- `authRules`
- `users`
- `certificates`
- `settings`

This structure matches the current page layout and keeps future expansion localized to the page or shared area being edited.

### 5. UI Integration

The UI gets language switching in three places:

- `Layout` for authenticated control-plane pages
- `LoginPage` for unauthenticated control-plane entry
- `AccessLoginPage` for unauthenticated route access entry

The switcher stays in the current UI style and uses a lightweight two-option control for `EN` and `中文`. A dropdown is unnecessary for two languages and would add visual weight without improving usability.

### 6. Display-Layer Translation Only

Business values remain in English and unchanged in state, API payloads, and routing logic.

Examples:

- user roles remain `member`, `viewer`, `editor`, `admin`
- certificate statuses remain `active`, `pending`, `renewing`, `failed`
- auth rule types remain `none`, `apikey`, `bearer`, `basic`, `gateway`

Only their rendered labels are translated.

This avoids introducing translated strings into logic branches, form submissions, or backend contracts.

## Translation Scope

This phase translates all frontend-controlled strings in the web UI, including:

- navigation labels and descriptions
- page headers
- card titles and descriptions
- metric labels and hints
- table headers
- empty states
- form labels
- form hints
- placeholders
- button labels
- modal titles
- loading text
- `confirm(...)` messages
- frontend-generated generic error messages

This phase does not translate backend-returned error message bodies. Those continue to display exactly as returned by the API.

## Page and Component Coverage

### Entry Pages

Translate all visible text in:

- `src/pages/LoginPage.tsx`
- `src/pages/AccessLoginPage.tsx`

This includes headings, support copy, form labels, placeholders, button text, route context, and client-generated network error text.

### Authenticated Pages

Translate all visible text in:

- `src/pages/RoutesPage.tsx`
- `src/pages/AuthRulesPage.tsx`
- `src/pages/UsersPage.tsx`
- `src/pages/CertificatesPage.tsx`
- `src/pages/SettingsPage.tsx`

### Shared Components

Translate all visible text in:

- `src/components/Layout.tsx`
- `src/components/PageHeader.tsx`
- `src/components/DataTable.tsx`
- `src/components/RouteForm.tsx`
- `src/components/UserForm.tsx`
- `src/components/AuthRuleForm.tsx`
- `src/components/CertificateForm.tsx`
- shared empty-state or common UI where text is owned by the component

If a shared component only renders text passed in from callers, the caller remains responsible for translation.

## Data and Formatting Rules

### Date Formatting

`CertificatesPage` currently uses a fixed `en-US` formatter. Replace this with locale-aware formatting driven by the active UI language.

Rules:

- `en` uses English formatting
- `zh-CN` uses Simplified Chinese formatting
- relative text like `days left` is translated through resource keys

Use the active locale as the source of truth for `Intl.DateTimeFormat`.

### Dynamic Strings

Interpolated UI messages are allowed where needed, especially for:

- confirmation messages
- route names
- usernames
- certificate domains
- counts in short summary text

Interpolation stays inside translation resources so templates remain localizable. The translated UI should not assemble sentence fragments in component code.

## Error Handling

Frontend-generated errors should use translated strings where the frontend owns the wording.

Examples:

- generic network error fallback
- reload failure fallback in settings
- generic loading labels

Backend-generated error strings should remain untouched in this phase.

This preserves backend diagnostics and avoids creating a partial or fragile error-code mapping system before the API contract is standardized for localization.

## Testing Strategy

### Unit Tests

Add tests for i18n initialization behavior:

- persisted locale wins over browser locale
- supported browser `zh` preference resolves to `zh-CN`
- unsupported persisted values fall back to browser or `en`
- manual language change updates persistence

### Component Tests

Add representative UI tests for:

- `LoginPage` rendering translated title and primary action in both locales
- `Layout` updating navigation labels after language switch

### Page-Level Regression Test

Add at least one page-focused regression test around `CertificatesPage` to verify:

- translated status display
- locale-aware date formatting
- translated relative expiry text

This page is the best target because it includes both static labels and locale-dependent formatting.

### Verification Commands

Implementation should be validated with:

- `pnpm --filter auth-gate-web test`
- `pnpm --filter auth-gate-web build`

Manual smoke testing should cover:

- control-plane login page language switching
- route access login page language switching
- authenticated layout language switching
- a list page with table headers and actions
- a form modal
- a `confirm(...)` flow
- settings reload UI

## Rollout Notes

This design keeps the initial implementation intentionally narrow:

- only two locales
- no URL localization
- no backend message translation
- no API changes

That keeps the first pass low-risk while establishing a scalable structure for future additions such as:

- more locales
- lazy-loaded namespaces
- localized backend error-code mapping
- locale-aware number and date helpers beyond current needs

## Risks and Mitigations

### Risk: Hardcoded strings remain scattered and some get missed

Mitigation:

- organize resources by page namespace
- translate shared components explicitly
- add representative tests for entry pages, layout, and a page with dates/statuses

### Risk: Translated labels leak into business logic

Mitigation:

- keep enum values and API payloads unchanged
- translate only at render time

### Risk: Locale initialization flashes the wrong language on first paint

Mitigation:

- initialize i18n before app render in `main.tsx`
- resolve persisted and browser locale during initialization, not after mount

### Risk: Future growth turns one resource file into a dumping ground

Mitigation:

- start with namespaces from day one
- keep shared text in `common` and page-owned text in page namespaces

## Acceptance Criteria

The design is complete when the implemented feature satisfies all of the following:

- the web UI supports `en` and `zh-CN`
- initial language follows persisted value first, then browser language, then `en`
- manual language switching works without reload
- the chosen language persists across browser restarts
- language switching is available on login, route access login, and authenticated control-plane screens
- all frontend-owned visible text in the current web UI is translated
- backend-returned error messages still display as returned
- certificate dates and relative expiry text follow the active locale
- tests cover runtime selection, representative UI rendering, and certificate formatting behavior
