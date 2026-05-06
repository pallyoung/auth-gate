# Auth Gate Architecture Refactor Checklist

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stabilize the backend/frontend contract, remove unsafe model exposure, unify request/session handling, and prepare the server for later control-plane and data-plane separation without over-engineering the current deployment model.

**Architecture:** Keep the current monorepo and single-process deployment, but refactor boundaries inside that shape first. The main direction is `store entity -> service/domain model -> API DTO`, plus a single frontend API/session layer that all pages use. Only after those boundaries are stable should the proxy path and admin path be separated further.

**Tech Stack:** Go, Gin, SQLite, JWT, React, TypeScript, Vite

---

## Refactor Order

### Phase 0: Stop the obvious problems first

**Why first:** These are correctness and security issues, not style issues.

- [ ] Stop returning auth secrets in API responses.
  Files:
  `packages/server/internal/store/models.go`
  `packages/server/internal/api/handlers.go`
  `packages/web/src/lib/api.ts`
  Outcome:
  `GET /api/auth-rules` and `GET /api/auth-rules/:id` should not expose `config.secret` or `config.password`.

- [ ] Remove direct `fetch()` calls from UI code that bypass auth/error handling.
  Files:
  `packages/web/src/lib/api.ts`
  `packages/web/src/App.tsx`
  `packages/web/src/pages/LoginPage.tsx`
  `packages/web/src/pages/SettingsPage.tsx`
  Outcome:
  All frontend HTTP calls go through one request helper.

- [ ] Fix the protected config reload flow.
  Files:
  `packages/server/internal/api/handlers.go`
  `packages/web/src/pages/SettingsPage.tsx`
  Outcome:
  Reload uses an authenticated non-GET mutation endpoint and reports failure correctly.

- [ ] Remove stale security wording from the UI.
  Files:
  `packages/web/src/pages/SettingsPage.tsx`
  `packages/server/internal/config/config.go`
  `packages/server/cmd/server/main.go`
  Outcome:
  UI and backend documentation both refer to `jwt_secret` and bootstrap admin password, not deprecated `admin_token`.

Verification:
  Login, list auth rules, reload config, and confirm no secret fields are visible in browser responses.

### Phase 1: Freeze the API contract

**Why next:** The current API shape is implementation-driven. Stabilize it before deeper refactors.

- [ ] Introduce explicit request/response DTOs for routes, auth rules, users, and auth session data.
  Files:
  Create `packages/server/internal/api/dto/route.go`
  Create `packages/server/internal/api/dto/auth_rule.go`
  Create `packages/server/internal/api/dto/user.go`
  Create `packages/server/internal/api/dto/auth.go`
  Modify `packages/server/internal/api/auth.go`
  Modify `packages/server/internal/api/handlers.go`

- [ ] Stop binding request bodies directly into `store.Route` and `store.AuthRule`.
  Files:
  `packages/server/internal/api/handlers.go`
  Outcome:
  Handlers bind input DTOs, validate them, then map into store/service inputs.

- [ ] Standardize API naming and envelope conventions.
  Decision:
  Keep resource fields in `snake_case` for now to minimize churn, but normalize permission fields and error payloads.
  Outcome:
  One error shape such as `{ "error": { "code": "...", "message": "..." } }` or one simpler project-wide alternative, used everywhere.

- [ ] Make mutation semantics consistent.
  Changes:
  `POST /api/auth/login`
  `POST /api/auth/logout`
  `GET /api/auth/me`
  `POST /api/config/reload` or `POST /api/admin/config/reload`
  Outcome:
  No state-changing `GET` endpoints remain.

- [ ] Write a small OpenAPI document for the current admin API.
  Files:
  Create `docs/api/admin-openapi.yaml`
  Outcome:
  Frontend and backend can align on one source of truth.

Verification:
  Backend tests cover DTO validation and response serialization for login, routes, auth rules, users, and config reload.

### Phase 2: Introduce a real backend application layer

**Why now:** The handlers currently own validation, persistence orchestration, and response shaping. That will not scale.

- [ ] Add service-level packages for each admin capability.
  Files:
  Create `packages/server/internal/service/routes/service.go`
  Create `packages/server/internal/service/authrules/service.go`
  Create `packages/server/internal/service/users/service.go`
  Create `packages/server/internal/service/session/service.go`

- [ ] Move business rules out of handlers.
  Rules to move:
  Route validation
  Role normalization
  Auth rule validation
  Existence checks like "route must exist before auth rule creation"
  Reload side effects after writes

- [ ] Keep `store` focused on persistence only.
  Files:
  `packages/server/internal/store/routes.go`
  `packages/server/internal/store/auth_rules.go`
  `packages/server/internal/store/users.go`
  Outcome:
  `store` no longer decides API behavior or user-facing error messages.

- [ ] Centralize error mapping.
  Files:
  Create `packages/server/internal/api/errors.go`
  Outcome:
  Services return typed errors, handlers translate them once.

Verification:
  Unit-test service packages without HTTP, then keep a thinner set of handler tests for status code and serialization behavior.

### Phase 3: Unify frontend state and request architecture

