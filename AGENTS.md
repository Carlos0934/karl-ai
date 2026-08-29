# Agent Rules (canonical)

<!-- karl-ai: controlled-development -->
## Controlled Development Workflow

For non-trivial behavioral development work:

- Act as MAIN and load the `karl-orchestrate` skill.
- Delegate implementation only to `karl-implementer`.
- Delegate independent verification only to `karl-verifier`.
- Keep worker delegation depth at zero.
- Do not forward one worker's reasoning transcript to another worker.
- Stop after two failed repair attempts and report `UNRESOLVED`.

MAIN may handle trivial, non-behavioral work directly when independent verification has negligible value.
<!-- /karl-ai: controlled-development -->
