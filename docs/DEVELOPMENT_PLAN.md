# iiiu-nav Development Plan

> Document ID: `IIU-NAV-PLAN-001`
> Version: `1.23`
> Updated: `2026-08-19`
> Product status: `RELEASE_CANDIDATE`
> Development status: `COMPLETE`

This document is the source of truth for product scope, architecture, acceptance criteria, and delivery status. New development sessions must read it completely before changing product code.

## 1. Product Definition

Build a lightweight, single-user, self-hosted navigation website. The public page presents categorized links in a modern responsive interface. One administrator can maintain links, categories, appearance, imports, exports, backups, and restores from the web interface.

### 1.1 Product boundaries

- There is one administrator and no registration, user list, roles, teams, subscriptions, or license system.
- Public categories are visible without authentication.
- Private categories and their links are returned only to an authenticated administrator.
- The application runs as one container with one application process.
- Persistent state lives under `/data` and can be backed up as one unit.
- The first release excludes widgets, monitoring, plugins, nested application categories, analytics, cloud sync, and AI features.

## 2. Fixed Architecture

| Area              | Decision                                                                                                                 |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Backend           | Go, using the standard HTTP stack unless a small dependency removes concrete complexity                                  |
| Frontend          | Preact + TypeScript + Vite                                                                                               |
| Styling           | Plain CSS with design tokens; no full UI framework                                                                       |
| UI primitives     | Small in-repository components for buttons, fields, dialogs, drawers, menus, tooltips, toasts, and empty states          |
| Interface icons   | `lucide-preact`, imported through a curated registry                                                                     |
| Brand icons       | Selected build-time imports from `simple-icons`; no runtime icon CDN                                                     |
| Floating elements | `@floating-ui/dom` for collision-aware menus, pickers, and tooltips                                                      |
| Reordering        | `@atlaskit/pragmatic-drag-and-drop` core package with non-drag alternatives                                              |
| Database          | SQLite through `database/sql` and the pinned pure-Go `modernc.org/sqlite` driver, with WAL mode and versioned migrations |
| Uploaded files    | `/data/uploads/logos`, `/data/uploads/backgrounds`, and `/data/uploads/site`                                             |
| Backups           | `/data/backups`                                                                                                          |
| Deployment        | Multi-stage Docker build; frontend embedded in the Go binary                                                             |
| Runtime           | One container, one process, one mounted `/data` volume                                                                   |
| Authentication    | One administrator; Argon2id password hashes; 256-bit random server-side sessions stored as SHA-256 hashes                |

### 2.1 Persistent layout

```text
/data/
├── nav.db
├── uploads/
│   ├── logos/
│   ├── backgrounds/
│   └── site/
└── backups/
```

The container sets `IIU_NAV_DATA_DIR=/data`. Direct local runs default to `./data`; the environment variable can select another root. Database timestamps are stored as UTC Unix milliseconds and schema tables use SQLite `STRICT` typing.

### 2.2 Initial data model

```text
categories
- id
- name
- slug
- icon_name
- visibility        public | private
- sort_order
- created_at
- updated_at

links
- id
- category_id
- name
- description
- url
- icon_source       auto | url | upload | generated
- icon_value
- sort_order
- created_at
- updated_at

settings
- key
- value_json

admin
- id
- password_hash
- updated_at

sessions
- token_hash
- expires_at
- created_at
```

Schema changes must use forward migrations. Deleting a category with links requires either moving its links to another category or explicitly deleting them.

### 2.3 Go module organization

- `cmd/iiiu-nav` is the composition root. It may load configuration, wire dependencies, and manage process lifecycle, but it contains no business rules or persistence queries.
- `internal/server` owns HTTP routing and transport concerns. Handlers decode and validate requests, call the responsible module, and encode responses; they do not contain SQL or filesystem workflows.
- Product behavior is grouped into cohesive modules such as navigation, authentication, metadata recognition, data transfer, and settings. Persistence and external HTTP access are adapters at explicit seams.
- Dependencies point toward product behavior. Domain modules do not import HTTP handlers or SQLite implementations.
- Interfaces live with the module that consumes them and are introduced only for a real alternate adapter or a useful test seam. Avoid pass-through layers and one-method wrappers that add no behavior.
- Do not create catch-all `utils`, `common`, `helpers`, or `service` packages. A package and file must have one clear reason to change and names must describe product responsibility.
- Prefer deep modules with small interfaces over many shallow packages. Module tests exercise the same interface used by callers; HTTP and SQLite behavior receive focused integration tests.

## 3. Functional Requirements

Each requirement is independently testable. A requirement is complete only when all of its acceptance criteria pass.

### `FR-001` Public navigation layout

Show a left category sidebar and a right link-content area on desktop. Each link is a compact module containing a stable logo area, name, and short description.

Acceptance criteria:

