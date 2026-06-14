# Hosts Management (SwitchHosts-style)

## Context

The auth-gate server today is a reverse proxy with route / auth / cert / user management, but offers no way to manage `/etc/hosts` on the host. Operators who want local DNS overrides for testing (e.g. point `api.local` at `127.0.0.1`) have to SSH in, edit the file as root, and lose their entries the next time something rewrites `/etc/hosts`. We need a control-plane UI that mirrors the workflow of SwitchHosts: maintain several named **profiles** of `IP + hostnames` entries, switch between them from the browser, and write the active profile back to the real `/etc/hosts` so the system resolver picks it up.

This is real file management, not a mockup. The auth-gate process must run as root to write `/etc/hosts`; the install/operational docs will say so. To stay safe we never overwrite the user's hand-written lines: every write happens inside a delimited marker block (`# BEGIN AUTH-GATE MANAGED BLOCK` / `# END AUTH-GATE MANAGED BLOCK`) and the block is the only region we touch. A pre-flight check refuses to activate if the marker is missing and the file already has any non-blank, non-comment lines outside our control.

Out of scope: DNS-over-HTTPS / DoH, dnsmasq integration, profile import from URLs, and OS-level privilege helper scripts. v1 is a single host's `/etc/hosts` edited by admin users via the existing control plane.

## Architecture

```
HostsPage (web)
 └─ hostsApi (web/lib/api/hosts.ts)
     └─ HTTP /api/host-profiles/* (http/admin/routes.go)
         └─ HostService (service/hosts/service.go)
             ├─ hoststore (store/hosts.go)        ← SQLite tables host_profiles / host_entries
             └─ syshosts.Renderer (syshosts/render.go) ← atomic write to /etc/hosts
```

Layered like existing modules (`routes`, `authrules`, `certificates`):
- `store` is dumb persistence with `*SQLite` methods.
- `service` wraps the store with validation, transactions, and domain errors.
- `http/admin` exposes the service through Gin handlers and a `HostService` interface (mirrors the `CertService` pattern) so handler tests can stub it.
- `syshosts` is a small, dependency-free Go package with one job: render and atomically rewrite the marker block in `/etc/hosts`. The service owns the file path and backup directory; the renderer takes a string of lines and writes it.

## Data model

```sql
CREATE TABLE host_profiles (
  id          TEXT PRIMARY KEY,
  name        TEXT UNIQUE NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_active   INTEGER NOT NULL DEFAULT 0,
  created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE host_entries (
  id          TEXT PRIMARY KEY,
  profile_id  TEXT NOT NULL,
  position    INTEGER NOT NULL,
  ip          TEXT NOT NULL,
  hostnames   TEXT NOT NULL,                -- single line, space-separated
  comment     TEXT NOT NULL DEFAULT '',
  enabled     INTEGER NOT NULL DEFAULT 1,
  created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (profile_id) REFERENCES host_profiles(id) ON DELETE CASCADE
);

CREATE INDEX idx_host_entries_profile_position ON host_entries(profile_id, position);
```

`is_active` mutex is enforced by transaction (`UPDATE host_profiles SET is_active=0` then `UPDATE … SET is_active=1 WHERE id=?`), not by a partial unique index, because SQLite does not support `WHERE` clauses on unique indexes. The application is the only writer.

Store layer (`store/hosts.go`) methods mirror `routes.go`:
`ListHostProfiles`, `GetHostProfile`, `CreateHostProfile`, `UpdateHostProfile`, `DeleteHostProfile`, `SetActiveHostProfile(tx, id)`, `ListHostEntries(profileID)`, `CreateHostEntry`, `UpdateHostEntry`, `ReorderHostEntries`, `DeleteHostEntry`. The `SetActiveHostProfile` accepts a `*sql.Tx` so the service can wrap it in a transaction with the file write.

## Service

`internal/service/hosts/service.go`, modeled on `internal/service/routes/service.go`.

