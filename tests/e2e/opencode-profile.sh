#!/bin/sh
# opencode-profile.sh - POSIX sh E2E test mirroring tests/e2e/opencode-profile.ps1.
#
# Exercises scripts/apply-opencode-profile.sh (project and global scopes)
# against isolated temporary directories. POSIX sh only; needs python3 for
# JSON assertions (the applier itself requires it too).

set -eu

REPO_ROOT=$(CDPATH='' cd "$(dirname -- "$0")/../.." && pwd -P) || exit 1
APPLIER=$REPO_ROOT/scripts/apply-opencode-profile.sh

command -v python3 >/dev/null 2>&1 || {
    printf 'FAIL: python3 is required for this test.\n' >&2
    exit 1
}

TMPBASE=${TMPDIR:-/tmp}
WORK_T=$(mktemp -d "$TMPBASE/karl-profile-e2e.XXXXXX")

cleanup() {
    rm -rf -- "$WORK_T"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

fail() {
    printf 'FAIL: %s\n' "$1" >&2
    exit 1
}

assert_eq() {
    # assert_eq EXPECTED ACTUAL MESSAGE
    if [ "$1" != "$2" ]; then
        printf 'FAIL: %s\n' "$3" >&2
        printf '  expected: %s\n  actual:   %s\n' "$1" "$2" >&2
        exit 1
    fi
}

# json_get FILE KEY [KEY...] — print a nested JSON value with python3.
json_get() {
    _jg_file=$1
    shift
    python3 - "$_jg_file" "$@" <<'PYEOF'
import json
import sys
with open(sys.argv[1], encoding='utf-8') as fh:
    node = json.load(fh)
for key in sys.argv[2:]:
    node = node[key]
print(node)
PYEOF
}

# --- 1. Profile listing -------------------------------------------------------
LIST_OUT=$WORK_T/list.txt
sh "$APPLIER" --list > "$LIST_OUT" || fail 'Profile listing failed.'
grep -qx 'karl-default' "$LIST_OUT" || fail "Profile list is missing 'karl-default'."
grep -qx 'openai' "$LIST_OUT" || fail "Profile list is missing 'openai'."

# --- 2. Real profiles parse (dry runs write nothing) --------------------------
for _name in karl-default openai; do
    sh "$APPLIER" --profile "$_name" --scope project --target-dir "$WORK_T/dry-$_name" --dry-run \
        || fail "Dry run of profile '$_name' failed."
    [ -e "$WORK_T/dry-$_name/opencode.json" ] && fail "Dry run of '$_name' wrote a file."
done

# --- 3. Unknown profile fails without writing ----------------------------------
if sh "$APPLIER" --profile no-such-profile --scope project --target-dir "$WORK_T/unknown"; then
    fail 'Unknown profile should fail.'
else
    _code=$?
    assert_eq 2 "$_code" 'Unknown profile should exit 2.'
fi
[ -e "$WORK_T/unknown/opencode.json" ] && fail 'Unknown profile wrote a file.'

# --- 4. Merge preserves unrelated keys, overwrites stale karl models -----------
cat > "$WORK_T/synthetic.json" <<'EOF'
{
  "karl-orchestrator": "test-provider/test-orch#medium",
  "karl-worker": "test-provider/test-work#xhigh",
  "karl-scout": "test-provider/test-scout#high",
  "karl-verify": "test-provider/test-verify#high",
  "karl-reviewer": "test-provider/test-review#high"
}
EOF

mkdir -p "$WORK_T/project"
cat > "$WORK_T/project/opencode.json" <<'EOF'
{
  "$schema": "https://opencode.ai/config.json",
  "model": "some-provider/some-model",
  "mcp": {"jira": {"type": "remote", "enabled": false}},
  "agent": {
    "build": {"mode": "primary", "model": "some-provider/other"},
    "karl-worker": {"mode": "subagent", "model": "stale-provider/stale"}
  }
}
EOF

sh "$APPLIER" --profile "$WORK_T/synthetic.json" --scope project --target-dir "$WORK_T/project" \
    || fail 'Profile apply failed.'
DEST=$WORK_T/project/opencode.json
assert_eq 'some-provider/some-model' "$(json_get "$DEST" model)" 'Global model was not preserved.'
assert_eq 'False' "$(json_get "$DEST" mcp jira enabled)" 'Unrelated mcp config was not preserved.'
assert_eq 'some-provider/other' "$(json_get "$DEST" agent build model)" 'Non-karl agent was not preserved.'
assert_eq 'test-provider/test-work#xhigh' "$(json_get "$DEST" agent karl-worker model)" 'Stale karl-worker model was not overwritten.'
assert_eq 'subagent' "$(json_get "$DEST" agent karl-worker mode)" 'Existing karl-worker fields were dropped.'
assert_eq 'test-provider/test-orch#medium' "$(json_get "$DEST" agent karl-orchestrator model)" 'karl-orchestrator was not added.'
assert_eq 'test-provider/test-review#high' "$(json_get "$DEST" agent karl-reviewer model)" 'karl-reviewer was not added.'
_BACKUPS=$(find "$WORK_T/project" -maxdepth 1 -name 'opencode.json.bak-*' | wc -l | tr -d ' ')
assert_eq 1 "$_BACKUPS" 'Expected exactly one backup.'

# --- 5. Idempotence -------------------------------------------------------------
_BEFORE=$(cksum "$DEST")
sh "$APPLIER" --profile "$WORK_T/synthetic.json" --scope project --target-dir "$WORK_T/project" \
    || fail 'Idempotent re-apply failed.'
assert_eq "$_BEFORE" "$(cksum "$DEST")" 'Idempotent re-apply rewrote the file.'
_BACKUPS=$(find "$WORK_T/project" -maxdepth 1 -name 'opencode.json.bak-*' | wc -l | tr -d ' ')
assert_eq 1 "$_BACKUPS" 'Idempotent re-apply created a backup.'

# --- 6. --force skips the backup -------------------------------------------------
python3 - "$WORK_T/synthetic.json" <<'PYEOF'
import json
import sys
path = sys.argv[1]
with open(path, encoding='utf-8') as fh:
    data = json.load(fh)
data['karl-scout'] = 'test-provider/test-scout-v2#high'
with open(path, 'w', encoding='utf-8') as fh:
    json.dump(data, fh, indent=2)
    fh.write('\n')
PYEOF
sh "$APPLIER" --profile "$WORK_T/synthetic.json" --scope project --target-dir "$WORK_T/project" --force \
    || fail 'Forced apply failed.'
_BACKUPS=$(find "$WORK_T/project" -maxdepth 1 -name 'opencode.json.bak-*' | wc -l | tr -d ' ')
assert_eq 1 "$_BACKUPS" '--force still created a backup.'
assert_eq 'test-provider/test-scout-v2#high' "$(json_get "$DEST" agent karl-scout model)" '--force did not apply the new model.'

# --- 7. Global scope --------------------------------------------------------------
sh "$APPLIER" --profile openai --scope global --home "$WORK_T/home" \
    || fail 'Global apply failed.'
GDEST=$WORK_T/home/.config/opencode/opencode.json
[ -f "$GDEST" ] || fail "Global apply did not create '$GDEST'."
assert_eq 'openai/gpt-6-sol#medium' "$(json_get "$GDEST" agent karl-orchestrator model)" 'Global karl-orchestrator mismatch.'
assert_eq 'openai/gpt-6-luna#xhigh' "$(json_get "$GDEST" agent karl-worker model)" 'Global karl-worker mismatch.'
assert_eq 'openai/gpt-6-luna#high' "$(json_get "$GDEST" agent karl-scout model)" 'Global karl-scout mismatch.'
assert_eq 'openai/gpt-6-luna#high' "$(json_get "$GDEST" agent karl-verify model)" 'Global karl-verify mismatch.'
assert_eq 'openai/gpt-6-sol#high' "$(json_get "$GDEST" agent karl-reviewer model)" 'Global karl-reviewer mismatch.'

# --- 8. Invalid inputs fail closed -------------------------------------------------
printf '{not json' > "$WORK_T/bad.json"
if sh "$APPLIER" --profile "$WORK_T/bad.json" --scope project --target-dir "$WORK_T/bad"; then
    fail 'Invalid profile JSON should fail.'
fi

printf '{"build": {"model": "x/y"}}' > "$WORK_T/evil.json"
if sh "$APPLIER" --profile "$WORK_T/evil.json" --scope project --target-dir "$WORK_T/evil"; then
    fail 'Non-karl profile entry should fail.'
fi

mkdir -p "$WORK_T/baddest"
printf '[1,2]' > "$WORK_T/baddest/opencode.json"
if sh "$APPLIER" --profile "$WORK_T/synthetic.json" --scope project --target-dir "$WORK_T/baddest"; then
    fail 'Non-object destination should fail.'
fi
assert_eq '[1,2]' "$(cat "$WORK_T/baddest/opencode.json")" 'Failed apply mutated the destination.'

# --- 9. Remote profile via --profile-url (file:// keeps it offline) ---------------
command -v curl >/dev/null 2>&1 || fail 'curl is required for the --profile-url test.'
mkdir -p "$WORK_T/url-project"
sh "$APPLIER" --profile-url "file://$WORK_T/synthetic.json" --scope project --target-dir "$WORK_T/url-project" \
    || fail 'Profile URL apply failed.'
UDEST=$WORK_T/url-project/opencode.json
[ -f "$UDEST" ] || fail 'Profile URL apply did not create the destination.'
assert_eq 'test-provider/test-work#xhigh' "$(json_get "$UDEST" agent karl-worker model)" 'Profile URL karl-worker mismatch.'
assert_eq 'test-provider/test-orch#medium' "$(json_get "$UDEST" agent karl-orchestrator model)" 'Profile URL karl-orchestrator mismatch.'

if sh "$APPLIER" --profile-url 'file:///no-such-profile-here.json' --scope project --target-dir "$WORK_T/url-missing"; then
    fail 'Unreachable profile URL should fail.'
fi
[ -e "$WORK_T/url-missing/opencode.json" ] && fail 'Failed URL apply wrote a file.'

printf 'PASS: opencode profile apply is scoped, preserving, idempotent, and fails closed.\n'