- At widths of `1024px` and above, the category sidebar remains visible and the content uses the remaining width.
- Link modules use a responsive grid and maintain stable dimensions while icons load or fail.
- A module opens its URL in a new tab with `noopener` protection.
- Empty categories and an entirely empty navigation have designed empty states.

### `FR-002` Responsive navigation

Adapt the public and administrative interfaces to desktop, tablet, and mobile layouts.

Acceptance criteria:

- Layouts are verified at widths `360`, `768`, `1024`, and `1440` pixels.
- On mobile, the persistent left sidebar becomes a compact category drawer or horizontal category control.
- Link names, descriptions, controls, and dialogs do not overlap or overflow.
- All administrative actions remain usable without hover and with touch input.

### `FR-003` Categories and privacy

Allow the administrator to create, rename, reorder, delete, and change the visibility of categories.

Acceptance criteria:

- Public API responses omit private categories and their links when no valid administrator session exists.
- Private data is filtered by the backend, not hidden with client-side styling.
- An authenticated administrator sees private categories with a clear lock indicator.
- Reordering persists across reloads.
- Category deletion with existing links requires an explicit move-or-delete choice.

### `FR-004` Link management

Allow the administrator to create, edit, move, reorder, and delete links.

Acceptance criteria:

- Required fields are category, URL, and name after metadata recognition or manual entry.
- URL input accepts only supported `http` and `https` links.
- Name, description, icon, and category can be overridden manually.
- Link movement and ordering persist across reloads.
- Mutating operations return useful validation and conflict errors.

### `FR-005` Metadata and logo recognition

Recognize a link's title, description, and logo while adding it, while allowing complete manual override.

Acceptance criteria:

- The authenticated recognition endpoint reads HTML title, description metadata, declared icons, and then tries `/favicon.ico` as fallback.
- Fetching applies timeouts, response-size limits, redirect limits, and content-type validation.
- Recognition failure leaves the form editable and produces a generated domain-initial icon.
- Icon processing never blocks completion of a bulk bookmark import.
- Cached icons remain available if the source site later becomes unavailable.
- Link edit controls provide a manual metadata and logo refresh action.
- Failed post-import recognition can be retried in bulk without re-importing bookmarks.

### `FR-006` Manual image uploads

Allow an administrator to upload link logos, one site background image, and site identity images.

Acceptance criteria:

- Link and site logos accept PNG, JPEG, WebP, and ICO up to `2 MiB` per file.
- Backgrounds accept PNG, JPEG, and WebP up to `10 MiB` per file.
- Favicons accept PNG and ICO up to `1 MiB` per file.
- File type is verified from content rather than trusting the filename.
- Filenames are generated by the server and cannot escape the configured upload directories.
- Replaced or deleted images are cleaned up only when no database record references them.

### `FR-007` Combined local and web search

Use one search surface for local link discovery and external search engines.

Acceptance criteria:

- Typing matches local link name, description, and URL without a network request.
- Local results are keyboard and pointer accessible.
- The administrator can enable, disable, and reorder configured web search engines.
- Default engines are Google, Baidu, Bing, and DuckDuckGo.
- Activating a web engine opens an encoded query URL in a new tab.
- If a local result is keyboard-selected, Enter opens it; otherwise Enter uses the selected web engine.

### `FR-008` Appearance and themes

Provide light, dark, and system theme modes with an optional administrator-supplied background.

Acceptance criteria:

- Theme selection persists in the browser and system mode follows OS changes.
- Both themes meet readable contrast targets with and without a background image.
- Background overlay strength is configurable.
- Missing or failed background images fall back to a complete neutral surface.

### `FR-009` Administrator authentication

Protect private reads and every mutation with a single administrator login.

Acceptance criteria:

- The first deployment requires an administrator password through a documented secure setup path.
- Initial bootstrap reads `ADMIN_PASSWORD_FILE` only when no administrator exists, hashes the value, and does not persist the plaintext.
- Passwords are stored using Argon2id; plaintext passwords are never logged or persisted.
- Sessions use random tokens stored as hashes and cookies marked `HttpOnly`, `SameSite`, and `Secure` when served over HTTPS.
- Login attempts are rate-limited and mutating requests validate their origin.
- Logout invalidates the current session.
- The administrator can change the password after confirming the current password; success invalidates every session and requires login again.
- A documented `iiiu-nav admin reset-password` container command recovers a forgotten password and invalidates every session.
- There are no registration or user-management endpoints.

### `FR-010` Browser bookmark import

Import Netscape Bookmark HTML files exported by Chrome, Edge, Firefox, and compatible browsers.

Acceptance criteria:

- Import shows a preview containing new links, duplicates, invalid entries, and category mapping before writing.
- Top-level bookmark folders map to categories.
- Nested folders flatten to names such as `Parent / Child`.
- Bookmarks without a folder enter a category named `未分类`.
- Category names are trimmed and Unicode-normalized; a case-insensitive exact match maps to the existing category and is shown in preview.
- Import visibility is selectable and defaults to private.
- Duplicate strategies are `skip` by default, `update`, and `create duplicate`.
- Duplicate comparison lowercases URL scheme and host, removes default ports and fragments, maps an empty path to `/`, and preserves query strings and non-root trailing slashes.
- Bookmark HTML accepts UTF-8 with or without BOM and declared encodings supported by the parser; unsupported encodings fail with an actionable message.
- Import files are limited to `20 MiB` by default.
- The database write is transactional; a failed import leaves no partial batch.
- Logo recognition runs asynchronously after imported records are committed.

### `FR-011` Bookmark data import and export

Provide portable bookmark exports and a lossless application bookmark format.

Acceptance criteria:

- Netscape HTML export preserves categories as folders and is accepted by major browsers.
- Application JSON export preserves categories, category icons, descriptions, link icon references, ordering, and visibility.
- Application JSON can be imported through the same preview and duplicate-resolution flow.
- Export scope can be public only, private only, or all bookmarks.
- Public export is available to the administrator only; no bulk export endpoint is exposed anonymously.

### `FR-012` Full backup

Create a consistent application backup containing the database, uploaded files, and a version manifest.

Acceptance criteria:

- A backup ZIP contains `nav.db`, `uploads/`, and `manifest.json`.
- The archive excludes `/data/backups` and temporary working files.
- The SQLite copy is created with a safe snapshot mechanism while the application is running.
- Active sessions are removed from the database copy included in the backup.
- The manifest contains application version, schema version, creation time, and integrity metadata.
- The administrator can create, list, download, and explicitly delete stored backups.
- Backup creation checks available space against current database and upload size before writing.
- Temporary archives are removed on success and failure; incomplete files never appear in the backup list.
- Backup creation failure does not leave a valid-looking partial archive.

### `FR-013` Full restore

Restore the application from a compatible full backup.

Acceptance criteria:

- Restore requires administrator re-authentication and a destructive-action confirmation.
- The upload is checked for archive traversal, size limits, manifest compatibility, checksums, and SQLite integrity.
- Restore archives are limited to `256 MiB` compressed and `512 MiB` extracted by default; extraction stops when the expanded limit is crossed.
- Restore verifies that free space can hold the extracted archive and the automatic pre-restore backup before changing active data.
- The current application state is backed up automatically before replacement.
- Database and files are swapped atomically where supported, with rollback on reopen or migration failure.
- All sessions are invalidated after restore and the administrator logs in using the restored credentials.
- An incompatible or corrupt backup changes no active data.

### `FR-014` In-place administration

Keep administration visually integrated with the navigation page instead of providing a separate legacy-style dashboard.

Acceptance criteria:

- Login adds edit affordances, drag handles, and private indicators to the normal page.
- Forms use responsive dialogs or drawers with loading, error, success, and disabled states.
- Destructive actions require confirmation and state exactly what data is affected.
- Icon-only controls have accessible names and tooltips where their meaning is not obvious.

### `FR-015` Category icons

Allow the administrator to assign an optional, visually consistent icon to each category for use in the sidebar and mobile category controls.

Acceptance criteria:

- Category create and edit forms include a searchable icon picker.
- The picker uses a curated Lucide registry with common category concepts and Chinese and English search terms.
- The database stores a stable icon key, not raw SVG markup.
- Unknown or removed icon keys render a neutral folder fallback without breaking layout.
- Category icons have fixed size, stroke, and alignment in desktop and mobile navigation.
- `No icon` is a supported explicit choice.
- Arbitrary SVG and image uploads are not accepted for category icons in the first release.

### `FR-016` Site identity settings

Allow the administrator to configure the site's identity from system settings.

Acceptance criteria:

- Settings include site name, optional site logo, optional favicon, and one accent color.
- The default name is `iiiu-nav`; the default mark is a bundled neutral navigation icon.
- The site name appears in the desktop sidebar, mobile header, document title, and relevant empty states.
- Uploaded site identity images follow `FR-006`, are stored under `/data/uploads/site`, and are included in full backups.
- Clearing a custom logo or favicon restores the bundled defaults without leaving a broken asset reference.
- Accent-color input is validated and mapped through design tokens in light and dark themes without reducing required contrast.

### `FR-017` Upgrade and migration safety

Protect persistent data when a new application version changes the database schema.

Acceptance criteria:

- Startup compares application and schema versions before serving requests.
- Before the first schema-changing migration, the application creates a pre-migration backup using the same safe snapshot rules as `FR-012`.
- Migrations run transactionally where SQLite permits and startup aborts without serving traffic if a migration fails.
- Failure restores or preserves the pre-migration database and reports the failed migration clearly.
- An older binary refuses a newer unsupported schema with a clear rollback instruction instead of attempting a downgrade.
- Operator documentation explains upgrade, pre-migration backup, rollback binary, and restore steps.

## 4. Visual Direction