```go
const (
    ErrCodeProfileNotFound     = "host_profile_not_found"
    ErrCodeEntryNotFound        = "host_entry_not_found"
    ErrCodeDuplicateProfileName = "duplicate_host_profile_name"
    ErrCodeInvalidProfileName   = "invalid_host_profile_name"
    ErrCodeInvalidIP            = "invalid_host_ip"
    ErrCodeInvalidHostname      = "invalid_host_hostname"
    ErrCodeDuplicateHostname    = "duplicate_host_hostname"
    ErrCodeMarkerMissing        = "host_marker_missing"
    ErrCodePermissionDenied     = "host_permission_denied"
    ErrCodeStoreFailure         = "host_store_failure"
    ErrCodeRenderFailure        = "host_render_failure"
)

type ProfileInput struct { Name, Description string }
type EntryInput struct { IP, Comment string; Hostnames []string; Enabled bool }

type Service struct {
    db       *store.SQLite
    renderer *syshosts.Renderer
}

func (s *Service) ListProfiles() ([]store.HostProfile, error)
func (s *Service) GetProfile(id string) (*store.HostProfile, error)
func (s *Service) CreateProfile(in ProfileInput) (*store.HostProfile, error)
func (s *Service) UpdateProfile(id string, in ProfileInput) (*store.HostProfile, error)
func (s *Service) DeleteProfile(id string) error
func (s *Service) ActivateProfile(id string) (*store.HostProfile, error)

func (s *Service) ListEntries(profileID string) ([]store.HostEntry, error)
func (s *Service) CreateEntry(profileID string, in EntryInput) (*store.HostEntry, error)
func (s *Service) UpdateEntry(profileID, entryID string, in EntryInput) (*store.HostEntry, error)
func (s *Service) ReorderEntries(profileID string, orderedIDs []string) error
func (s *Service) DeleteEntry(profileID, entryID string) error
```

Validation rules (`internal/service/hosts/validate.go`):
- Profile name: trimmed, length 1-32, `^[A-Za-z0-9 _.\-]+$`.
- IP: `net.ParseIP` must be non-nil.
- Hostname (each, trimmed, non-empty): `^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`.
- Comment: optional, ≤ 200 chars.
- Duplicate hostname check: load all current entries for the profile, then ensure no surviving hostname equals the new/updated set. This is O(n²) per call; n is small (typical profile < 50 entries) so the simplicity is fine.

`ActivateProfile` flow:
1. `BEGIN` transaction.
2. Verify the target profile exists.
3. `UPDATE host_profiles SET is_active=0`.
4. `UPDATE host_profiles SET is_active=1, updated_at=now WHERE id=?`.
5. Read the profile's enabled entries, render to text.
6. Call `renderer.Apply(text)`. If it returns an error (e.g. marker missing, file unwritable, atomic rename failed) → `ROLLBACK` and return `ErrCodeRenderFailure` or `ErrCodeMarkerMissing` as appropriate.
7. `COMMIT`.

The renderer writes the file *before* commit, so on success the file matches the committed DB state; on failure, the file is untouched (because we use atomic rename and the renderer rejects up front) and the DB is rolled back.

## `/etc/hosts` renderer

`internal/syshosts/render.go`:

```go
const (
    BeginMarker = "# BEGIN AUTH-GATE MANAGED BLOCK"
    EndMarker   = "# END AUTH-GATE MANAGED BLOCK"
    DefaultHostsPath = "/etc/hosts"
)

type Renderer struct {
    HostsPath string
    BackupDir string
    KeepBackups int
    Now       func() time.Time
}

func NewRenderer(dataDir string) *Renderer
func (r *Renderer) Apply(content string) error
```

`Apply` steps:
1. `os.ReadFile(HostsPath)`.
2. Locate `BeginMarker` and `EndMarker` via `bytes.Index`:
   - **Neither present** → file is "fresh". If `bytes.TrimSpace` of the file is empty, write `[BeginMarker, EndMarker]` plus `content`. If it is non-empty, return `ErrMarkerMissing` (caller maps to 409).
   - **Both present** → split into `prefix` (everything before the line containing `BeginMarker`), drop everything between markers (inclusive), `suffix` (everything after the line containing `EndMarker`).
   - **Only one present** → also `ErrMarkerMissing`; the file is in an inconsistent state we will not silently repair.
