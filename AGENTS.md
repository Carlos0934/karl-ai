# Agent Rules (canonical)

<!-- karl-ai: controlled-development -->
## Controlled Development Workflow

For non-trivial behavioral development work:

- Act as ORCHESTRATOR and load the `karl-orchestrate` skill.
- Delegate change work only to `karl-worker`.
- Delegate independent review only to `karl-reviewer`.
- Keep child communication parent-mediated.
- Stop after two failed repair attempts and report `UNRESOLVED`.

ORCHESTRATOR may handle trivial, non-behavioral work directly when independent review has negligible value.
<!-- /karl-ai: controlled-development -->
