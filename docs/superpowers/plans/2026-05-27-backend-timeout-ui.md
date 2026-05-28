# Backend Timeout UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose per-backend dial, read, and write timeout controls in route management so operators can configure the runtime overrides that already exist in the backend route model.

**Architecture:** Keep the backend contract unchanged and extend only the web route-management surface. Add typed, localized numeric inputs inside each backend pool entry, normalize those values through the existing `RouteForm` submission flow, and render compact per-backend timeout summaries in the routes table when any override is configured.

**Tech Stack:** React, TypeScript, Vitest, i18next

---

## File Map

### Create

- None

### Modify

- `packages/web/src/components/RouteForm.tsx`
- `packages/web/src/components/RouteForm.test.tsx`
- `packages/web/src/pages/RoutesPage.tsx`
- `packages/web/src/pages/RoutesPage.test.tsx`
- `packages/web/src/lib/i18n/resources/routes.ts`

## Task 1: Prove the UI Contract Gap

**Files:**
- Modify: `packages/web/src/components/RouteForm.test.tsx`
- Modify: `packages/web/src/pages/RoutesPage.test.tsx`

- [x] **Step 1: Add a failing route form regression**

Extend the existing load-balanced route tests so backend timeout controls hydrate from route data and submitted payloads include `dial_timeout_ms`, `read_timeout_ms`, and `write_timeout_ms`.

- [x] **Step 2: Add a failing routes table regression**

Extend the routes page table summary test so a route with backend-specific timeout overrides shows a localized per-backend timeout summary.

- [x] **Step 3: Run the targeted tests and confirm RED**

Run:

```bash
cd packages/web && npm test -- src/components/RouteForm.test.tsx
cd packages/web && npm test -- src/pages/RoutesPage.test.tsx
```

Expected:

```text
FAIL because the route form does not yet expose backend timeout inputs and the routes table does not summarize backend timeout overrides.
```

## Task 2: Implement the UI Support

**Files:**
- Modify: `packages/web/src/components/RouteForm.tsx`
- Modify: `packages/web/src/pages/RoutesPage.tsx`
- Modify: `packages/web/src/lib/i18n/resources/routes.ts`

- [x] **Step 1: Add localized backend timeout controls**

Initialize backend timeout values in `getInitialBackends`, render numeric inputs for dial/read/write timeouts inside each backend card, and preserve those values through the existing update helpers.

- [x] **Step 2: Normalize backend timeout values on submit**

Make sure submitted pooled backends retain the timeout fields after URL trimming and empty-backend filtering.

- [x] **Step 3: Render compact backend timeout summaries in the routes table**

When a backend entry has any non-zero timeout override, render a short localized summary alongside the existing backend pool and TLS metadata.

- [x] **Step 4: Re-run the targeted tests and confirm GREEN**

Run:

```bash
cd packages/web && npm test -- src/components/RouteForm.test.tsx
cd packages/web && npm test -- src/pages/RoutesPage.test.tsx
```

Expected:

```text
PASS for the new backend-timeout regressions.
```

## Task 3: Final Verification

**Files:**
- Modify: none

- [x] **Step 1: Run the broader web gates**

Run:

```bash
cd packages/web && npm test -- --reporter=dot
cd packages/web && npm run build
```

Expected:

```text
PASS for the full web suite and successful production build output.
```

- [x] **Step 2: Run the project-level gate**

## Execution Notes

- Targeted route form verification completed with `npm test -- src/components/RouteForm.test.tsx`.
- Targeted routes page verification completed with `npm test -- src/pages/RoutesPage.test.tsx`.
- Broader web verification completed with `npm test -- --reporter=dot` and `npm run build`.
- Project-level verification completed with `make test`.

Run:

```bash
make test
```

Expected:

```text
PASS with no regressions introduced by the backend-timeout UI changes.
```