The interface is a quiet personal workspace, not a traditional link directory or a monitoring dashboard.

- Desktop sidebar target width: approximately `216px`.
- Link module target height: `72-84px` with a fixed logo box.
- Card radius: no more than `8px`.
- Use neutral surfaces and one configurable accent color.
- Use subtle borders and restrained elevation; background images must not reduce readability.
- Avoid oversized welcome text, decorative gradients, excessive glass effects, weather, clocks, and marketing composition.
- Use system fonts with appropriate Chinese fallbacks.
- Interaction transitions target `120-180ms` and must respect reduced-motion preferences.
- The administration experience uses the same visual system as the public page.

Visual implementation is accepted only after screenshots are reviewed in light and dark themes at the four required viewport widths.

## 5. Non-Functional Requirements

### `NFR-001` Lightweight delivery

- Initial application JavaScript must remain at or below `120 KiB` gzip, excluding user-uploaded images.
- The production container image target is at or below `60 MiB`; exceeding it requires a documented reason.
- Idle application RSS target is at or below `64 MiB` with 500 sample links.
- Public navigation reads use bounded queries and remain responsive with 2,000 links.

### `NFR-002` Data safety

- Foreign keys are enabled.
- Migrations and multi-record mutations are transactional.
- A failed upload, import, backup, or restore leaves no orphaned active state.
- Restore and migration paths have automated integration coverage.

### `NFR-003` Security

- Private data authorization is tested at the API boundary.
- Uploads and archives are treated as untrusted input.
- Responses set appropriate content-type, framing, and sniffing protections.
- Secrets and live data under `/data` are excluded from Git.

### `NFR-004` Accessibility

- Core public and administrative workflows are operable by keyboard.
- Visible focus styles are present.
- Text and interactive controls meet WCAG AA contrast targets.
- Theme, search, menu, dialog, and drag alternatives expose usable accessible names.

### `NFR-005` UI system consistency

- Shared controls are implemented once in the internal UI primitive layer and consume design tokens.
- Buttons, fields, dialogs, drawers, menus, tooltips, toasts, file inputs, and empty states cover loading, error, disabled, focus, and dark-theme states.
- Lucide is the single interface and category icon language; link favicons and selected search-engine brand marks are separate content assets.
- Only referenced icons enter the production bundle. The icon picker must not import the complete Lucide catalog into the initial application chunk.
- Third-party interaction libraries provide behavior and positioning but do not dictate the visual style.

### `NFR-006` Runtime compatibility

- The first release UI is Simplified Chinese; user-facing text is centralized so later localization does not require component rewrites.
- Support the latest two stable major versions of Chrome, Edge, Firefox, and Safari at release time.
- Persist timestamps in UTC and render them in the browser's local timezone.
- The first release is served at a domain root and does not guarantee subpath deployment.
- Production images are built and smoke-tested for `linux/amd64` and `linux/arm64`.

### `NFR-007` Public and private response privacy

- Search indexing is disabled by default and can be enabled by the administrator in system settings.
- Private and administrative responses use `Cache-Control: no-store`.
- External link elements use `noopener noreferrer`.
- Responses set `Referrer-Policy: no-referrer` so external sites do not receive the navigation page URL.
- Privacy behavior is verified for anonymous, authenticated, cached, and search-engine crawler requests.

### `NFR-008` Maintainable Go code

- All Go code passes `gofmt`, `go vet`, and the applicable automated tests before a backlog item is completed.
- Non-generated Go files approaching `300` lines require a responsibility review. Split files that contain unrelated reasons to change; a cohesive implementation may remain intact with a short review note.
- The executable entry point remains limited to composition and lifecycle, and transport, product behavior, persistence, and external integrations remain independently testable.
- Shared behavior is centralized behind a small interface only when doing so improves leverage or locality; speculative abstractions and circular package dependencies are not accepted.

## 6. Verification Gates

Run the current scaffold verification from the repository root:

```text
npm ci
npm run check
npm run build
docker build --target go-test -t iiiu-nav:test .
docker build -t iiiu-nav:dev .
docker run --rm -p 8080:8080 -v iiiu-nav-data:/data iiiu-nav:dev
```

Each completed feature must also satisfy the applicable gates below.

| Gate            | Required evidence                                                                                                                            |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Backend         | Go formatting, static analysis, unit tests, and integration tests pass                                                                       |
| Frontend        | Formatting, lint, TypeScript check, and component tests pass                                                                                 |
| End-to-end      | Login, password change/reset, privacy, site settings, category/link CRUD, search, import/export, backup, restore, and upgrade workflows pass |
| Visual          | Playwright screenshots at `360`, `768`, `1024`, and `1440` in light and dark themes                                                          |
| Responsive      | No incoherent overlaps, clipping, or horizontal page overflow                                                                                |
| Security        | Anonymous private-data, privacy-header, upload validation, and restore validation tests pass                                                 |
| Build           | Production Docker image builds and starts with a fresh `/data` volume                                                                        |
| Compatibility   | Current supported browsers pass core smoke tests; `amd64` and `arm64` images build and start                                                 |
| Persistence     | Restart retains settings, categories, links, and uploaded assets                                                                             |
| Maintainability | Go package responsibilities, dependency direction, file-size review, formatting, static analysis, and focused tests satisfy `NFR-008`        |

