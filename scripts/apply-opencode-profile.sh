#!/bin/sh
# apply-opencode-profile.sh - Apply a karl-ai agent model profile to an
# opencode.json file, per project (default) or global.
#
# Mirrors scripts/apply-opencode-profile.ps1. Only writes
# agent.<karl-*>.model keys; every other key is preserved verbatim in value
# (key order included). Requires python3 for the JSON merge.
#
# POSIX sh only (dash, macOS sh): no arrays, no [[ ]], no readlink -f,
# no process substitution, no $'...' quoting.

set -eu

PROFILE='karl-default'
SCOPE='project'
TARGET_DIR=''
HOME_DIR=${HOME:-}
DRY_RUN=0
FORCE=0
LIST=0

usage() {
    cat <<'EOF'
Usage: apply-opencode-profile.sh [options]

Options:
       --profile <name-or-path>  Profile from profiles/<name>.json, or a direct
                                 path to a JSON file. Default: karl-default.
       --scope <project|global>  Where to write opencode.json. Default: project.
       --target-dir <dir>        Project directory (project scope only).
                                 Default: current directory.
       --home <dir>              Home directory (global scope only).
                                 Default: $HOME.
   -n, --dry-run                 Print planned changes without modifying anything.
   -f, --force                   Skip the timestamped backup of opencode.json.
       --list                    List available profiles and exit.
   -h, --help                    Show this help and exit.
EOF
}

die() {
    printf 'apply-opencode-profile.sh: %s\n' "$1" >&2
    exit 1
}

log_action() {
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '[DRY RUN] %s\n' "$1"
    else
        printf '[OK] %s\n' "$1"
    fi
}

# abs_path PATH — same approach as install.sh: no readlink -f (macOS lacks it).
abs_path() {
    _ap_dir=$(dirname -- "$1") || return 1
    _ap_base=$(basename -- "$1") || return 1
    _ap_parent=$(CDPATH='' cd "$_ap_dir" 2>/dev/null && pwd -P) || return 1
    printf '%s/%s\n' "$_ap_parent" "$_ap_base"
}

while [ $# -gt 0 ]; do
    case $1 in
        --profile)
            if [ $# -lt 2 ] || [ -z "$2" ]; then
                printf 'apply-opencode-profile.sh: --profile requires an argument.\n' >&2
                usage >&2
                exit 2
            fi
            PROFILE=$2
            shift
            ;;
        --scope)
            if [ $# -lt 2 ] || [ -z "$2" ]; then
                printf 'apply-opencode-profile.sh: --scope requires an argument.\n' >&2
                usage >&2
                exit 2
            fi
            SCOPE=$2
            shift
            ;;
        --target-dir)
            if [ $# -lt 2 ] || [ -z "$2" ]; then
                printf 'apply-opencode-profile.sh: --target-dir requires a directory argument.\n' >&2
                usage >&2
                exit 2
            fi
            TARGET_DIR=$2
            shift
            ;;
        --home)
            if [ $# -lt 2 ] || [ -z "$2" ]; then
                printf 'apply-opencode-profile.sh: --home requires a directory argument.\n' >&2
                usage >&2
                exit 2
            fi
            HOME_DIR=$2
            shift
            ;;
        -n|--dry-run)
            DRY_RUN=1
            ;;
        -f|--force)
            FORCE=1
            ;;
        --list)
            LIST=1
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            printf 'apply-opencode-profile.sh: unknown option: %s\n' "$1" >&2
            usage >&2
            exit 2
            ;;
    esac
    shift
done

case $SCOPE in
    project|global) ;;
    *)
        printf 'apply-opencode-profile.sh: --scope must be project or global.\n' >&2
        usage >&2
        exit 2
        ;;
esac

# Repository root: this script lives in <root>/scripts, so the root is the
# parent of the script directory. Works from any working directory.
SCRIPT_DIR=''
if [ -n "$0" ] && [ "${0#-}" = "$0" ] && [ -f "$0" ]; then
    SCRIPT_DIR=$(CDPATH='' cd "$(dirname -- "$0")" 2>/dev/null && pwd -P) || SCRIPT_DIR=''
fi
REPO_ROOT=''
if [ -n "$SCRIPT_DIR" ]; then
    REPO_ROOT=$(CDPATH='' cd "$SCRIPT_DIR/.." 2>/dev/null && pwd -P) || REPO_ROOT=''
fi
[ -n "$REPO_ROOT" ] && [ -d "$REPO_ROOT/profiles" ] \
    || die 'Cannot locate the profiles directory (run from a checkout).'

