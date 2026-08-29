#!/bin/sh
# install.sh - POSIX sh installer for the karl-ai controlled-development setup.
#
# Mirrors install.ps1 (install and uninstall paths).
#
# POSIX sh only (dash, macOS sh): no arrays, no [[ ]], no readlink -f,
# no process substitution, no $'...' quoting.

set -eu

MARKER_START='<!-- karl-ai: controlled-development -->'
MARKER_END='<!-- /karl-ai: controlled-development -->'

DRY_RUN=0
FORCE=0
UNINSTALL=0
TARGET_HOME=${HOME:-}

usage() {
    cat <<'EOF'
Usage: install.sh [options]

Options:
  -n, --dry-run            Print planned actions without modifying anything.
  -f, --force              Back up and replace existing regular files or
                           directories at link destinations.
      --uninstall          Remove the managed block, owned agent links, and
                           karl-* skills from the target home.
      --target-home <dir>  Install into <dir> instead of the current user's
                           home directory.
  -h, --help               Show this help and exit.
EOF
}

die() {
    printf 'install.sh: %s\n' "$1" >&2
    exit 1
}

warn() {
    printf 'install.sh: %s\n' "$1" >&2
}

log_action() {
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '[DRY RUN] %s\n' "$1"
    else
        printf '[OK] %s\n' "$1"
    fi
}

# abs_path PATH
# Print an absolute path for PATH: the canonical (pwd -P) directory of the
# parent joined with the basename. Never uses readlink -f (macOS lacks it).
abs_path() {
    _ap_dir=$(dirname -- "$1") || return 1
    _ap_base=$(basename -- "$1") || return 1
    _ap_parent=$(CDPATH='' cd "$_ap_dir" 2>/dev/null && pwd -P) || return 1
    printf '%s/%s\n' "$_ap_parent" "$_ap_base"
}

# backup_file SOURCE RELATIVE_BACKUP_PATH
# Copy SOURCE under $BACKUP_ROOT/$2. No-op when SOURCE does not exist.
# -e misses dangling symlinks, so -L is checked too; cp -R copies a symlink
# as a symlink (never dereferences), so a dangling link is preserved as-is.
backup_file() {
    [ -e "$1" ] || [ -L "$1" ] || return 0
    _bf_dest=$BACKUP_ROOT/$2
    mkdir -p "$(dirname -- "$_bf_dest")"
    cp -R "$1" "$_bf_dest"
}

# --- Argument parsing -------------------------------------------------------