## 7. Development Backlog

Status values:

- `TODO`: ready when dependencies are done.
- `IN_PROGRESS`: actively being implemented; at most one item.
- `BLOCKED`: cannot progress; reason must be recorded in Notes.
- `DONE`: acceptance criteria and verification gates have passed.

| ID        | Status | Depends on                      | Deliverable                                                                                | Completion criterion                                                                                                                    | Notes                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| --------- | ------ | ------------------------------- | ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `NAV-001` | `DONE` | -                               | Scaffold Go, Preact, TypeScript, Vite, Docker, and local development                       | Fresh checkout builds frontend and backend, runs locally, and serves embedded production assets                                         | Checks, production build, Docker build, API smoke test, and browser smoke test passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `NAV-002` | `DONE` | `NAV-001`                       | Data paths, SQLite connection, WAL, migrations, and repositories                           | Fresh and existing databases migrate idempotently; repository integration tests pass                                                    | Fresh/reopen migrations, repository persistence and constraints, Linux tests, production build, and named-volume restart passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `NAV-003` | `DONE` | `NAV-002`                       | Administrator bootstrap, login, sessions, logout, and middleware                           | All `FR-009` criteria pass, including anonymous rejection tests                                                                         | Unit, SQLite/HTTP integration, anonymous rejection, rate-limit, origin, Cookie, container bootstrap, reset, and restart checks passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| `NAV-100` | `DONE` | `NAV-001`                       | Design tokens, internal UI primitives, icon registry, floating elements, and drag behavior | `NFR-005` passes and component tests cover every shared control state                                                                   | 14 component tests, production build, 12.90 KiB gzip JavaScript, and light/dark screenshots at all four required widths passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `NAV-101` | `DONE` | `NAV-100`                       | Visual shell, sidebar, grid, cards, and theme                                              | `FR-001`, `FR-002`, and `FR-008` visual criteria pass with fixture data                                                                 | 22 frontend tests, full checks, production build, 22.17 KiB gzip JavaScript, theme persistence, and light/dark screenshots at all four required widths passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `NAV-102` | `DONE` | `NAV-002`, `NAV-003`, `NAV-101` | Public/private navigation read API and UI integration                                      | Anonymous and administrator views return exactly the permitted categories and links                                                     | SQLite aggregate, anonymous/authenticated HTTP integration, invalid-session fallback, 26 frontend tests, full checks, production build, and real empty-database browser smoke passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `NAV-103` | `DONE` | `NAV-101`, `NAV-102`            | Combined local and web search                                                              | Every `FR-007` criterion passes with keyboard and pointer tests                                                                         | Local name, description, and URL matching, keyboard and pointer behavior, web URL encoding, authenticated engine enable/disable/reordering, 36 frontend tests, full checks, production build, 31.97 KiB gzip JavaScript, and light/dark screenshots at all four required widths passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                                                                              |
| `NAV-201` | `DONE` | `NAV-003`, `NAV-102`            | In-place administrator mode and responsive forms                                           | Every `FR-014` criterion passes on desktop and mobile                                                                                   | Login, rate-limit and error handling, administrator menu, password change with session invalidation, logout, toast feedback, stable focus trapping, same-origin development proxy, 42 frontend tests, full checks, production build, 34.63 KiB gzip JavaScript, real Cookie-session browser flow, and light/dark responsive checks passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                                           |
| `NAV-202` | `DONE` | `NAV-201`                       | Category CRUD, icons, privacy, deletion flow, and ordering                                 | Every `FR-003` and `FR-015` criterion passes, including reload and API privacy tests                                                    | Authenticated category CRUD and full ordering APIs, transactional link move/delete choices, backend privacy, stable generated slugs, curated searchable icons, no-icon support, keyboard ordering, 47 frontend tests, full checks, production build, 36.86 KiB gzip JavaScript, real session persistence/privacy workflow, and light/dark responsive checks passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                  |
| `NAV-203` | `DONE` | `NAV-202`                       | Link CRUD, movement, deletion, and ordering                                                | Every `FR-004` criterion passes, including reload tests                                                                                 | Authenticated link CRUD and full per-category ordering APIs, transactional cross-category movement, strict HTTP/HTTPS validation, manual name, description, category and generated-logo overrides, in-card edit affordances, destructive confirmation, 51 frontend tests, full checks, production build, 38.52 KiB gzip JavaScript, real session create/edit/move/order/delete/reload workflow, and light/dark responsive checks passed on 2026-08-19                                                                                                                                                                                                                                                             |
| `NAV-204` | `DONE` | `NAV-203`                       | Metadata recognition, favicon caching, generated fallback, and logo upload                 | Every `FR-005` and logo portion of `FR-006` passes                                                                                      | Public-address-only fetching with DNS rebinding protection, bounded redirects, timeouts and response sizes, structured title/description/icon discovery, root-favicon fallback, normalized PNG caching from PNG/JPEG/WebP/ICO, authenticated 2 MiB uploads, generated server filenames, reference-aware cleanup, cached-logo preservation on refresh failures, individual and bulk retries, editable failure fallback, 54 frontend tests, full checks, production build, 40.25 KiB gzip JavaScript, real PNG HTTP lifecycle, and light/dark browser checks at 360/768/1024/1440 px passed on 2026-08-19                                                                                                           |
| `NAV-205` | `DONE` | `NAV-201`, `NAV-204`            | Site identity, appearance, background upload, indexing, and search-engine settings         | Site/background portions of `FR-006`, plus `FR-008`, `FR-016`, and the indexing setting in `NFR-007`, pass                              | Authenticated site-name, logo, favicon, accent, background, overlay and indexing settings; format-specific bounded image normalization and orphan cleanup; default favicon and crawler directives; 59 frontend tests, full checks, production build, 42.20 KiB gzip JavaScript, real PNG/ICO/WebP lifecycle, and light/dark browser checks at 360/768/1024/1440 px passed on 2026-08-19                                                                                                                                                                                                                                                                                                                           |
| `NAV-301` | `DONE` | `NAV-202`, `NAV-203`, `NAV-204` | Bookmark HTML and application JSON import preview and commit                               | Every import criterion in `FR-010` and `FR-011` passes with transactional tests                                                         | Authenticated 20 MiB HTML/JSON uploads, UTF-8 BOM and declared browser encodings, nested-folder flattening, Unicode case-folded category mapping, unfiled handling, canonical URL duplicate detection, skip/update/create strategies, atomic SQLite commit and rollback, asynchronous post-commit metadata refresh, 61 frontend tests, full checks, production build, 43.74 KiB gzip JavaScript, real preview/commit/cleanup HTTP lifecycle, and light/dark responsive drawer checks passed on 2026-08-19                                                                                                                                                                                                         |
| `NAV-302` | `DONE` | `NAV-301`                       | Browser HTML and application JSON export                                                   | Every export criterion in `FR-011` passes and exported HTML imports into a test parser                                                  | Authenticated HTML/JSON downloads with public/private/all scopes, Netscape folder output accepted by the import parser, lossless application JSON for category icons, descriptions, link icon references, ordering and visibility, attachment and no-store responses, anonymous rejection, 63 frontend tests, full checks, production build, 44.43 KiB gzip JavaScript, real scoped-download HTTP lifecycle, and light/dark responsive drawer checks passed on 2026-08-19                                                                                                                                                                                                                                         |
| `NAV-303` | `DONE` | `NAV-205`                       | Consistent full backup creation and management                                             | Every `FR-012` criterion passes against live WAL writes and uploaded assets                                                             | Runtime `VACUUM INTO` snapshots under concurrent WAL writes, session removal and integrity checks, conservative main/WAL/SHM/upload space budgeting, SHA-256 manifest, database and upload ZIP entries, hidden workspace cleanup and atomic publication, authenticated create/list/download/delete APIs, 65 frontend tests, full checks, production build, 45.59 KiB gzip JavaScript, Linux cross-build, real archive inspection and delete lifecycle, and light/dark responsive management checks passed on 2026-08-19                                                                                                                                                                                           |
| `NAV-304` | `DONE` | `NAV-303`                       | Validated, recoverable full restore                                                        | Every `FR-013` criterion passes for valid, corrupt, incompatible, and malicious archives                                                | Authenticated password reverification and explicit confirmation; 256 MiB compressed and 512 MiB extracted limits; traversal, duplicate, case-collision, path-policy, manifest, checksum, schema, administrator and SQLite integrity validation; pre-restore backup and space checks; request maintenance gate; database/upload swap with stage-specific rollback and reopen; restored-session invalidation; 67 frontend tests, full checks, production build, 46.59 KiB gzip JavaScript, real backup/modify/restore/relogin/cleanup lifecycle, and light/dark responsive restore checks passed on 2026-08-19                                                                                                      |
| `NAV-305` | `DONE` | `NAV-303`, `NAV-304`            | Version checks, pre-migration backup, safe upgrade, and rollback behavior                  | Every `FR-017` criterion passes for successful, failed, and incompatible migrations                                                     | Read-only startup preflight, explicit application/schema logging, safe v1 snapshot before v2, all-pending-migrations transaction, named failure and preserved backup reporting, newer-schema rollback guidance, successful/failed/incompatible integration tests, full checks, production build, Windows build, Linux amd64 cross-build, and real idempotent v1-to-v2 startup lifecycle passed on 2026-08-19                                                                                                                                                                                                                                                                                                      |
| `NAV-401` | `DONE` | `NAV-103`, `NAV-205`, `NAV-305` | Complete responsive and visual QA                                                          | Required screenshots are reviewed; no overlap, overflow, blank, or unreadable states remain                                             | Fixture-backed light/dark screenshots at 360/768/1024/1440 px; anonymous/private and administrator views; search results; login, category, link, settings, import, export, backup and restore overlays; loading failure and retry; computed overflow and dialog-bound checks; mobile sortable-control density fixes; 67 frontend tests and production build passed on 2026-08-19                                                                                                                                                                                                                                                                                                                                  |
| `NAV-402` | `DONE` | `NAV-305`                       | Security, privacy, and data-safety hardening                                               | `NFR-002`, `NFR-003`, `NFR-007`, and security verification gates pass                                                                   | Private API authorization and cache boundaries, cross-origin write rejection, bounded image/import/archive handling, traversal and rollback coverage, CSP/COOP/CORP/framing/sniffing/referrer/permissions headers, unknown-API isolation, crawler privacy, credential exclusions, Go 1.26.6 security toolchain pin, zero reachable govulncheck findings, zero npm audit findings, full tests, static analysis, container test stage, and production CSP rendering passed on 2026-08-19                                                                                                                                                                                                                            |
| `NAV-403` | `DONE` | `NAV-401`, `NAV-402`            | Performance and footprint optimization                                                     | `NFR-001` budgets are measured and met or exceptions are documented and accepted                                                        | Initial JavaScript 46.59 KiB gzip against 120 KiB; final distroless image 6.33 MiB against 60 MiB; 500-link idle container RSS 4.43 MiB against 64 MiB; indexed 2,000-link SQLite read 5.36 ms; 100 real public HTTP reads at 2,000 links measured 15.32 ms median and 17.37 ms P95 with a 354,232-byte response; repeatable benchmark/index-plan test, full checks, and production build passed on 2026-08-19                                                                                                                                                                                                                                                                                                    |
| `NAV-404` | `DONE` | `NAV-403`                       | Compatible images, deployment, and operator documentation                                  | `NFR-006` passes and a new operator can deploy, reset the password, back up, restore, upgrade, and roll back using only repository docs | Chromium, Firefox 153, and WebKit 26.5 core smoke tests passed at desktop and mobile widths; combined `linux/amd64` and `linux/arm64` OCI output built; native amd64 and QEMU user-mode arm64 binaries started and served the embedded application; hardened Compose deployment started with an isolated fresh volume; README and operator documentation cover initial deployment, TLS proxies, password reset, backup, restore, upgrades, schema rollback, multi-architecture publishing, compatibility, and troubleshooting; fresh dependency install, 67 frontend tests, Go tests, static analysis, production build, container test stage, and zero reachable dependency vulnerabilities passed on 2026-08-19 |

