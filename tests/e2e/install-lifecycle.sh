#!/bin/sh
# install-lifecycle.sh - POSIX sh E2E test mirroring tests/e2e/install-lifecycle.ps1.
#
# Exercises install.sh (install and uninstall paths) against an isolated
# temporary home directory. POSIX sh only; needs only Debian base utils
# (no git, no Pester). OpenCode link assertions only; Codex links are not
# asserted but their presence does not break the test.

set -eu

REPO_ROOT=$(CDPATH='' cd "$(dirname -- "$0")/../.." && pwd -P) || exit 1
INSTALLER=$REPO_ROOT/install.sh
MARKER_START='<!-- karl-ai: controlled-development -->'
MARKER_END='<!-- /karl-ai: controlled-development -->'

TMPBASE=${TMPDIR:-/tmp}
HOME_T=$(mktemp -d "$TMPBASE/karl-install-e2e-home.XXXXXX")
WORK_T=$(mktemp -d "$TMPBASE/karl-install-e2e-work.XXXXXX")

cleanup() {
    if [ -n "${HOME_T:-}" ]; then rm -rf -- "$HOME_T"; fi
    if [ -n "${WORK_T:-}" ]; then rm -rf -- "$WORK_T"; fi
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

run_install() {
    sh "$INSTALLER" --target-home "$HOME_T" "$@"
}

fingerprint() {
    # fingerprint DIR: deterministic listing of entry type, relative path, and
    # content hash (or link target) for everything under DIR.
    (
        CDPATH='' cd "$1" || exit 1
        find . -print | LC_ALL=C sort | while IFS= read -r entry; do
            if [ "$entry" = '.' ]; then
                continue
            fi
            if [ -L "$entry" ]; then
                printf 'L:%s:%s\n' "$entry" "$(readlink -- "$entry")"
            elif [ -d "$entry" ]; then
                printf 'D:%s\n' "$entry"
            else
                printf 'F:%s:%s\n' "$entry" "$(cksum "$entry")"
            fi
        done
    )
}

count_backups() {
    _cb_count=0
    if [ -d "$HOME_T/.agents-backup" ]; then
        for _cb_dir in "$HOME_T/.agents-backup"/*; do
            if [ -d "$_cb_dir" ]; then
                _cb_count=$((_cb_count + 1))
            fi
        done
    fi
    printf '%s\n' "$_cb_count"
}

resolve_link() {
    # resolve_link LINK: print the canonical absolute path LINK points to,
    # canonicalized via cd -P / pwd -P (never readlink -f).
    _rl_link=$1
    _rl_value=$(readlink -- "$_rl_link") || fail "Expected a symlink at '$_rl_link'."
    case "$_rl_value" in
        /*) _rl_cand=$_rl_value ;;
        *) _rl_cand=$(dirname -- "$_rl_link")/$_rl_value ;;
    esac
    _rl_dir=$(CDPATH='' cd "$(dirname -- "$_rl_cand")" 2>/dev/null && pwd -P) \
        || fail "Cannot canonicalize link target '$_rl_cand' of '$_rl_link'."
    printf '%s/%s\n' "$_rl_dir" "$(basename -- "$_rl_cand")"
}

check_opencode_links() {
    _cl_count=0
    for _cl_entry in "$OPENCODE_AGENTS"/*; do
        if [ -L "$_cl_entry" ] || [ -e "$_cl_entry" ]; then
            _cl_count=$((_cl_count + 1))
        fi
    done
    assert_eq 3 "$_cl_count" "Expected exactly 3 entries in '$OPENCODE_AGENTS', found $_cl_count."
    for _cl_name in karl-orchestrator.md karl-worker.md karl-reviewer.md; do
        _cl_link=$OPENCODE_AGENTS/$_cl_name
        [ -L "$_cl_link" ] || fail "Expected a symlink at '$_cl_link'."
        _cl_actual=$(resolve_link "$_cl_link")
        _cl_expected=$REPO_ROOT/harnesses/opencode/agents/$_cl_name
        assert_eq "$_cl_expected" "$_cl_actual" "Wrong link target for '$_cl_link': got '$_cl_actual'."
    done
}

assert_block() {
    # assert_block TARGET_FILE EXPECTED_BLOCK_FILE LABEL
    _ab_file=$1
    _ab_expected=$2
    _ab_label=$3
    _ab_count=$(LC_ALL=C grep -cF -- "$MARKER_START" "$_ab_file" || true)
    assert_eq 1 "$_ab_count" "Expected exactly one managed block in '$_ab_file' ($_ab_label), found $_ab_count."
    awk -v sstart="$MARKER_START" -v send="$MARKER_END" '
        !found && index($0, sstart) { found = 1 }
        found {
            print
            if (index($0, send)) { ok = 1; exit }
        }
        END { if (!ok) exit 1 }
    ' "$_ab_file" > "$WORK_T/block-installed.txt" || fail "Managed block not found in '$_ab_file'."
    cmp -s "$WORK_T/block-installed.txt" "$_ab_expected" \
        || fail "Managed block in '$_ab_file' does not byte-match the canonical block ($_ab_label)."
}

extract_prefix() {
    # extract_prefix FILE OUT: everything before the managed block start line.
    awk -v sstart="$MARKER_START" 'index($0, sstart) { exit } { print }' "$1" > "$2"
}

assert_no_marker() {
    if LC_ALL=C grep -qF -- "$MARKER_START" "$1"; then
        fail "Managed block survived in '$1' ($2)."
    fi
}

# --- Canonical managed block -------------------------------------------------

CANON_BLOCK=$(awk -v sstart="$MARKER_START" -v send="$MARKER_END" '
    !found && index($0, sstart) { found = 1 }
    found {
        print
        if (index($0, send)) { ok = 1; exit }
    }
    END { if (!ok) exit 1 }
' "$REPO_ROOT/AGENTS.md") || fail 'Canonical AGENTS.md does not contain the managed karl-ai block.'

# Normalize to LF; CRLF expectations are derived from the LF block.
printf '%s\n' "$CANON_BLOCK" | tr -d '\r' > "$WORK_T/block-lf.txt"
# awk instead of `sed 's/$/\r/'`: BSD sed (macOS) handles the substitution
# differently, awk's printf is portable.
awk '{ printf "%s\r\n", $0 }' "$WORK_T/block-lf.txt" > "$WORK_T/block-crlf.txt"

# --- Scenario setup ----------------------------------------------------------

OPENCODE_ROOT=$HOME_T/.config/opencode
CODEX_ROOT=$HOME_T/.codex
OPENCODE_AGENTS=$OPENCODE_ROOT/agents
OPENCODE_AGENTS_FILE=$OPENCODE_ROOT/AGENTS.md
CODEX_AGENTS_FILE=$CODEX_ROOT/AGENTS.md
SKILLS_DIR=$HOME_T/.agents/skills

mkdir -p "$OPENCODE_ROOT" "$CODEX_ROOT" "$SKILLS_DIR/unrelated" "$SKILLS_DIR/karl-fake"

OPENCODE_ORIG=$WORK_T/opencode-original.md
CODEX_ORIG=$WORK_T/codex-original.md
SKILL_ORIG=$WORK_T/skill-original.md

printf '%s\n' '# Existing OpenCode rules' '' 'Keep OpenCode content.' > "$OPENCODE_ORIG"
printf '%s\n' '# Existing Codex rules' '' 'Keep Codex content.' > "$CODEX_ORIG"
printf '%s\n' '---' 'name: unrelated' '---' '' 'Unrelated skill content.' > "$SKILL_ORIG"

cp "$OPENCODE_ORIG" "$OPENCODE_AGENTS_FILE"
cp "$CODEX_ORIG" "$CODEX_AGENTS_FILE"
cp "$SKILL_ORIG" "$SKILLS_DIR/unrelated/SKILL.md"
printf '%s\n' '---' 'name: karl-fake' '---' '' 'Fake karl skill.' > "$SKILLS_DIR/karl-fake/SKILL.md"

# --- 1. Install --------------------------------------------------------------

run_install

assert_block "$OPENCODE_AGENTS_FILE" "$WORK_T/block-lf.txt" install
assert_block "$CODEX_AGENTS_FILE" "$WORK_T/block-lf.txt" install

extract_prefix "$OPENCODE_AGENTS_FILE" "$WORK_T/prefix.txt"
{ cat "$OPENCODE_ORIG"; printf '\n'; } > "$WORK_T/prefix-expected.txt"
cmp -s "$WORK_T/prefix.txt" "$WORK_T/prefix-expected.txt" \
    || fail "Pre-existing OpenCode content was not preserved byte-exact around the managed block."
extract_prefix "$CODEX_AGENTS_FILE" "$WORK_T/prefix.txt"
{ cat "$CODEX_ORIG"; printf '\n'; } > "$WORK_T/prefix-expected.txt"
cmp -s "$WORK_T/prefix.txt" "$WORK_T/prefix-expected.txt" \
    || fail "Pre-existing Codex content was not preserved byte-exact around the managed block."

check_opencode_links
[ -d "$HOME_T/.agents-backup" ] || fail 'Backups were not rooted under the target home.'
printf 'PASS: install writes one canonical managed block per AGENTS.md and links the OpenCode agents.\n'

# --- 2. Idempotence ----------------------------------------------------------

fingerprint "$HOME_T" > "$WORK_T/fp-installed.txt"
backups_before=$(count_backups)
run_install
fingerprint "$HOME_T" > "$WORK_T/fp-reinstalled.txt"
cmp -s "$WORK_T/fp-installed.txt" "$WORK_T/fp-reinstalled.txt" \
    || fail 'A repeated install changed managed state.'
backups_after=$(count_backups)
assert_eq "$backups_before" "$backups_after" 'A repeated install created another backup.'
printf 'PASS: repeated install is a no-op.\n'

# --- 3. Uninstall dry-run ----------------------------------------------------

run_install --uninstall --dry-run
fingerprint "$HOME_T" > "$WORK_T/fp-dryrun.txt"
cmp -s "$WORK_T/fp-installed.txt" "$WORK_T/fp-dryrun.txt" \
    || fail 'Uninstall dry-run mutated the test home.'
printf 'PASS: uninstall dry-run does not mutate the home.\n'

# --- 4. Uninstall ------------------------------------------------------------

run_install --uninstall

for _name in karl-orchestrator.md karl-worker.md karl-reviewer.md; do
    _link=$OPENCODE_AGENTS/$_name
    if [ -e "$_link" ] || [ -L "$_link" ]; then
        fail "Managed link '$_link' survived uninstall."
    fi
done
assert_no_marker "$OPENCODE_AGENTS_FILE" uninstall
assert_no_marker "$CODEX_AGENTS_FILE" uninstall
cmp -s "$OPENCODE_AGENTS_FILE" "$OPENCODE_ORIG" \
    || fail 'Existing OpenCode content was not restored byte-exact after uninstall.'
cmp -s "$CODEX_AGENTS_FILE" "$CODEX_ORIG" \
    || fail 'Existing Codex content was not restored byte-exact after uninstall.'
cmp -s "$SKILLS_DIR/unrelated/SKILL.md" "$SKILL_ORIG" \
    || fail 'The unrelated skill changed during uninstall.'
if [ -e "$SKILLS_DIR/karl-fake" ] || [ -L "$SKILLS_DIR/karl-fake" ]; then
    fail "karl-* skill '$SKILLS_DIR/karl-fake' survived uninstall."
fi

run_install --uninstall
fingerprint "$HOME_T" > "$WORK_T/fp-uninstalled.txt"
run_install --uninstall
fingerprint "$HOME_T" > "$WORK_T/fp-uninstalled2.txt"
cmp -s "$WORK_T/fp-uninstalled.txt" "$WORK_T/fp-uninstalled2.txt" \
    || fail 'A repeated uninstall changed managed state.'
printf 'PASS: uninstall restores original content and removes owned links and karl-* skills.\n'

# --- 5. Stale-link regression ------------------------------------------------

STALE_LINK=$OPENCODE_AGENTS/karl-stale.md
ln -s "$REPO_ROOT/harnesses/opencode/agents/karl-stale-source.md" "$STALE_LINK"

run_install

if [ -e "$STALE_LINK" ] || [ -L "$STALE_LINK" ]; then
    fail "Stale owned link '$STALE_LINK' survived install."
fi
check_opencode_links
printf 'PASS: stale owned links pointing at missing sources are removed on install.\n'

# --- 6. Refusal and --force --------------------------------------------------

WORKER_REGULAR=$WORK_T/worker-regular.txt
WORKER_LINK=$OPENCODE_AGENTS/karl-worker.md
printf '%s\n' 'not a symlink' > "$WORKER_REGULAR"
rm -f "$WORKER_LINK"
cp "$WORKER_REGULAR" "$WORKER_LINK"

if run_install >/dev/null 2>&1; then
    fail "install.sh exited 0 despite a regular file at '$WORKER_LINK' and no --force."
fi
cmp -s "$WORKER_LINK" "$WORKER_REGULAR" \
    || fail 'The refused install modified the regular file at the link destination.'
if [ -L "$WORKER_LINK" ]; then
    fail 'The refused install replaced the regular file with a symlink anyway.'
fi

run_install --force

[ -L "$WORKER_LINK" ] || fail "--force did not replace '$WORKER_LINK' with a symlink."
assert_eq "$REPO_ROOT/harnesses/opencode/agents/karl-worker.md" \
    "$(resolve_link "$WORKER_LINK")" "Wrong link target for '$WORKER_LINK' after --force."

# The glob lists backup dirs in ascending timestamp order; keep the LAST
# match so the newest --force run's backup is checked (older runs may have
# backed up the same destination with different content).
FOUND_BACKUP=''
for _backup_dir in "$HOME_T/.agents-backup"/*; do
    if [ -f "$_backup_dir/opencode/agents/karl-worker.md" ]; then
        FOUND_BACKUP=$_backup_dir
    fi
done
[ -n "$FOUND_BACKUP" ] || fail '--force did not back up the replaced regular file.'
cmp -s "$FOUND_BACKUP/opencode/agents/karl-worker.md" "$WORKER_REGULAR" \
    || fail 'The --force backup of the regular file does not match its content.'
printf 'PASS: install refuses a regular file without --force and backs it up before replacing with --force.\n'

# --- 7. Wrong-target symlink: refusal and --force -----------------------------

WRONG_TARGET_FILE=$WORK_T/wrong-target.md
printf '%s\n' 'wrong target content' > "$WRONG_TARGET_FILE"
WRONG_TARGET_CANON=$(CDPATH='' cd "$(dirname -- "$WRONG_TARGET_FILE")" && pwd -P)/wrong-target.md
rm -f "$WORKER_LINK"
ln -s "$WRONG_TARGET_FILE" "$WORKER_LINK"

backups_refusal_before=$(count_backups)
if run_install >/dev/null 2>&1; then
    fail "install.sh exited 0 despite a wrong-target symlink at '$WORKER_LINK' and no --force."
fi
[ -L "$WORKER_LINK" ] || fail 'The refused install removed the wrong-target symlink.'
assert_eq "$WRONG_TARGET_CANON" "$(resolve_link "$WORKER_LINK")" \
    "The refused install changed the wrong-target symlink at '$WORKER_LINK'."
backups_refusal_after=$(count_backups)
assert_eq "$backups_refusal_before" "$backups_refusal_after" 'The refused install created a backup.'

run_install --force

[ -L "$WORKER_LINK" ] || fail "--force did not replace '$WORKER_LINK' with a symlink."
assert_eq "$REPO_ROOT/harnesses/opencode/agents/karl-worker.md" \
    "$(resolve_link "$WORKER_LINK")" "Wrong link target for '$WORKER_LINK' after --force."
FOUND_BACKUP=''
for _backup_dir in "$HOME_T/.agents-backup"/*; do
    if [ -e "$_backup_dir/opencode/agents/karl-worker.md" ]; then
        FOUND_BACKUP=$_backup_dir
    fi
done
[ -n "$FOUND_BACKUP" ] || fail '--force did not back up the wrong-target symlink.'
cmp -s "$FOUND_BACKUP/opencode/agents/karl-worker.md" "$WRONG_TARGET_FILE" \
    || fail 'The --force backup of the wrong-target symlink does not preserve its content.'
printf 'PASS: install refuses a wrong-target symlink without --force and backs it up before replacing with --force.\n'

# --- 8. Directory destination: refusal and --force -----------------------------

# A directory (with content) at the karl-worker.md destination: install must
# refuse it without --force, and --force must back it up with its content and
# replace it with a symlink (install.ps1 Remove-Item -Force parity).
WORKER_DIR=$OPENCODE_AGENTS/karl-worker.md
DIR_INNER=$WORK_T/worker-dir-inner.txt
printf '%s\n' 'inside the directory' > "$DIR_INNER"
rm -f "$WORKER_LINK"
mkdir -p "$WORKER_DIR"
cp "$DIR_INNER" "$WORKER_DIR/inner.txt"

backups_dir_before=$(count_backups)
if run_install >/dev/null 2>&1; then
    fail "install.sh exited 0 despite a directory at '$WORKER_DIR' and no --force."
fi
[ -d "$WORKER_DIR" ] || fail 'The refused install removed the directory at the link destination.'
if [ -L "$WORKER_DIR" ]; then
    fail 'The refused install replaced the directory with a symlink anyway.'
fi
cmp -s "$WORKER_DIR/inner.txt" "$DIR_INNER" \
    || fail 'The refused install modified the content inside the directory.'
backups_dir_after=$(count_backups)
assert_eq "$backups_dir_before" "$backups_dir_after" 'The refused install created a backup.'

run_install --force

[ -L "$WORKER_DIR" ] || fail "--force did not replace the directory '$WORKER_DIR' with a symlink."
assert_eq "$REPO_ROOT/harnesses/opencode/agents/karl-worker.md" \
    "$(resolve_link "$WORKER_DIR")" "Wrong link target for '$WORKER_DIR' after --force."

FOUND_BACKUP=''
for _backup_dir in "$HOME_T/.agents-backup"/*; do
    if [ -d "$_backup_dir/opencode/agents/karl-worker.md" ]; then
        FOUND_BACKUP=$_backup_dir
    fi
done
[ -n "$FOUND_BACKUP" ] || fail '--force did not back up the replaced directory.'
cmp -s "$FOUND_BACKUP/opencode/agents/karl-worker.md/inner.txt" "$DIR_INNER" \
    || fail 'The --force backup of the directory does not preserve its inner file.'
check_opencode_links
printf 'PASS: install refuses a directory without --force and backs it up (content preserved) before replacing it with a symlink via --force.\n'

# --- 9. CRLF target ----------------------------------------------------------

CRLF_ORIG=$WORK_T/codex-crlf-original.md
printf '%s\n' '# Codex CRLF rules' '' 'Keep CRLF.' | awk '{ printf "%s\r\n", $0 }' > "$CRLF_ORIG"
cp "$CRLF_ORIG" "$CODEX_AGENTS_FILE"

run_install

assert_block "$CODEX_AGENTS_FILE" "$WORK_T/block-crlf.txt" 'CRLF install'
awk '!/\r$/ { bad = 1 } END { if (bad) exit 1 }' "$WORK_T/block-installed.txt" \
    || fail 'Installed managed block lines in the CRLF target do not all carry CR.'
extract_prefix "$CODEX_AGENTS_FILE" "$WORK_T/prefix.txt"
{ cat "$CRLF_ORIG"; printf '\r\n'; } > "$WORK_T/prefix-expected.txt"
cmp -s "$WORK_T/prefix.txt" "$WORK_T/prefix-expected.txt" \
    || fail "Kept CRLF lines in '$CODEX_AGENTS_FILE' were not preserved byte-exact."
printf 'PASS: CRLF targets keep their newline convention.\n'

# --- 10. Multiple managed blocks -----------------------------------------------

MULTI_ORIG=$WORK_T/multi-original.md
{
    printf '%s\n' 'pre'
    printf '%s\n' "$MARKER_START" 'stale one' "$MARKER_END"
    printf '%s\n' 'middle-preserve'
    printf '%s\n' "$MARKER_START" 'stale two' "$MARKER_END"
    printf '%s\n' 'post'
} > "$MULTI_ORIG"
cp "$MULTI_ORIG" "$CODEX_AGENTS_FILE"

run_install

_multi_start=$(LC_ALL=C grep -cF -- "$MARKER_START" "$CODEX_AGENTS_FILE" || true)
assert_eq 2 "$_multi_start" \
    "Expected exactly two managed blocks in '$CODEX_AGENTS_FILE' (multi-block install), found $_multi_start."
{
    printf '%s\n' 'pre'
    cat "$WORK_T/block-lf.txt"
    printf '%s\n' 'middle-preserve'
    cat "$WORK_T/block-lf.txt"
    printf '%s\n' 'post'
} > "$WORK_T/multi-expected.txt"
cmp -s "$CODEX_AGENTS_FILE" "$WORK_T/multi-expected.txt" \
    || fail 'Multiple managed blocks were not each replaced with one canonical block while preserving content between pairs.'

run_install --uninstall

assert_no_marker "$CODEX_AGENTS_FILE" 'multi-block uninstall'
{
    printf '%s\n' 'pre' '' 'middle-preserve' '' 'post'
} > "$WORK_T/multi-uninstalled-expected.txt"
cmp -s "$CODEX_AGENTS_FILE" "$WORK_T/multi-uninstalled-expected.txt" \
    || fail 'Uninstall of multiple managed blocks did not preserve the content between pairs.'
printf 'PASS: each managed block pair is replaced/removed independently and inter-block content survives byte-exact.\n'

# --- 11. No final newline idempotence -----------------------------------------

NONL_ORIG=$WORK_T/nonl-original.md
{
    printf '%s\n' 'pre' "$MARKER_START" 'stale'
    printf '%s' "$MARKER_END"
} > "$NONL_ORIG"
cp "$NONL_ORIG" "$CODEX_AGENTS_FILE"

run_install

{
    printf '%s\n' 'pre'
    printf '%s' "$(cat "$WORK_T/block-lf.txt")"
} > "$WORK_T/nonl-expected.txt"
cmp -s "$CODEX_AGENTS_FILE" "$WORK_T/nonl-expected.txt" \
    || fail 'Install did not preserve the missing final newline of a file ending at the end marker.'
assert_block "$CODEX_AGENTS_FILE" "$WORK_T/block-lf.txt" 'no-final-newline install'

run_install

cmp -s "$CODEX_AGENTS_FILE" "$WORK_T/nonl-expected.txt" \
    || fail 'A repeated install on a file without a final newline was not byte-identical.'
# Last byte must not be LF: a non-empty substitution of `tail -c 1` means the
# file does not end with a newline (command substitution strips only LFs).
if [ -z "$(tail -c 1 -- "$CODEX_AGENTS_FILE")" ]; then
    fail 'Install added a trailing newline to a file that ended without one.'
fi
printf 'PASS: files without a final newline stay byte-identical across installs.\n'

printf 'PASS: install.sh lifecycle is isolated, idempotent, and non-destructive.\n'