3. Render new file: `prefix + "\n" + BeginMarker + "\n" + content + EndMarker + "\n" + suffix`. Trim trailing whitespace, ensure a single trailing newline.
4. Back up the current file: copy to `data/hosts/backup/hosts-YYYYMMDD-HHMMSS.bak`. Prune to the most recent `KeepBackups` (default 20) by name sort.
5. Write atomically:
   - `f, _ := os.OpenFile(HostsPath+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)`
   - `f.Write(data)`
   - `f.Sync()` — flush bytes + metadata to disk.
   - `f.Close()`.
   - `os.Rename(HostsPath+".tmp", HostsPath)`.
6. On any error after the backup was taken, attempt to restore the backup; return wrapped error. The `.tmp` file is left in place on failure so an operator can inspect it.

Renderer is unaware of the auth-gate service config. The service constructs it with `dataDir` (so backup dir is `dataDir/hosts/backup`) and the `HostsPath` defaults to `/etc/hosts` (overridable for tests).

## HTTP routes

`internal/http/admin/routes.go` defines a `HostService` interface and registers routes:

```go
type HostService interface {
    ListProfiles() ([]store.HostProfile, error)
    GetProfile(id string) (*store.HostProfile, error)
    CreateProfile(in hostservice.ProfileInput) (*store.HostProfile, error)
    UpdateProfile(id string, in hostservice.ProfileInput) (*store.HostProfile, error)
    DeleteProfile(id string) error
    ActivateProfile(id string) (*store.HostProfile, error)
    ListEntries(profileID string) ([]store.HostEntry, error)
    CreateEntry(profileID string, in hostservice.EntryInput) (*store.HostEntry, error)
    UpdateEntry(profileID, entryID string, in hostservice.EntryInput) (*store.HostEntry, error)
    ReorderEntries(profileID string, orderedIDs []string) error
    DeleteEntry(profileID, entryID string) error
}

func RegisterRoutes(group *gin.RouterGroup, routerMgr *router.Manager, db *store.SQLite,
    certSvc CertService, hostSvc HostService)
```

Endpoint map:

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/host-profiles` | any logged in | includes `active_id` query helper |
| GET | `/host-profiles/:id` | any logged in | |
| GET | `/host-profiles/:id/entries` | any logged in | |
| POST | `/host-profiles` | admin | |
| PUT | `/host-profiles/:id` | admin | name + description only |
| DELETE | `/host-profiles/:id` | admin | cascades entries |
| POST | `/host-profiles/:id/activate` | admin | writes /etc/hosts |
| POST | `/host-profiles/:id/entries` | admin | |
| PUT | `/host-profiles/:id/entries/:eid` | admin | |
| PUT | `/host-profiles/:id/entries/reorder` | admin | v1 accepts `[]string` body |
| DELETE | `/host-profiles/:id/entries/:eid` | admin | |

`writeServiceError` chain gains a `hostServiceError` step that maps the typed errors to HTTP status codes (404/400/409/403/500).

The `GET /host-profiles` response shape includes a convenience `active_id` field so the UI doesn't need a second call:

```json
{ "profiles": [ ... ], "active_id": "uuid-or-empty" }
```

But the standard pattern in the codebase is `ListResponse` returning a flat array. To stay consistent, the active id is returned as a separate top-level field of a wrapper object `HostProfileListResponse`, and `GetProfile` is unchanged. (See "Decisions" below for the small deviation.)

DTO types in `internal/api/dto/host.go`:

```go
type HostProfile struct { ID, Name, Description string; IsActive bool; CreatedAt, UpdatedAt time.Time }
type HostEntry   struct { ID, ProfileID, IP, Hostnames, Comment string; Position int; Enabled bool; CreatedAt, UpdatedAt time.Time }
type HostProfileListResponse struct { Profiles []HostProfile `json:"profiles"`; ActiveID string `json:"active_id"` }
type HostProfileRequest      struct { Name, Description string }
type HostEntryRequest        struct { IP, Comment string; Hostnames []string; Enabled bool }
type HostEntryReorderRequest struct { EntryIDs []string `json:"entry_ids"` }
```

`RegisterRoutes` signature change forces `cmd/server/main.go` to construct the new service and pass it in. `buildEngine` already takes the `CertService` interface — same pattern for `HostService`.

## Permissions

`store.GetPermissions` gains `CanManageHosts bool`. Only admin role has it true. `Permissions` JSON gains `can_manage_hosts: bool` (transferred through `dto.CurrentUserResponse`).

The frontend's `lib/api/types.ts` `Permissions` interface gets `can_manage_hosts: boolean`. `Layout.tsx` adds a nav item `{ path: '/hosts', icon: Network, label: 'Hosts', description: '...', visible: user?.permissions?.can_manage_hosts === true }`. Editor and viewer see no entry.

Even if a non-admin manually visits `/#/hosts`, the page renders entries as read-only and the API returns 403, so the UI degrades gracefully.

