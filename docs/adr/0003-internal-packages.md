# 0003: Keep all packages under `internal/`

**Status:** Accepted

**Context.** A public Go API is a promise. Making one before the design is stable forces breaking changes
on users, or freezes mistakes.

**Decision.** Everything lives under `internal/`. Go prevents other modules from importing it, so we can
rename and restructure freely until we choose to publish a library.

**Consequences.** There is no Go library for others to import yet. When a part is stable and wanted, we move
it out in a minor release with a note in the changelog.