## 8. Delivery Sequence

1. Complete `NAV-001` through `NAV-003` to establish a secure, persistent foundation.
2. Complete `NAV-100` and `NAV-101` with fixture data before connecting full CRUD. Review the visual direction at this point.
3. Complete public navigation and search through `NAV-103`.
4. Complete in-place management through `NAV-205`.
5. Complete import, export, backup, restore, and migration safety through `NAV-305`.
6. Complete quality gates and operator documentation through `NAV-404`.

Do not mark a backlog item `DONE` from code inspection alone. Run and record the checks required by its completion criterion.

## 9. Confirmed Defaults

- The homepage is public; only private categories and administration require login.
- Bulk imports default to private visibility.
- Unfiled imported bookmarks go to `未分类` in the Chinese UI.
- Links open in a new tab.
- Search combines local matching with selectable external engines.
- The application stores one background image, with configurable overlay strength.
- Complete recovery uses a ZIP containing both SQLite data and uploaded assets.
- Automatic scheduled backups are outside the first release; manual, pre-restore, and pre-migration backups are included.
- Site name, site logo, favicon, accent color, and search indexing are administrator-configurable system settings.
- The default site identity is `iiiu-nav` with a bundled neutral navigation mark.
- Search indexing defaults to disabled.
- The first release is Simplified Chinese, root-path deployed, and built for `linux/amd64` and `linux/arm64`.
- Administrator passwords contain 12 or more characters and at most 1024 bytes. Bootstrap and recovery read `ADMIN_PASSWORD_FILE`; trailing newlines are ignored.
- Argon2id uses `64 MiB` memory, three iterations, and four threads. Sessions expire after 30 days and only a SHA-256 token hash is stored.
- Five failed logins from one direct client address within 10 minutes block further attempts until the window expires; a successful login clears that client's failures.
- State-changing HTTP requests require a matching `Origin` or `Referer`, reject cross-site Fetch Metadata, and use a host-only `HttpOnly`, `SameSite=Strict` session cookie.
- TLS-terminating reverse proxies preserve the public `Host` and set `X-Forwarded-Proto: https`.

