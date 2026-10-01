# 0004: TOML config with a version key

**Status:** Accepted

**Context.** Users want to keep settings in a file. Config formats are hard to change after release.

**Decision.** Use TOML (comments allowed, readable). Require no file; flags override the file. The file has
a `version` key (currently 1), unknown keys are errors, and a file written by a newer release is refused with
a clear message.

**Consequences.** A future release can read and upgrade old files. A typo in a key is reported instead of ignored.
Moving to another format later would be a breaking change and would need a major version.
