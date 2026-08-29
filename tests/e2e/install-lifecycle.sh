#!/bin/sh
# install-lifecycle.sh - POSIX sh E2E test mirroring tests/e2e/install-lifecycle.ps1.
#
# Exercises install.sh (install and uninstall paths) against an isolated
# temporary home directory, plus the --repo bootstrap-clone mode against a
# fixture repository. POSIX sh only; needs Debian base utils plus git.
# OpenCode link assertions only; Codex links are not asserted but their
# presence does not break the test.

set -eu

REPO_ROOT=$(CDPATH='' cd "$(dirname -- "$0")/../.." && pwd -P) || exit 1
INSTALLER=$REPO_ROOT/install.sh
MARKER_START='<!-- karl-ai: controlled-development -->'
MARKER_END='<!-- /karl-ai: controlled-development -->'

TMPBASE=${TMPDIR:-/tmp}
HOME_T=$(mktemp -d "$TMPBASE/karl-install-e2e-home.XXXXXX")
HOME_T2=''
HOME_T3=''
WORK_T=$(mktemp -d "$TMPBASE/karl-install-e2e-work.XXXXXX")

cleanup() {
    if [ -n "${HOME_T:-}" ]; then rm -rf -- "$HOME_T"; fi
    if [ -n "${HOME_T2:-}" ]; then rm -rf -- "$HOME_T2"; fi
    if [ -n "${HOME_T3:-}" ]; then rm -rf -- "$HOME_T3"; fi
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
    # count_backups ROOT: number of backup dirs under ROOT/.agents-backup.
    _cb_root=$1
    _cb_count=0
    if [ -d "$_cb_root/.agents-backup" ]; then
        for _cb_dir in "$_cb_root/.agents-backup"/*; do
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
backups_before=$(count_backups "$HOME_T")
run_install
fingerprint "$HOME_T" > "$WORK_T/fp-reinstalled.txt"
cmp -s "$WORK_T/fp-installed.txt" "$WORK_T/fp-reinstalled.txt" \
    || fail 'A repeated install changed managed state.'
backups_after=$(count_backups "$HOME_T")
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

backups_refusal_before=$(count_backups "$HOME_T")
if run_install >/dev/null 2>&1; then
    fail "install.sh exited 0 despite a wrong-target symlink at '$WORKER_LINK' and no --force."
fi
[ -L "$WORKER_LINK" ] || fail 'The refused install removed the wrong-target symlink.'
assert_eq "$WRONG_TARGET_CANON" "$(resolve_link "$WORKER_LINK")" \
    "The refused install changed the wrong-target symlink at '$WORKER_LINK'."
backups_refusal_after=$(count_backups "$HOME_T")
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

backups_dir_before=$(count_backups "$HOME_T")
if run_install >/dev/null 2>&1; then
    fail "install.sh exited 0 despite a directory at '$WORKER_DIR' and no --force."
fi
[ -d "$WORKER_DIR" ] || fail 'The refused install removed the directory at the link destination.'
if [ -L "$WORKER_DIR" ]; then
    fail 'The refused install replaced the directory with a symlink anyway.'
fi
cmp -s "$WORKER_DIR/inner.txt" "$DIR_INNER" \
    || fail 'The refused install modified the content inside the directory.'
backups_dir_after=$(count_backups "$HOME_T")
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

# --- 12. --repo bootstrap clone -----------------------------------------------

# Fixture repository: a minimal copy of the real repo (AGENTS.md with the
# managed block, skills/, harnesses/) committed to a local git repo. The
# installer is run from the REAL repo but installs from this fixture clone.
FIXTURE_REPO=$WORK_T/fixture-repo
mkdir -p "$FIXTURE_REPO"
cp "$REPO_ROOT/AGENTS.md" "$FIXTURE_REPO/AGENTS.md"
cp -R "$REPO_ROOT/skills" "$FIXTURE_REPO/skills"
cp -R "$REPO_ROOT/harnesses" "$FIXTURE_REPO/harnesses"
git init -q "$FIXTURE_REPO"
git -C "$FIXTURE_REPO" config user.name 'Karl E2E'
git -C "$FIXTURE_REPO" config user.email 'karl-e2e@example.invalid'
git -C "$FIXTURE_REPO" add -A
git -C "$FIXTURE_REPO" commit -q -m 'fixture'
FIXTURE_ABS=$(CDPATH='' cd "$FIXTURE_REPO" && pwd -P)

strip_git_suffix() {
    _sgs=$1
    while :; do
        case $_sgs in
            /) break ;;
            */) _sgs=${_sgs%/} ;;
            *) break ;;
        esac
    done
    case $_sgs in
        *.git) _sgs=${_sgs%.git} ;;
    esac
    printf '%s\n' "$_sgs"
}