**Why now:** Once the backend contract is stable, the frontend can stop carrying duplicate logic.

- [ ] Choose one frontend data pattern and delete the other.
  Recommendation:
  For this app size, use one lightweight app store for session + resource lists, or use page hooks only. Do not keep both page-local fetches and an unused global store.
  Files:
  `packages/web/src/lib/store.ts`
  `packages/web/src/App.tsx`
  `packages/web/src/pages/RoutesPage.tsx`
  `packages/web/src/pages/AuthRulesPage.tsx`
  `packages/web/src/pages/SettingsPage.tsx`

- [ ] Extract session state from `App.tsx` into a dedicated module.
  Files:
  Create `packages/web/src/lib/session.ts`
  or `packages/web/src/features/auth/useSession.ts`
  Outcome:
  Login, logout, token persistence, current-user refresh, and unauthorized handling live in one place.

- [ ] Split API types by resource instead of one growing file.
  Files:
  Replace `packages/web/src/lib/api.ts` with:
  `packages/web/src/lib/api/client.ts`
  `packages/web/src/lib/api/auth.ts`
  `packages/web/src/lib/api/routes.ts`
  `packages/web/src/lib/api/auth-rules.ts`
  `packages/web/src/lib/api/users.ts`

- [ ] Make UI capability-driven, not hardcoded.
  Outcome:
  Use `me` or login permissions consistently to hide or disable admin-only actions.

Verification:
  Manual smoke test login, logout, route CRUD, auth rule CRUD, and unauthorized expiry behavior from the browser.

### Phase 4: Separate control-plane and proxy concerns inside the server

**Why later:** This is worthwhile, but only after the admin API boundary is clean.

- [ ] Split route registration into admin routes, static UI routes, and proxy routes.
  Files:
  `packages/server/cmd/server/main.go`
  Create `packages/server/internal/http/admin/routes.go`
  Create `packages/server/internal/http/static/routes.go`
  Create `packages/server/internal/http/proxy/routes.go`

- [ ] Define clear ownership for runtime components.
  Target shape:
  `config` loads process config
  `store` persists admin state
  `service` applies business rules
  `router.Manager` holds compiled routing state
  `proxy` only handles forwarding
  `api/http` only handles admin transport

- [ ] Introduce an explicit route-compile step.
  Outcome:
  DB records are converted into immutable runtime route definitions for matching and forwarding, instead of using DB-shaped objects directly.

- [ ] Add basic observability hooks around proxy and admin requests.
  Minimum:
  Request logging
  Route match result
  Proxy upstream error count
  Admin mutation audit log

Verification:
  Existing proxy tests still pass; route reload remains functional after admin writes.

### Phase 5: Clean up dead ends and misleading capabilities

**Why last:** Remove unused paths after the main architecture is stable.

- [ ] Delete duplicate or obsolete handler paths.
  Files:
  `packages/server/internal/api/users.go`
  `packages/server/internal/api/auth.go`
  Compare against `packages/server/internal/api/handlers.go`
  Outcome:
  One canonical handler implementation per endpoint area.

- [ ] Decide which auth-rule features are real.
  Candidates:
  `whitelist`
  `rate_limit`
  `jwks_url`
  `issuer`
  `audience`
  Outcome:
  Either implement them end-to-end or remove them from model, API, docs, and UI.

- [ ] Tighten configuration behavior for production.
  Changes:
  Fail fast when `jwt_secret` is missing outside development mode.
  Keep ephemeral secret generation as explicit dev-only behavior.

- [ ] Remove committed build artifacts and local dependency directories from tracked project structure if they are not intentional.
  Review:
  `packages/web/dist`
  `packages/web/node_modules`
  `packages/server/bin`
  `packages/dist`

Verification:
  Repo shape matches intended source-of-truth layout and no misleading fields remain in docs or UI.

## Priority Summary

### Must do now

- [ ] Stop leaking secrets through auth-rule APIs.
- [ ] Unify frontend request/auth handling.
- [ ] Replace direct store-model binding in handlers with DTOs.
- [ ] Fix config reload endpoint semantics.

### Should do next

- [ ] Introduce service layer and centralized error mapping.
- [ ] Split frontend API/session modules.
- [ ] Write minimal OpenAPI spec.

### Can do after that

- [ ] Internal control-plane/data-plane separation.
- [ ] Observability improvements.
- [ ] Remove or implement dormant auth-rule fields.

## Suggested Delivery Sequence

1. PR 1: Security + request-layer cleanup
2. PR 2: Backend DTOs + consistent admin API contract
3. PR 3: Backend service layer extraction
4. PR 4: Frontend state/API reorganization
5. PR 5: Runtime/proxy boundary cleanup
6. PR 6: Dead code and dormant feature cleanup

## Definition of Done

- [ ] No admin API response leaks auth secrets.
- [ ] Frontend has one HTTP client path and one session source of truth.
- [ ] Backend handlers no longer expose or bind raw store entities.
- [ ] Admin API contract is documented and test-covered.
- [ ] Proxy path and admin path have explicit module boundaries.
- [ ] Deprecated and fake capabilities are either implemented or removed.