## Frontend

`App.tsx` gains a switch case:

```ts
case '/hosts': return <HostsPage />
```

and `knownControlPlanePaths` adds `/hosts`.

New files:

- `packages/web/src/lib/api/hosts.ts` — `hostsApi.{ list, get, create, update, delete, activate, listEntries, createEntry, updateEntry, deleteEntry, reorderEntries }`. Returns the typed shapes; `list()` returns `{ profiles, active_id }`.
- `packages/web/src/lib/api/types.ts` — adds `HostProfile`, `HostEntry`, `HostProfileInput`, `HostEntryInput`, plus an update to `Permissions` (`can_manage_hosts: boolean`).
- `packages/web/src/lib/api/index.ts` — re-exports `hostsApi`.
- `packages/web/src/pages/HostsPage.tsx` — the page itself.
- `packages/web/src/pages/HostsPage.test.tsx` — tests.
- `packages/web/src/components/Hosts/ProfileSwitcher/index.tsx` — segmented buttons, marks the active one.
- `packages/web/src/components/Hosts/HostsTable/index.tsx` — entries table.
- `packages/web/src/components/Hosts/HostEntryForm/index.tsx` — add/edit entry modal form.
- `packages/web/src/components/Hosts/HostProfileForm/index.tsx` — add/edit profile modal form.
- `packages/web/src/lib/i18n/resources/hosts.ts` — full en + zh-CN resources.
- `packages/web/src/lib/i18n/resources/index.ts` — register `hosts` namespace for both locales.

### Page layout

```
[PageHeader: eyebrow "Local DNS", title "System Hosts", description, badge "Hosts", action [Add Profile] (admin only)]

[Card: profile switcher]
  segmented buttons: <Dev> <Prod> <Staging>
  hint line: "Active profile: Dev"

[Card: entries]
  table | # | IP | Hostnames | Comment | Enabled | Actions |
  ...
  [+ Add Entry]

[Footer: [Activate this profile] [Last applied ...]]
```

`markerMissing` error from the most recent activate response is rendered as an `Alert` between the table and the footer.

### Error → translation mapping

The page's `getErrorState` switch covers the new error codes from the service: `host_profile_not_found`, `host_entry_not_found`, `duplicate_host_profile_name`, `invalid_host_profile_name`, `invalid_host_ip`, `invalid_host_hostname`, `duplicate_host_hostname`, `host_marker_missing` (renders the dedicated banner), `host_render_failure`, `host_store_failure`. The existing `unauthorized` / `insufficient_permissions` / `network` keys are reused.

## Testing

Backend:

- `internal/store/hosts_test.go` — `newTestSQLite` helper (already exists); cover `ListHostProfiles` empty/non-empty, `CreateHostProfile` + dup-name error, `UpdateHostProfile`, `DeleteHostProfile` cascade, `SetActiveHostProfile` mutex behavior inside a transaction, `ListHostEntries` ordering.
- `internal/service/hosts/service_test.go` — happy paths for all CRUD, Activate success (renderer stubbed), Activate rollback when renderer fails, Activate returns `ErrCodeMarkerMissing` when renderer says so, all validation codes, duplicate hostname in same profile, enabled=0 entry skipped from rendered content.
- `internal/service/hosts/validate_test.go` — table tests for IP/hostname/profile-name regex.
- `internal/syshosts/render_test.go` — fresh file (empty + non-empty), existing marker split, only-one-marker error, atomic write, backup creation + pruning, content with multi-line entry round-trip. Use a temp `HostsPath` in `t.TempDir()`.
- `internal/http/admin/hosts_routes_test.go` — extend the existing stub pattern (or add a `stubHostService`) to cover: 201 create, 400 invalid IP, 400 duplicate hostname, 404 unknown profile, 403 non-admin path, 409 marker missing on activate, 200 activate happy path.

