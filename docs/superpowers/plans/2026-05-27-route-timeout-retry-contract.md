# Route Timeout Retry Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore end-to-end control-plane support for route-level `timeout_ms` and `retry_attempts` so operators can persist, edit, and review the runtime settings that the proxy already consumes.

**Architecture:** Keep the current route model and runtime behavior intact. Extend the SQLite schema, store queries, DTOs, admin handlers, and route service inputs so the existing control-plane contract carries both fields end-to-end. On the web side, extend the typed API contract, add compact numeric controls to `RouteForm`, and surface a small summary in the routes table when either value is configured.

**Tech Stack:** Go, Gin, SQLite, React, TypeScript, Vitest

---

## File Map

### Create

- None

### Modify

- `packages/server/internal/store/sqlite.go`
- `packages/server/internal/store/routes.go`
- `packages/server/internal/api/dto/route.go`
- `packages/server/internal/service/routes/service.go`
- `packages/server/internal/http/admin/routes.go`
- `packages/server/internal/store/sqlite_test.go`
- `packages/server/internal/service/routes/service_test.go`
- `packages/server/internal/http/admin/routes_test.go`
- `packages/web/src/lib/api/types.ts`
- `packages/web/src/components/RouteForm.tsx`
- `packages/web/src/pages/RoutesPage.tsx`
- `packages/web/src/lib/i18n/resources/routes.ts`
- `packages/web/src/components/RouteForm.test.tsx`
- `packages/web/src/pages/RoutesPage.test.tsx`

## Task 1: Prove the Backend Contract Gap

**Files:**
- Modify: `packages/server/internal/store/sqlite_test.go`
- Modify: `packages/server/internal/service/routes/service_test.go`
- Modify: `packages/server/internal/http/admin/routes_test.go`

- [x] **Step 1: Add a failing store regression for runtime policy persistence**

Add a test that creates a route with `TimeoutMs: 4500` and `RetryAttempts: 3`, reads it back through `GetRoute`, and asserts both values survive persistence.

- [x] **Step 2: Add a failing service regression for omitted-field preservation**

Extend the route update preservation test so an update payload that only changes the route name still preserves existing `TimeoutMs` and `RetryAttempts`.

- [x] **Step 3: Add a failing admin API regression**

Add or extend an admin route test so `POST /routes` accepts `timeout_ms` and `retry_attempts`, returns them in JSON, and persists them in SQLite.

- [x] **Step 4: Run the targeted backend tests and confirm RED**

Run:

```bash
cd packages/server && go test ./internal/store -run 'TestRoute_PersistsRuntimePolicyFields' -count=1
cd packages/server && go test ./internal/service/routes -run 'TestServiceUpdateRoute_PreservesOmittedFields' -count=1
cd packages/server && go test ./internal/http/admin -run 'TestRegisterRoutes_CreateRoutePersistsAndReturnsRuntimePolicyFields|TestRegisterRoutes_UpdateRoutePreservesOmittedFields' -count=1
```

Expected:

```text
FAIL because timeout_ms and retry_attempts are not yet stored or exposed by the control plane.
```

## Task 2: Implement the Backend Fix

**Files:**
- Modify: `packages/server/internal/store/sqlite.go`
- Modify: `packages/server/internal/store/routes.go`
- Modify: `packages/server/internal/api/dto/route.go`
- Modify: `packages/server/internal/service/routes/service.go`
- Modify: `packages/server/internal/http/admin/routes.go`

- [x] **Step 1: Extend schema and migrations**

Add `timeout_ms` and `retry_attempts` to the `routes` table schema and append matching `ALTER TABLE` migrations.

- [x] **Step 2: Extend store reads and writes**

Update route `SELECT`, `INSERT`, and `UPDATE` statements so both fields round-trip through `store.Route`.

- [x] **Step 3: Extend DTOs and handler wiring**

Add `timeout_ms` and `retry_attempts` to route response, create request, and update request DTOs, then map them in admin create/update handlers.

