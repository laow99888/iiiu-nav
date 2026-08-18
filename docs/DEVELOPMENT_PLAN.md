# iiiu-nav Development Plan

> Document ID: `IIU-NAV-PLAN-001`
> Version: `1.2`
> Updated: `2026-08-19`
> Product status: `REQUIREMENTS_CONFIRMED`
> Development status: `NOT_STARTED`

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

| Area | Decision |
| --- | --- |
| Backend | Go, using the standard HTTP stack unless a small dependency removes concrete complexity |
| Frontend | Preact + TypeScript + Vite |
| Styling | Plain CSS with design tokens; no full UI framework |
| UI primitives | Small in-repository components for buttons, fields, dialogs, drawers, menus, tooltips, toasts, and empty states |
| Interface icons | `lucide-preact`, imported through a curated registry |
| Brand icons | Selected build-time imports from `simple-icons`; no runtime icon CDN |
| Floating elements | `@floating-ui/dom` for collision-aware menus, pickers, and tooltips |
| Reordering | `@atlaskit/pragmatic-drag-and-drop` core package with non-drag alternatives |
| Database | SQLite with WAL mode and versioned migrations |
| Uploaded files | `/data/uploads/logos`, `/data/uploads/backgrounds`, and `/data/uploads/site` |
| Backups | `/data/backups` |
| Deployment | Multi-stage Docker build; frontend embedded in the Go binary |
| Runtime | One container, one process, one mounted `/data` volume |
| Authentication | One administrator password and server-side sessions |

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

## 6. Verification Gates

Commands will be filled in after scaffolding establishes the project scripts. Until then, each completed feature must satisfy the applicable gates below.

| Gate | Required evidence |
| --- | --- |
| Backend | Go formatting, static analysis, unit tests, and integration tests pass |
| Frontend | Formatting, lint, TypeScript check, and component tests pass |
| End-to-end | Login, password change/reset, privacy, site settings, category/link CRUD, search, import/export, backup, restore, and upgrade workflows pass |
| Visual | Playwright screenshots at `360`, `768`, `1024`, and `1440` in light and dark themes |
| Responsive | No incoherent overlaps, clipping, or horizontal page overflow |
| Security | Anonymous private-data, privacy-header, upload validation, and restore validation tests pass |
| Build | Production Docker image builds and starts with a fresh `/data` volume |
| Compatibility | Current supported browsers pass core smoke tests; `amd64` and `arm64` images build and start |
| Persistence | Restart retains settings, categories, links, and uploaded assets |

## 7. Development Backlog

Status values:

- `TODO`: ready when dependencies are done.
- `IN_PROGRESS`: actively being implemented; at most one item.
- `BLOCKED`: cannot progress; reason must be recorded in Notes.
- `DONE`: acceptance criteria and verification gates have passed.