Frontend:

- `HostsPage.test.tsx` — i18n zh-CN, profile switcher switches visible entries, Add Entry modal validates IP + hostnames, Activate calls API and updates banner, marker-missing banner rendered after rejected activate, viewer/editor see no nav (mock `getSessionUser` to drop `can_manage_hosts`).
- `HostEntryForm.test.tsx` — IP validation, hostname space-split, comment optional, submit re-enables on error.
- `ProfileSwitcher.test.tsx` — rendering, click to switch, active marker visible.
- `HostProfileForm.test.tsx` — name validation, dup-name error from API surfaces in form, submit re-enables.

## Verification (end-to-end)

```bash
# 1. Build
cd packages/server && go mod tidy && go build ./...
cd ../web && npm install && npm run build

# 2. Tests
cd ../server && go test ./internal/store/... ./internal/service/hosts/... ./internal/syshosts/... ./internal/http/admin/...
cd ../web && npm test

# 3. Run as root (required for /etc/hosts writes)
sudo -E make run   # or: sudo -E ./bin/server

# 4. Open http://localhost:8080/_authgate, log in as admin.
#    - Sidebar shows "Hosts" entry.
#    - Click it → /hosts page renders empty state.

# 5. Add profile "Dev", add entry 127.0.0.1 api.local, click Activate.
#    - Server returns 409 host_marker_missing.
#    - Banner shows "marker missing" guidance.

# 6. Manually append to /etc/hosts:
#    sudo sh -c 'printf "\n# BEGIN AUTH-GATE MANAGED BLOCK\n# END AUTH-GATE MANAGED BLOCK\n" >> /etc/hosts'

# 7. Click Activate again.
#    - 200 OK, banner shows "Last applied <now>".
#    - `cat /etc/hosts` shows the entry inside the marker block; lines outside are untouched.

# 8. Add a second profile "Prod" with ::1 api.local; activate it.
#    - Dev entries vanish from /etc/hosts, Prod entry appears.
#    - Backup file appears at data/hosts/backup/hosts-*.bak.

# 9. Try adding an entry with IP "not-an-ip" → 400 invalid_host_ip.
# 10. Try adding a duplicate hostname in same profile → 400 duplicate_hostname.
# 11. Log in as a viewer → sidebar has no Hosts entry. /#/hosts is blank.
```

## Critical files

**New (backend):**
- `packages/server/internal/store/hosts.go`
- `packages/server/internal/store/hosts_test.go`
- `packages/server/internal/service/hosts/service.go`
- `packages/server/internal/service/hosts/service_test.go`
- `packages/server/internal/service/hosts/validate.go`
- `packages/server/internal/service/hosts/validate_test.go`
- `packages/server/internal/syshosts/render.go`
- `packages/server/internal/syshosts/render_test.go`
- `packages/server/internal/api/dto/host.go`
- `packages/server/internal/http/admin/hosts_routes_test.go`