- [x] **Step 4: Extend route service inputs**

Add both fields to `CreateInput` and `UpdateInput`, set them on create, and preserve stored values when an update omits them.

- [x] **Step 5: Re-run the targeted backend tests and then the server suite**

Run:

```bash
cd packages/server && go test ./internal/store -run 'TestRoute_PersistsRuntimePolicyFields' -count=1
cd packages/server && go test ./internal/service/routes -run 'TestServiceUpdateRoute_PreservesOmittedFields' -count=1
cd packages/server && go test ./internal/http/admin -run 'TestRegisterRoutes_CreateRoutePersistsAndReturnsRuntimePolicyFields|TestRegisterRoutes_UpdateRoutePreservesOmittedFields' -count=1
cd packages/server && go test ./...
```

Expected:

```text
PASS for the targeted regressions and PASS for the broader server suite.
```

## Task 3: Prove and Fix the Web Contract Gap

**Files:**
- Modify: `packages/web/src/components/RouteForm.test.tsx`
- Modify: `packages/web/src/pages/RoutesPage.test.tsx`
- Modify: `packages/web/src/lib/api/types.ts`
- Modify: `packages/web/src/components/RouteForm.tsx`
- Modify: `packages/web/src/pages/RoutesPage.tsx`
- Modify: `packages/web/src/lib/i18n/resources/routes.ts`

- [x] **Step 1: Add a failing form regression**

Extend the route form tests so a route with `timeout_ms` and `retry_attempts` hydrates both controls, and a submitted form includes both values in the payload.

- [x] **Step 2: Add a failing routes table regression**

Extend the routes page tests so a listed route shows compact timeout and retry summaries when the API returns non-zero values.

- [x] **Step 3: Run the targeted web tests and confirm RED**

Run:

```bash
cd packages/web && npm test -- src/components/RouteForm.test.tsx
cd packages/web && npm test -- src/pages/RoutesPage.test.tsx
```

Expected:

```text
FAIL because route types, form controls, and table summaries do not yet include timeout_ms or retry_attempts.
```

- [x] **Step 4: Extend the typed API contract and form**

Add both fields to `Route` and `RouteInput`, initialize them in `getInitialRouteForm`, render numeric inputs in the main route policy section, and submit normalized numeric values.

- [x] **Step 5: Surface runtime policy summaries in the routes table**

When either value is greater than zero, render a small translated summary alongside the existing backend/TLS metadata.

- [x] **Step 6: Re-run the targeted web tests, then the broader web gates**

Run:

```bash
cd packages/web && npm test -- src/components/RouteForm.test.tsx
cd packages/web && npm test -- src/pages/RoutesPage.test.tsx
cd packages/web && npm test -- --reporter=dot
cd packages/web && npm run build
```

Expected:

```text
PASS for the targeted regressions, PASS for the web test suite, and successful build output.
```

## Task 4: Final Verification

**Files:**
- Modify: none

- [x] **Step 1: Run the project-level gate**

## Execution Notes

- Targeted backend verification completed with:
  - `go test ./internal/store -run 'TestRoute_PersistsRuntimePolicyFields' -count=1`
  - `go test ./internal/service/routes -run 'TestServiceUpdateRoute_PreservesOmittedFields' -count=1`
  - `go test ./internal/http/admin -run 'TestRegisterRoutes_CreateRoutePersistsAndReturnsRuntimePolicyFields|TestRegisterRoutes_UpdateRoutePreservesOmittedFields' -count=1`
- Broader server verification completed with `go test ./...`.
- Targeted web verification completed with:
  - `npm test -- src/components/RouteForm.test.tsx`
  - `npm test -- src/pages/RoutesPage.test.tsx`
- Broader web verification completed with `npm test -- --reporter=dot` and `npm run build`.
- Project-level verification completed with `make test`.

Run:

```bash
make test
```

Expected:

```text
PASS with no new route timeout/retry regressions anywhere in the repo.
```