| ID | Status | Depends on | Deliverable | Completion criterion | Notes |
| --- | --- | --- | --- | --- | --- |
| `NAV-001` | `TODO` | - | Scaffold Go, Preact, TypeScript, Vite, Docker, and local development | Fresh checkout builds frontend and backend, runs locally, and serves embedded production assets | - |
| `NAV-002` | `TODO` | `NAV-001` | Data paths, SQLite connection, WAL, migrations, and repositories | Fresh and existing databases migrate idempotently; repository integration tests pass | - |
| `NAV-003` | `TODO` | `NAV-002` | Administrator bootstrap, login, sessions, logout, and middleware | All `FR-009` criteria pass, including anonymous rejection tests | - |
| `NAV-100` | `TODO` | `NAV-001` | Design tokens, internal UI primitives, icon registry, floating elements, and drag behavior | `NFR-005` passes and component tests cover every shared control state | - |
| `NAV-101` | `TODO` | `NAV-100` | Visual shell, sidebar, grid, cards, and theme | `FR-001`, `FR-002`, and `FR-008` visual criteria pass with fixture data | - |
| `NAV-102` | `TODO` | `NAV-002`, `NAV-003`, `NAV-101` | Public/private navigation read API and UI integration | Anonymous and administrator views return exactly the permitted categories and links | - |
| `NAV-103` | `TODO` | `NAV-101`, `NAV-102` | Combined local and web search | Every `FR-007` criterion passes with keyboard and pointer tests | - |
| `NAV-201` | `TODO` | `NAV-003`, `NAV-102` | In-place administrator mode and responsive forms | Every `FR-014` criterion passes on desktop and mobile | - |
| `NAV-202` | `TODO` | `NAV-201` | Category CRUD, icons, privacy, deletion flow, and ordering | Every `FR-003` and `FR-015` criterion passes, including reload and API privacy tests | - |
| `NAV-203` | `TODO` | `NAV-202` | Link CRUD, movement, deletion, and ordering | Every `FR-004` criterion passes, including reload tests | - |
| `NAV-204` | `TODO` | `NAV-203` | Metadata recognition, favicon caching, generated fallback, and logo upload | Every `FR-005` and logo portion of `FR-006` passes | - |
| `NAV-205` | `TODO` | `NAV-201`, `NAV-204` | Site identity, appearance, background upload, indexing, and search-engine settings | Site/background portions of `FR-006`, plus `FR-008`, `FR-016`, and the indexing setting in `NFR-007`, pass | - |
| `NAV-301` | `TODO` | `NAV-202`, `NAV-203`, `NAV-204` | Bookmark HTML and application JSON import preview and commit | Every import criterion in `FR-010` and `FR-011` passes with transactional tests | - |
| `NAV-302` | `TODO` | `NAV-301` | Browser HTML and application JSON export | Every export criterion in `FR-011` passes and exported HTML imports into a test parser | - |
| `NAV-303` | `TODO` | `NAV-205` | Consistent full backup creation and management | Every `FR-012` criterion passes against live WAL writes and uploaded assets | - |
| `NAV-304` | `TODO` | `NAV-303` | Validated, recoverable full restore | Every `FR-013` criterion passes for valid, corrupt, incompatible, and malicious archives | - |
| `NAV-305` | `TODO` | `NAV-303`, `NAV-304` | Version checks, pre-migration backup, safe upgrade, and rollback behavior | Every `FR-017` criterion passes for successful, failed, and incompatible migrations | - |
| `NAV-401` | `TODO` | `NAV-103`, `NAV-205`, `NAV-305` | Complete responsive and visual QA | Required screenshots are reviewed; no overlap, overflow, blank, or unreadable states remain | - |
| `NAV-402` | `TODO` | `NAV-305` | Security, privacy, and data-safety hardening | `NFR-002`, `NFR-003`, `NFR-007`, and security verification gates pass | - |
| `NAV-403` | `TODO` | `NAV-401`, `NAV-402` | Performance and footprint optimization | `NFR-001` budgets are measured and met or exceptions are documented and accepted | - |
| `NAV-404` | `TODO` | `NAV-403` | Compatible images, deployment, and operator documentation | `NFR-006` passes and a new operator can deploy, reset the password, back up, restore, upgrade, and roll back using only repository docs | - |

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
- Automatic scheduled backups are outside the first release; manual and pre-restore backups are included.
- Site name, site logo, favicon, accent color, and search indexing are administrator-configurable system settings.
- The default site identity is `iiiu-nav` with a bundled neutral navigation mark.
- Search indexing defaults to disabled.
- The first release is Simplified Chinese, root-path deployed, and built for `linux/amd64` and `linux/arm64`.

## 10. Change Log

| Version | Date | Change |
| --- | --- | --- |
| `1.2` | `2026-08-19` | Added site identity settings, credential recovery, import edge cases, backup limits, upgrade safety, runtime compatibility, and privacy defaults |
| `1.1` | `2026-08-19` | Added the internal UI component strategy, library decisions, category icons, and their acceptance criteria |
| `1.0` | `2026-08-19` | Initial confirmed scope, architecture, acceptance criteria, and backlog |
