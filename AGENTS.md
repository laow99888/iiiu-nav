# Repository Instructions

- Before planning, implementing, testing, or reviewing product work, read `docs/DEVELOPMENT_PLAN.md` completely. It is the source of truth for scope, architecture, acceptance criteria, and task status.
- When starting a development item, set its status to `IN_PROGRESS`. When its acceptance criteria and required checks pass, set it to `DONE`. Record blockers as `BLOCKED` with a short reason.
- Keep at most one item `IN_PROGRESS`. Work in dependency order unless the user explicitly changes priorities.
- Update the development plan in the same change when implementation changes an accepted behavior, architecture decision, verification command, or task status.
- Preserve the project's single-user, self-hosted, lightweight boundary. New subsystems or dependencies require a concrete accepted requirement.