**Modified (backend):**
- `packages/server/internal/store/sqlite.go` — append two CREATE TABLE blocks + index. Add the migration `ALTER TABLE` no-ops defensively (table is new so they'll fail to apply on a fresh DB; wrapped in `db.Exec` so the error is swallowed like the others).
- `packages/server/internal/store/users.go` — `Permissions` struct gains `CanManageHosts bool`; `GetPermissions` returns it true only for `RoleAdmin`.
- `packages/server/internal/store/models.go` — add `HostProfile` and `HostEntry` structs.
- `packages/server/internal/api/dto/certificate.go` (or new `user.go` patch) — `CurrentUserResponse` adds `can_manage_hosts`.
- `packages/server/internal/http/admin/routes.go` — new `HostService` interface; new handlers; new `hostServiceError` and `writeServiceError` chain step; `RegisterRoutes` signature.
- `packages/server/cmd/server/main.go` — construct `hostservice.NewService`, build `syshosts.Renderer`, pass into `RegisterRoutes`.

**New (frontend):**
- `packages/web/src/lib/api/hosts.ts`
- `packages/web/src/pages/HostsPage.tsx`
- `packages/web/src/pages/HostsPage.test.tsx`
- `packages/web/src/components/Hosts/ProfileSwitcher/index.tsx`
- `packages/web/src/components/Hosts/ProfileSwitcher/index.test.tsx`
- `packages/web/src/components/Hosts/HostsTable/index.tsx`
- `packages/web/src/components/Hosts/HostsTable/index.test.tsx`
- `packages/web/src/components/Hosts/HostEntryForm/index.tsx`
- `packages/web/src/components/Hosts/HostEntryForm/index.test.tsx`
- `packages/web/src/components/Hosts/HostProfileForm/index.tsx`
- `packages/web/src/components/Hosts/HostProfileForm/index.test.tsx`
- `packages/web/src/lib/i18n/resources/hosts.ts`

**Modified (frontend):**
- `packages/web/src/lib/api/types.ts` — add `HostProfile`, `HostEntry`, `HostProfileInput`, `HostEntryInput`; add `can_manage_hosts` to `Permissions`.
- `packages/web/src/lib/api/index.ts` — re-export `hostsApi`.
- `packages/web/src/lib/i18n/resources/index.ts` — register `hosts` namespace.
- `packages/web/src/lib/i18n/resources/layout.ts` — add `sections.hosts.{label,description}` strings.
- `packages/web/src/components/Layout.tsx` — add nav item with `visible: user.permissions.can_manage_hosts === true`.
- `packages/web/src/App.tsx` — add `/hosts` to `knownControlPlanePaths`; add switch case.

## Reused existing utilities

- `store.NewSQLite` and `newTestSQLite` for tests.
- `routesservice.Error` and `Code()` pattern (`internal/service/routes/service.go:32-60`) — copy verbatim shape.
- `routesservice.Service.Create/Update/Delete` patterns for transaction wrapping + reloader calls (`internal/service/routes/service.go:135-243`).
- `http/admin/routes.go:531-541` `writeServiceError` chain.
- `http/admin/routes.go:617-624` `writeError` helper.
- `http/admin/routes.go:543-558` `routeServiceError` mapping as the template for `hostServiceError`.
- `cmd/server/main.go` wiring of services into `RegisterRoutes` and `buildEngine`.
- Frontend `lib/error-state.ts` `LocalizedError` / `resolveLocalizedText` and the existing `getErrorState` switch in `CertificatesPage` (which is the same shape as we'll add for hosts).
- Frontend `lib/api/client.ts` `request` and `listResource`; we will use `request` for the wrapper-object `list` response.
- Frontend `components/ui` `Modal`, `Button`, `Input`, `Card`, `EmptyState`, `Alert`, `Badge`.
- Frontend `DataTable` for the entries table.

## Decisions

- **Server must run as root.** We don't ship a `setcap` helper in v1; the install docs say to run as root or set capabilities. Future helper script is a v2 concern.
- **Renderer rejects /etc/hosts with hand-written entries but no marker.** This is the safest default — we never silently lose work — and the error path is friendly: the UI shows a banner with the snippet the operator can copy/paste. A separate `auth-gate hosts init` CLI is deferred.
- **Atomic write uses rename.** This is the only safe pattern on POSIX. The temp file is in the same directory as `/etc/hosts` so the rename is atomic on the same filesystem.
- **Reorder endpoint is exposed even though v1 UI doesn't use it.** It's a small backend addition, tests cover it, and it leaves the door open for drag-and-drop in v2 without a backend change.
- **No external hosts source (URL/file pull).** v1 covers the local file only. The data model could grow to add it later by giving each profile a `source_type` column.
- **One marker block per file.** We don't write `[profile A]` / `[profile B]` sub-blocks. The active profile is the *only* content inside the marker; switching to another profile rewrites the same block.

## Out of scope (deferred)

- `auth-gate hosts init` CLI to add the marker block automatically.
- `setcap` / sudoers helper to let the server run as non-root.
- Multi-source profiles (URL fetch, remote git).
- IPv6 zone ids, IDN / punycode hostnames.
- Profile import / export of raw text.
- Drag-and-drop reordering UI.
- Diff view of `/etc/hosts` before activating.
- DoH, dnsmasq, systemd-resolved integration.