## 10. Change Log

| Version | Date         | Change                                                                                                                                                                                                                   |
| ------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `1.23`  | `2026-08-19` | Completed supported-browser smoke coverage, multi-architecture OCI builds, hardened Compose deployment, production operation and recovery documentation, final quality gates, and release-candidate status               |
| `1.22`  | `2026-08-19` | Added a repeatable indexed 2,000-link repository benchmark and completed JavaScript, image, idle-memory, and public-HTTP performance budget measurements                                                                 |
| `1.21`  | `2026-08-19` | Added strict browser security headers, non-cacheable cross-origin and unknown-API responses, Docker credential exclusions, a patched Go 1.26.6 toolchain pin, and dependency vulnerability verification                  |
| `1.20`  | `2026-08-19` | Completed full light/dark responsive visual regression, public/private and administrative overlay review, overflow measurements, failure-state recovery, and mobile sortable-control density fixes                       |
| `1.19`  | `2026-08-19` | Added read-only schema preflight, automatic pre-migration snapshots, transactional migration batches, failed-upgrade preservation and diagnostics, incompatible-newer-schema refusal, and a real v1-to-v2 migration      |
| `1.18`  | `2026-08-19` | Added password-confirmed full restore, bounded and traversal-safe extraction, manifest and SQLite validation, automatic pre-restore backup, maintenance gating, atomic data exchange, rollback, and session invalidation |
| `1.17`  | `2026-08-19` | Added consistent SQLite snapshot backups, session stripping, upload archiving, integrity manifests, disk-space checks, atomic publication, authenticated lifecycle APIs, and in-place backup management                  |
| `1.16`  | `2026-08-19` | Added administrator-only Netscape HTML and lossless application JSON exports, public/private/all scopes, verified round trips, download responses, and responsive export UI                                              |
| `1.15`  | `2026-08-19` | Added browser HTML and application JSON import preview, encoding-aware nested-folder parsing, Unicode category matching, canonical duplicate strategies, transactional commit, and responsive import UI                  |
| `1.14`  | `2026-08-19` | Added configurable site identity, normalized logo/favicon/background uploads, accent and overlay appearance settings, safe image cleanup, indexing controls, crawler directives, and responsive settings UI              |
| `1.13`  | `2026-08-19` | Added SSRF-resistant metadata recognition, bounded favicon discovery and caching, normalized logo uploads, generated fallbacks, reference-aware cleanup, manual and bulk refresh controls, and responsive image previews |
| `1.12`  | `2026-08-19` | Added authenticated link creation, editing, category movement, full ordering, strict URL validation, manual generated-logo overrides, in-card edit controls, and deletion confirmation                                   |
| `1.11`  | `2026-08-19` | Added authenticated category creation, editing, privacy, full ordering, transactional deletion choices, generated slugs, and a searchable curated category icon picker                                                   |
| `1.10`  | `2026-08-19` | Added in-place administrator login, password change and logout flows, responsive authentication forms, session feedback, stable modal focus handling, and a same-origin development proxy                                |
| `1.9`   | `2026-08-19` | Added combined local and external search, keyboard and pointer result handling, fixed engine configuration, and authenticated engine enable, disable, and ordering settings                                              |
| `1.8`   | `2026-08-19` | Added bounded navigation reads, backend public/private filtering with optional sessions, API response validation, and loading, error, retry, empty, and administrator UI states                                          |
| `1.7`   | `2026-08-19` | Added the responsive navigation shell, desktop and mobile category controls, stable link cards, fixture states, and persistent light, dark, and system themes                                                            |
| `1.6`   | `2026-08-19` | Added design tokens, shared UI primitives, curated category icons, collision-aware floating elements, accessible reordering, and component-test coverage                                                                 |
| `1.5`   | `2026-08-19` | Added administrator bootstrap, Argon2id policy, hashed sessions, login rate limiting, origin protection, password recovery, and authentication verification defaults                                                     |
| `1.4`   | `2026-08-19` | Fixed the SQLite driver and connection policy, documented data-root and timestamp behavior, and completed the persistent data foundation                                                                                 |
| `1.3`   | `2026-08-19` | Added Go module boundaries, dependency direction, maintainability requirements, and a file responsibility review threshold                                                                                               |
| `1.2`   | `2026-08-19` | Added site identity settings, credential recovery, import edge cases, backup limits, upgrade safety, runtime compatibility, and privacy defaults                                                                         |
| `1.1`   | `2026-08-19` | Added the internal UI component strategy, library decisions, category icons, and their acceptance criteria                                                                                                               |
| `1.0`   | `2026-08-19` | Initial confirmed scope, architecture, acceptance criteria, and backlog                                                                                                                                                  |