while [ $# -gt 0 ]; do
    case $1 in
        -n|--dry-run)
            DRY_RUN=1
            ;;
        -f|--force)
            FORCE=1
            ;;
        --uninstall)
            UNINSTALL=1
            ;;
        --target-home)
            if [ $# -lt 2 ] || [ -z "$2" ]; then
                printf 'install.sh: --target-home requires a directory argument.\n' >&2
                usage >&2
                exit 2
            fi
            TARGET_HOME=$2
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            printf 'install.sh: unknown option: %s\n' "$1" >&2
            usage >&2
            exit 2
            ;;
    esac
    shift
done

[ -n "$TARGET_HOME" ] || die 'Could not determine the target home directory. Pass --target-home <dir>.'

# Strip trailing slashes (but keep a lone root slash).
while [ "$TARGET_HOME" != '/' ] && [ "${TARGET_HOME%/}" != "$TARGET_HOME" ]; do
    TARGET_HOME=${TARGET_HOME%/}
done

REPO_ROOT=$(CDPATH='' cd "$(dirname -- "$0")" && pwd -P)

# Timestamp is computed once per run; backup dirs are created lazily and
# only on real (non-dry-run) modifications.
BACKUP_ROOT=$TARGET_HOME/.agents-backup/$(date +%Y%m%d-%H%M%S)
# Two runs can start within the same second and resolve the same backup
# root. Suffix the root so each run that actually backs up gets its own
# directory; otherwise a later run's cp could not replace an earlier run's
# backup of a different type (file vs directory) without failing or
# destroying it.
_br_n=0
while [ -e "$BACKUP_ROOT" ] || [ -L "$BACKUP_ROOT" ]; do
    _br_n=$((_br_n + 1))
    BACKUP_ROOT=$TARGET_HOME/.agents-backup/$(date +%Y%m%d-%H%M%S)-$_br_n
done

TMP_DIR=$(mktemp -d 2>/dev/null) || die 'mktemp is unavailable or failed.'
trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM
BLOCK_FILE=$TMP_DIR/block.txt
NEW_FILE=$TMP_DIR/new.txt

if [ "$UNINSTALL" -eq 0 ] && [ "${OPENCODE_DISABLE_EXTERNAL_SKILLS:-}" = '1' ]; then
    warn 'OPENCODE_DISABLE_EXTERNAL_SKILLS=1 prevents OpenCode from discovering ~/.agents/skills. Remove that environment variable before starting OpenCode.'
fi

# --- Canonical managed block ------------------------------------------------

if [ "$UNINSTALL" -eq 0 ]; then
    CANONICAL=$REPO_ROOT/AGENTS.md
    [ -f "$CANONICAL" ] || die "Canonical AGENTS.md not found at '$CANONICAL'."

    MANAGED_BLOCK=$(awk -v sstart="$MARKER_START" -v send="$MARKER_END" '
        !found && index($0, sstart) { found = 1 }
        found {
            print
            if (index($0, send)) { ok = 1; exit }
        }
        END { if (!ok) exit 1 }
    ' "$CANONICAL") || die 'Canonical AGENTS.md does not contain the managed karl-ai block.'

    # Normalize the block to LF; CRLF targets get CR re-added per line.
    MANAGED_BLOCK=$(printf '%s\n' "$MANAGED_BLOCK" | tr -d '\r')
    printf '%s\n' "$MANAGED_BLOCK" > "$BLOCK_FILE"
fi

# --- Skill validation -------------------------------------------------------

validate_skills() {
    for skill_dir in "$REPO_ROOT"/skills/karl-*; do
        [ -d "$skill_dir" ] || continue
        vs_skill_file=$skill_dir/SKILL.md
        if [ ! -f "$vs_skill_file" ]; then
            die "Missing SKILL.md in '$skill_dir'."
        fi
        vs_dir_name=$(basename -- "$skill_dir")
        vs_name=$(sed -n 's/^name:[[:space:]]*//p' "$vs_skill_file" | head -n 1)
        vs_name=$(printf '%s\n' "$vs_name" | sed 's/[[:space:]]*$//')
        if [ -z "$vs_name" ] || [ "$vs_name" != "$vs_dir_name" ]; then
            die "Skill name in '$vs_skill_file' must match directory '$vs_dir_name'."
        fi
    done
    log_action 'Validated Karl skills'
}

# --- Managed block update ---------------------------------------------------

# update_managed_block TARGET_FILE RELATIVE_BACKUP_PATH
update_managed_block() {
    umb_target=$1
    umb_crlf=0
    if [ -f "$umb_target" ]; then
        umb_cr=$(printf '\r')
        if LC_ALL=C grep -qF -- "$umb_cr" "$umb_target"; then
            umb_crlf=1
        fi
    fi

    # Detect a missing trailing newline. Command substitution strips trailing
    # newlines, so substituting `tail -c 1` yields an empty string exactly
    # when the file's last byte is LF.
    umb_no_final_nl=0
    if [ -f "$umb_target" ] && [ -n "$(tail -c 1 -- "$umb_target" 2>/dev/null)" ]; then
        umb_no_final_nl=1
    fi

    if [ -f "$umb_target" ]; then
        umb_input=$umb_target
    else
        umb_input=/dev/null
    fi

    # Render the updated file. awk (multiline-safe) instead of sed:
    # - every managed block pair (start marker line .. end marker line) is
    #   replaced independently, mirroring install.ps1's non-greedy per-match
    #   regex replacement; content between two pairs is byte-preserved;
    # - otherwise a whitespace-only/missing file becomes just the block;
    # - otherwise the block is appended after one blank line, mirroring
    #   install.ps1 (existing.TrimEnd() + newline + newline + block + newline);
    # - in the replace path the file's trailing-byte state is preserved
    #   exactly (a file with no final newline stays without one).
    awk -v blkfile="$BLOCK_FILE" -v crlf="$umb_crlf" -v nonl="$umb_no_final_nl" \
        -v sstart="$MARKER_START" -v send="$MARKER_END" '
        BEGIN {
            while ((getline bline < blkfile) > 0) blk[nb++] = bline
            close(blkfile)
            cr = (crlf == "1") ? "\r" : ""
            inblk = 0
            npairs = 0
        }
        {
            srcline[NR] = $0
            if (!inblk && index($0, sstart)) { inblk = 1; pstart[++npairs] = NR }
            if (inblk && index($0, send)) { inblk = 0; pend[npairs] = NR; replaced = 1 }
        }
        END {
            if (nb == 0) { exit 1 }
            out = ""
            if (replaced) {
                i = 1
                while (i <= NR) {
                    hit = 0
                    for (p = 1; p <= npairs; p++) {
                        if (i == pstart[p]) {
                            for (j = 0; j < nb; j++) out = out blk[j] cr "\n"
                            i = pend[p] + 1
                            hit = 1
                            break
                        }
                    }
                    if (!hit) { out = out srcline[i] "\n"; i++ }
                }
                if (nonl == "1") sub(/\n$/, "", out)
            } else {
                last = 0
                for (i = NR; i >= 1; i--) {
                    if (srcline[i] !~ /^[ \t\r]*$/) { last = i; break }
                }
                if (last == 0) {
                    for (j = 0; j < nb; j++) out = out blk[j] cr "\n"
                } else {
                    for (i = 1; i <= last; i++) out = out srcline[i] "\n"
                    out = out cr "\n"
                    for (j = 0; j < nb; j++) out = out blk[j] cr "\n"
                }
            }
            printf "%s", out
        }
    ' "$umb_input" > "$NEW_FILE" || die "Failed to render the managed block for '$umb_target'."

    if [ -f "$umb_target" ] && cmp -s "$umb_target" "$NEW_FILE"; then
        log_action "Managed block already current in $umb_target"
        return 0
    fi

    if [ "$DRY_RUN" -eq 1 ]; then
        log_action "Install managed block in $umb_target"
        return 0
    fi

    backup_file "$umb_target" "$2"
    mkdir -p "$(dirname -- "$umb_target")"
    : >> "$umb_target"
    cat "$NEW_FILE" > "$umb_target"
    log_action "Install managed block in $umb_target"
}

# --- Agent link sync --------------------------------------------------------

# sync_agents HARNESS_NAME HARNESS_ROOT SOURCE_DIR EXTENSION
sync_agents() {
    sa_harness=$1
    sa_root=$2
    sa_srcdir=$3
    sa_ext=$4

    if [ ! -d "$sa_root" ]; then
        warn "Skipping $sa_harness because '$sa_root' does not exist."
        return 0
    fi

    sa_agents=$sa_root/agents
    if [ -e "$sa_agents" ] && [ ! -d "$sa_agents" ]; then
        die "Expected a directory at '$sa_agents'."
    fi
    if [ ! -d "$sa_agents" ]; then
        if [ "$DRY_RUN" -eq 1 ]; then
            log_action "Create directory $sa_agents"
        else
            mkdir -p "$sa_agents"
            log_action "Create directory $sa_agents"
        fi
    fi

    sa_srcdir_abs=$(abs_path "$sa_srcdir") || die "Cannot resolve source directory '$sa_srcdir'."

    # Remove stale owned links: symlinks that point into the repo source dir
    # but whose source file no longer exists.
    if [ -d "$sa_agents" ]; then
        for sa_entry in "$sa_agents"/*; do
            [ -L "$sa_entry" ] || continue
            sa_name=$(basename -- "$sa_entry")
            sa_cur=$(readlink "$sa_entry")
            case $sa_cur in
                /*) sa_cand=$sa_cur ;;
                *) sa_cand=$sa_agents/$sa_cur ;;
            esac
            sa_cand_abs=$(abs_path "$sa_cand" 2>/dev/null) || sa_cand_abs=''
            case $sa_cand_abs in
                "$sa_srcdir_abs"/*) ;;
                *) continue ;;
            esac
            sa_keep=0
            for sa_src in "$sa_srcdir"/*."$sa_ext"; do
                [ -f "$sa_src" ] || continue
                if [ "$(basename -- "$sa_src")" = "$sa_name" ]; then
                    sa_keep=1
                    break
                fi
            done
            if [ "$sa_keep" -eq 0 ]; then
                if [ "$DRY_RUN" -eq 1 ]; then
                    log_action "Remove stale managed link $sa_entry"
                else
                    rm -f "$sa_entry"
                    log_action "Remove stale managed link $sa_entry"
                fi
            fi
        done
    fi

    # Install or update per-file symlinks.
    for sa_src in "$sa_srcdir"/*."$sa_ext"; do
        [ -f "$sa_src" ] || continue
        sa_name=$(basename -- "$sa_src")
        sa_target=$sa_agents/$sa_name
        sa_src_abs=$(abs_path "$sa_src") || die "Cannot resolve source file '$sa_src'."

        if [ -L "$sa_target" ]; then
            sa_cur=$(readlink "$sa_target")
            case $sa_cur in
                /*) sa_cand=$sa_cur ;;
                *) sa_cand=$sa_agents/$sa_cur ;;
            esac
            sa_cand_abs=$(abs_path "$sa_cand" 2>/dev/null) || sa_cand_abs=''
            if [ -n "$sa_cand_abs" ] && [ "$sa_cand_abs" = "$sa_src_abs" ]; then
                log_action "Link already current: $sa_target"
                continue
            fi
        fi

        if [ -e "$sa_target" ] || [ -L "$sa_target" ]; then
            # Any existing destination that is not already the correct link is
            # refused without --force (install.ps1 parity); --force backs it up
            # and replaces it.
            if [ "$FORCE" -ne 1 ]; then
                die "Destination '$sa_target' already exists. Re-run with --force to back it up and replace it."
            fi
            if [ "$DRY_RUN" -eq 1 ]; then
                log_action "Link $sa_target -> $sa_src"
            else
                backup_file "$sa_target" "$sa_harness/agents/$sa_name"
                # rm -rf removes every destination type (install.ps1 parity:
                # Remove-Item -Force clears files and directories alike);
                # on a symlink it removes the link itself, never the target.
                rm -rf -- "$sa_target"
                ln -s "$sa_src" "$sa_target"
                log_action "Link $sa_target -> $sa_src"
            fi
            continue
        fi

        if [ "$DRY_RUN" -eq 1 ]; then
            log_action "Link $sa_target -> $sa_src"
        else
            ln -s "$sa_src" "$sa_target"
            log_action "Link $sa_target -> $sa_src"
        fi
    done
}

# --- Uninstall --------------------------------------------------------------

# uninstall_managed_block TARGET_FILE RELATIVE_BACKUP_PATH
# Remove the managed block, mirroring install.ps1: kept lines are byte-exact,
# then the result is TrimEnd()-ed and one newline of the file's convention is
# appended. A file left blank/whitespace-only is deleted.
uninstall_managed_block() {
    umb_target=$1
    if [ ! -f "$umb_target" ]; then
        log_action "Managed block already absent from $umb_target"
        return 0
    fi

    umb_crlf=0
    umb_cr=$(printf '\r')
    if LC_ALL=C grep -qF -- "$umb_cr" "$umb_target"; then
        umb_crlf=1
    fi

    umb_matched=0
    awk -v crlf="$umb_crlf" -v sstart="$MARKER_START" -v send="$MARKER_END" '
        BEGIN { cr = (crlf == "1") ? "\r" : ""; inblk = 0; npairs = 0; matched = 0 }
        {
            srcline[NR] = $0
            if (!inblk && index($0, sstart)) { inblk = 1; pstart[++npairs] = NR }
            if (inblk && index($0, send)) { inblk = 0; pend[npairs] = NR; matched = 1 }
        }
        END {
            if (!matched) exit 3
            # Remove every start..end pair independently (ps1 non-greedy
            # per-match parity); content between pairs is kept. The newline
            # that terminated each end-marker line is not part of the ps1
            # match and therefore survives removal (kept as a whitespace-only
            # line; the trailing trim below drops it when it lands at EOF).
            n = 0
            i = 1
            while (i <= NR) {
                hit = 0
                for (p = 1; p <= npairs; p++) {
                    if (i == pstart[p]) { i = pend[p] + 1; hit = 1; break }
                }
                if (hit) {
                    kept[++n] = cr
                } else {
                    kept[++n] = srcline[i]
                    i++
                }
            }
            last = 0
            for (i = n; i >= 1; i--) {
                s = kept[i]
                sub(/[ \t\r]+$/, "", s)
                if (s != "") { last = i; break }
            }
            for (i = 1; i <= last; i++) {
                if (i == last) {
                    s = kept[i]
                    sub(/[ \t\r]+$/, "", s)
                    print s cr
                } else {
                    print kept[i]
                }
            }
        }
    ' "$umb_target" > "$NEW_FILE" || umb_matched=$?

    if [ "$umb_matched" -eq 3 ]; then
        log_action "Managed block already absent from $umb_target"
        return 0
    fi
    [ "$umb_matched" -eq 0 ] || die "Failed to process '$umb_target'."

    if [ "$DRY_RUN" -eq 1 ]; then
        log_action "Remove managed block from $umb_target"
        return 0
    fi

    backup_file "$umb_target" "$2"
    if [ -s "$NEW_FILE" ]; then
        cat "$NEW_FILE" > "$umb_target"
    else
        rm -f -- "$umb_target"
    fi
    log_action "Remove managed block from $umb_target"
}

# remove_owned_agents HARNESS_NAME AGENTS_DIR SOURCE_DIR
# Remove symlinks in AGENTS_DIR whose target resolves inside SOURCE_DIR.
remove_owned_agents() {
    roa_name=$1
    roa_agents=$2
    roa_srcdir=$3

    [ -d "$roa_agents" ] || return 0
    roa_srcdir_abs=$(abs_path "$roa_srcdir" 2>/dev/null) || roa_srcdir_abs=''
    if [ -z "$roa_srcdir_abs" ]; then
        warn "Skipping $roa_name link cleanup because '$roa_srcdir' cannot be resolved."
        return 0
    fi

    for roa_entry in "$roa_agents"/*; do
        [ -L "$roa_entry" ] || continue
        roa_cur=$(readlink "$roa_entry")
        case $roa_cur in
            /*) roa_cand=$roa_cur ;;
            *) roa_cand=$roa_agents/$roa_cur ;;
        esac
        roa_cand_abs=$(abs_path "$roa_cand" 2>/dev/null) || roa_cand_abs=''
        case $roa_cand_abs in
            "$roa_srcdir_abs"/*) ;;
            *) continue ;;
        esac
        if [ "$DRY_RUN" -eq 1 ]; then
            log_action "Remove managed link $roa_entry"
        else
            rm -f -- "$roa_entry"
            log_action "Remove managed link $roa_entry"
        fi
    done
}

# remove_karl_skills
# Remove karl-* skill directories under <target>/.agents/skills that contain a
# SKILL.md; unrelated skills and karl-* directories without SKILL.md survive.
remove_karl_skills() {
    rks_skills=$TARGET_HOME/.agents/skills
    [ -d "$rks_skills" ] || return 0
    for rks_dir in "$rks_skills"/karl-*; do
        [ -d "$rks_dir" ] || continue
        [ -f "$rks_dir/SKILL.md" ] || continue
        if [ "$DRY_RUN" -eq 1 ]; then
            log_action "Remove skill $rks_dir"
        else
            rm -rf -- "$rks_dir"
            log_action "Remove skill $rks_dir"
        fi
    done
}

# --- Main -------------------------------------------------------------------

OPENCODE_ROOT=$TARGET_HOME/.config/opencode
CODEX_ROOT=$TARGET_HOME/.codex

if [ "$UNINSTALL" -eq 1 ]; then
    if [ -d "$OPENCODE_ROOT" ]; then
        uninstall_managed_block "$OPENCODE_ROOT/AGENTS.md" opencode/AGENTS.md
    fi
    if [ -d "$CODEX_ROOT" ]; then
        uninstall_managed_block "$CODEX_ROOT/AGENTS.md" codex/AGENTS.md
    fi
    remove_owned_agents opencode "$OPENCODE_ROOT/agents" "$REPO_ROOT/harnesses/opencode/agents"
    remove_owned_agents codex "$CODEX_ROOT/agents" "$REPO_ROOT/harnesses/codex/agents"
    remove_karl_skills
else
    validate_skills

    if [ -d "$OPENCODE_ROOT" ]; then
        update_managed_block "$OPENCODE_ROOT/AGENTS.md" opencode/AGENTS.md
    fi
    if [ -d "$CODEX_ROOT" ]; then
        update_managed_block "$CODEX_ROOT/AGENTS.md" codex/AGENTS.md
    fi

    sync_agents opencode "$OPENCODE_ROOT" "$REPO_ROOT/harnesses/opencode/agents" md
    sync_agents codex "$CODEX_ROOT" "$REPO_ROOT/harnesses/codex/agents" toml
fi

if [ "$DRY_RUN" -ne 1 ] && [ -d "$BACKUP_ROOT" ]; then
    printf 'Backups: %s\n' "$BACKUP_ROOT"
fi