# A second, isolated target home: its .agents slot must be free so the clone
# can be created there.
HOME_T2=$(mktemp -d "$TMPBASE/karl-install-e2e-repohome.XXXXXX")
OPENCODE_ROOT2=$HOME_T2/.config/opencode
CODEX_ROOT2=$HOME_T2/.codex
mkdir -p "$OPENCODE_ROOT2" "$CODEX_ROOT2"
printf '%s\n' '# Existing OpenCode rules' '' 'Keep OpenCode content.' > "$OPENCODE_ROOT2/AGENTS.md"
printf '%s\n' '# Existing Codex rules' '' 'Keep Codex content.' > "$CODEX_ROOT2/AGENTS.md"

run_repo_install() {
    sh "$INSTALLER" --repo "$FIXTURE_REPO" --target-home "$HOME_T2" "$@"
}

CLONE_DIR=$HOME_T2/.agents

run_repo_install

[ -d "$CLONE_DIR" ] || fail "--repo did not clone the fixture into '$CLONE_DIR'."
[ -f "$CLONE_DIR/AGENTS.md" ] || fail "The clone at '$CLONE_DIR' is missing AGENTS.md."
[ -f "$CLONE_DIR/skills/karl-orchestrate/SKILL.md" ] \
    || fail "The clone at '$CLONE_DIR' is missing the Karl skills."
CLONE_ORIGIN=$(git -C "$CLONE_DIR" remote get-url origin) \
    || fail "The clone at '$CLONE_DIR' has no origin remote."
assert_eq "$(strip_git_suffix "$FIXTURE_ABS")" "$(strip_git_suffix "$CLONE_ORIGIN")" \
    "Clone origin at '$CLONE_DIR' does not match the fixture repository."

assert_block "$OPENCODE_ROOT2/AGENTS.md" "$WORK_T/block-lf.txt" 'repo install'
assert_block "$CODEX_ROOT2/AGENTS.md" "$WORK_T/block-lf.txt" 'repo install'

for _name in karl-orchestrator.md karl-worker.md karl-reviewer.md; do
    _link=$OPENCODE_ROOT2/agents/$_name
    [ -L "$_link" ] || fail "--repo install did not create the symlink '$_link'."
    assert_eq "$CLONE_DIR/harnesses/opencode/agents/$_name" "$(resolve_link "$_link")" \
        "Wrong link target for '$_link': expected a link into the clone dir."
done
printf 'PASS: --repo clones the fixture into the target home and installs from the clone.\n'

# Second run with the same flag: pull (no-op) plus already-current state and
# no new backup.
_repo_out=$(run_repo_install 2>&1) || fail 'A repeated --repo install failed.'
case $_repo_out in
    *"Managed block already current"*) ;;
    *) fail 'A repeated --repo install did not report an already-current managed block.' ;;
esac
_repo_backups_before=$(count_backups "$HOME_T2")
run_repo_install >/dev/null 2>&1 || fail 'A repeated --repo install failed.'
_repo_backups_after=$(count_backups "$HOME_T2")
assert_eq "$_repo_backups_before" "$_repo_backups_after" \
    'A repeated --repo install created another backup.'
printf 'PASS: a repeated --repo install updates the clone and is a no-op.\n'

# Mismatch refusal: .agents is a git repo but not a clone of the fixture URL.
MISMATCH_FILE=$CLONE_DIR/local-only.txt
printf '%s\n' 'keep me' > "$MISMATCH_FILE"
git -C "$CLONE_DIR" remote remove origin
_mismatch_backups_before=$(count_backups "$HOME_T2")
if run_repo_install >/dev/null 2>&1; then
    fail "install.sh exited 0 despite a non-matching git repo at '$CLONE_DIR' and no --force."
fi
[ -f "$MISMATCH_FILE" ] || fail "The refused --repo install modified '$CLONE_DIR'."
git -C "$CLONE_DIR" rev-parse --git-dir >/dev/null 2>&1 \
    || fail "The refused --repo install damaged the existing git repo at '$CLONE_DIR'."
_mismatch_backups_after=$(count_backups "$HOME_T2")
assert_eq "$_mismatch_backups_before" "$_mismatch_backups_after" \
    'The refused --repo install created a backup.'

# --force backs the mismatched dir up and re-clones.
if ! run_repo_install --force >/dev/null 2>&1; then
    fail '--force re-clone failed.'
fi
[ -f "$CLONE_DIR/AGENTS.md" ] || fail "--force did not re-clone the fixture into '$CLONE_DIR'."
if [ -e "$MISMATCH_FILE" ] || [ -L "$MISMATCH_FILE" ]; then
    fail "--force re-clone kept the mismatching content at '$MISMATCH_FILE'."
fi
CLONE_ORIGIN2=$(git -C "$CLONE_DIR" remote get-url origin) \
    || fail "The re-cloned repo at '$CLONE_DIR' has no origin remote."
assert_eq "$(strip_git_suffix "$FIXTURE_ABS")" "$(strip_git_suffix "$CLONE_ORIGIN2")" \
    "Re-cloned origin at '$CLONE_DIR' does not match the fixture repository."