if [ "$LIST" -eq 1 ]; then
    for _f in "$REPO_ROOT"/profiles/*.json; do
        [ -e "$_f" ] || continue
        basename -- "$_f" .json
    done
    exit 0
fi

case $PROFILE in
    *[!A-Za-z0-9_.+-]*|*.json)
        PROFILE_PATH=$PROFILE
        ;;
    *)
        PROFILE_PATH=$REPO_ROOT/profiles/$PROFILE.json
        ;;
esac
[ -f "$PROFILE_PATH" ] || {
    printf 'apply-opencode-profile.sh: unknown profile: %s\n' "$PROFILE" >&2
    printf 'Available profiles:\n' >&2
    for _f in "$REPO_ROOT"/profiles/*.json; do
        [ -e "$_f" ] || continue
        printf '  %s\n' "$(basename -- "$_f" .json)" >&2
    done
    exit 2
}

if [ "$SCOPE" = 'global' ]; then
    [ -n "$HOME_DIR" ] || die 'Could not determine the home directory. Pass --home <dir>.'
    DEST=$HOME_DIR/.config/opencode/opencode.json
else
    if [ -z "$TARGET_DIR" ]; then
        TARGET_DIR=$(pwd -P)
    fi
    DEST=$TARGET_DIR/opencode.json
fi

command -v python3 >/dev/null 2>&1 \
    || die 'python3 is required for the JSON merge but was not found on PATH.'

TMPDIR_BASE=${TMPDIR:-/tmp}
WORK=$(mktemp -d "$TMPDIR_BASE/karl-profile-XXXXXX") || die 'Cannot create a temporary directory.'
trap 'rm -rf "$WORK"' EXIT INT TERM HUP
MERGED=$WORK/merged.json
REPORT=$WORK/report.txt

# The merge itself: validate the profile, validate the destination, and write
# the merged document plus a tab-separated change report. Only
# agent.<karl-*>.model keys are touched; order of existing keys is preserved.
python3 - "$PROFILE_PATH" "$DEST" "$MERGED" "$REPORT" <<'PYEOF'
import json
import sys

profile_path, dest_path, merged_path, report_path = sys.argv[1:5]

with open(profile_path, encoding='utf-8') as fh:
    try:
        profile = json.load(fh)
    except json.JSONDecodeError as exc:
        sys.exit('profile is not valid JSON: %s' % exc)

if not isinstance(profile, dict) or not profile:
    sys.exit('profile must be a non-empty JSON object')

for name, model in profile.items():
    if not name.startswith('karl-'):
        sys.exit("profile must only define 'karl-*' agents (found '%s')" % name)
    if not isinstance(model, str) or '/' not in model or any(c.isspace() for c in model):
        sys.exit("profile entry '%s' must be '<provider>/<model>'" % name)

try:
    with open(dest_path, encoding='utf-8') as fh:
        config = json.load(fh)
except FileNotFoundError:
    config = {'$schema': 'https://opencode.ai/config.json'}
except json.JSONDecodeError as exc:
    sys.exit('destination is not valid JSON: %s' % exc)

if not isinstance(config, dict):
    sys.exit('destination must hold a JSON object')

agent = config.get('agent')
if agent is None:
    agent = {}
    config['agent'] = agent
if not isinstance(agent, dict):
    sys.exit("destination has a non-object 'agent' key; refusing to merge")

changes = []
for name in sorted(profile):
    wanted = profile[name]
    entry = agent.get(name)
    if entry is None:
        agent[name] = {'model': wanted}
        changes.append((name, '(absent)', wanted))
    else:
        if not isinstance(entry, dict):
            sys.exit("destination has a non-object 'agent.%s' entry; refusing to merge" % name)
        before = entry.get('model')
        if before != wanted:
            entry['model'] = wanted
            changes.append((name, before if before is not None else '(absent)', wanted))

with open(merged_path, 'w', encoding='utf-8') as fh:
    json.dump(config, fh, indent=2)
    fh.write('\n')

with open(report_path, 'w', encoding='utf-8') as fh:
    for name, before, after in changes:
        fh.write('%s\t%s\t%s\n' % (name, before, after))
PYEOF

if [ ! -s "$REPORT" ]; then
    _count=$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))))' "$PROFILE_PATH")
    log_action "Profile '$PROFILE' already applied to '$DEST' ($_count models current)."
    exit 0
fi

while read -r _name _before _after; do
    [ -n "$_name" ] || continue
    printf '%s: %s -> %s\n' "$_name" "$_before" "$_after"
done < "$REPORT"
_CHANGE_COUNT=$(wc -l < "$REPORT" | tr -d ' ')

if [ "$DRY_RUN" -eq 1 ]; then
    log_action "Profile '$PROFILE' would apply $_CHANGE_COUNT models to '$DEST'."
    exit 0
fi

_DEST_DIR=$(dirname -- "$DEST")
[ -d "$_DEST_DIR" ] || mkdir -p "$_DEST_DIR"

if [ -f "$DEST" ] && [ "$FORCE" -eq 0 ]; then
    _stamp=$(date +%Y%m%d-%H%M%S)
    _backup=$DEST.bak-$_stamp
    _suffix=2
    while [ -e "$_backup" ]; do
        _backup=$DEST.bak-$_stamp-$_suffix
        _suffix=$((_suffix + 1))
    done
    cp "$DEST" "$_backup"
    log_action "Back up $DEST -> $_backup"
fi

mv "$MERGED" "$DEST"
trap - EXIT INT TERM HUP
rm -rf "$WORK"
log_action "Apply profile '$PROFILE' ($_CHANGE_COUNT models) to '$DEST'."
