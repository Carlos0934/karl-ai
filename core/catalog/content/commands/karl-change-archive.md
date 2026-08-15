# Archive A Validated Change

1. Follow the `karl-change-lifecycle` skill and its resources. Require state
   `validated`; verify with `karl-ai change status <name>`.
2. Run `karl-ai change validate <name> archive` and resolve every blocker:
   dependencies archived, `conflicts_with` empty, and no uncommitted
   implementation files outside `docs/` and the active package.
3. Move the package with `karl-ai change archive <name>`; it lands in
   `changes/archive/YYYY-MM-DD-<name>/`. Never move files manually or bypass a
   failed gate.
4. Report that archive is not complete yet: the CLI never commits. Archive
   completes only when the developer explicitly requests the final commit that
   contains baseline updates and the package move. Propose a concise commit
   message and wait for that request; do not create any commit otherwise.

Every state change goes through `karl-ai change` and no commit is created
automatically.

Return the destination path, gate results, files moved, an explicit statement
that no commit was created, and the proposed final commit message.