FOUND_FORCED_BACKUP=''
for _backup_dir in "$HOME_T2/.agents-backup"/*; do
    if [ -f "$_backup_dir/agents/local-only.txt" ]; then
        FOUND_FORCED_BACKUP=$_backup_dir
    fi
done
[ -n "$FOUND_FORCED_BACKUP" ] || fail '--force did not back up the mismatched clone dir.'
printf '%s\n' 'keep me' > "$WORK_T/mismatch-expected.txt"
cmp -s "$FOUND_FORCED_BACKUP/agents/local-only.txt" "$WORK_T/mismatch-expected.txt" \
    || fail 'The --force backup of the mismatched clone dir lost its content.'
printf 'PASS: --repo refuses a non-matching git repo without --force and re-clones with --force.\n'

# Bootstrap requirement: an installer copy outside any checkout must demand
# --repo and exit 2.
BARE_DIR=$WORK_T/bare
mkdir -p "$BARE_DIR"
cp "$INSTALLER" "$BARE_DIR/install.sh"
_bare_code=0
sh "$BARE_DIR/install.sh" --target-home "$HOME_T2" >/dev/null 2>&1 || _bare_code=$?
assert_eq 2 "$_bare_code" \
    "Expected exit 2 when --repo is missing outside a checkout, got $_bare_code."
printf 'PASS: install.sh outside a checkout requires --repo and exits 2 without it.\n'

# Bootstrap requirement: a copied installer sitting beside an AGENTS.md (no
# skills/ or harnesses/ directories) is NOT a checkout: it must also demand
# --repo and exit 2.
SEMI_BARE_DIR=$WORK_T/semi-bare
mkdir -p "$SEMI_BARE_DIR"
cp "$INSTALLER" "$SEMI_BARE_DIR/install.sh"
printf '%s\n' '# Stray AGENTS.md, not a checkout.' > "$SEMI_BARE_DIR/AGENTS.md"
_semi_bare_code=0
sh "$SEMI_BARE_DIR/install.sh" --target-home "$HOME_T2" >/dev/null 2>&1 || _semi_bare_code=$?
assert_eq 2 "$_semi_bare_code" \
    "Expected exit 2 for an installer copy beside a bare AGENTS.md without --repo, got $_semi_bare_code."
printf 'PASS: an installer copy beside a bare AGENTS.md requires --repo and exits 2 without it.\n'

# --- 13. stdin-equivalent bootstrap via sh -c ---------------------------------

# Run the installer with no script file context ($0="--" under `sh -c`), the
# same shape as the documented curl | sh bootstrap: it must clone the fixture
# and run the ENTIRE flow from the clone.
HOME_T3=$(mktemp -d "$TMPBASE/karl-install-e2e-stdin.XXXXXX")
OPENCODE_ROOT3=$HOME_T3/.config/opencode
CODEX_ROOT3=$HOME_T3/.codex
mkdir -p "$OPENCODE_ROOT3" "$CODEX_ROOT3"
printf '%s\n' '# Existing OpenCode rules' '' 'Keep OpenCode content.' > "$OPENCODE_ROOT3/AGENTS.md"
printf '%s\n' '# Existing Codex rules' '' 'Keep Codex content.' > "$CODEX_ROOT3/AGENTS.md"

CLONE_DIR3=$HOME_T3/.agents
sh -c "$(cat "$INSTALLER")" -- --repo "$FIXTURE_REPO" --target-home "$HOME_T3" >/dev/null 2>&1 \
    || fail 'The sh -c (stdin-equivalent) bootstrap install failed.'

[ -f "$CLONE_DIR3/AGENTS.md" ] || fail "The sh -c bootstrap did not clone the fixture into '$CLONE_DIR3'."
[ -f "$CLONE_DIR3/skills/karl-orchestrate/SKILL.md" ] \
    || fail "The sh -c bootstrap clone at '$CLONE_DIR3' is missing the Karl skills."
assert_block "$OPENCODE_ROOT3/AGENTS.md" "$WORK_T/block-lf.txt" 'sh -c bootstrap install'
assert_block "$CODEX_ROOT3/AGENTS.md" "$WORK_T/block-lf.txt" 'sh -c bootstrap install'
for _name in karl-orchestrator.md karl-worker.md karl-reviewer.md; do
    _link=$OPENCODE_ROOT3/agents/$_name
    [ -L "$_link" ] || fail "The sh -c bootstrap did not create the symlink '$_link'."
    assert_eq "$CLONE_DIR3/harnesses/opencode/agents/$_name" "$(resolve_link "$_link")" \
        "Wrong link target for '$_link': expected a link into the sh -c bootstrap clone dir."
done
printf 'PASS: sh -c (stdin-equivalent) bootstrap clones the fixture and installs from the clone.\n'

printf 'PASS: install.sh lifecycle is isolated, idempotent, and non-destructive.\n'
