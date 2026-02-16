#!/bin/sh
set -e

# Docker entrypoint for KafClaw headless agents.
#
# Handles:
# 1. Auto-clone work repo if WORK_REPO_GIT_URL is set and /opt/work-repo is empty
# 2. Ensure workspace directory exists
# 3. Exec gomikrobot with supplied arguments

WORK_REPO="${MIKROBOT_AGENTS_WORK_REPO_PATH:-/opt/work-repo}"
WORKSPACE="${MIKROBOT_AGENTS_WORKSPACE:-/opt/workspace}"

# Clone work repo if mount is empty and a git URL is provided
if [ -n "$WORK_REPO_GIT_URL" ] && [ -z "$(ls -A "$WORK_REPO" 2>/dev/null)" ]; then
    echo "Cloning work repo from $WORK_REPO_GIT_URL ..."
    git clone "$WORK_REPO_GIT_URL" "$WORK_REPO"
fi

# Ensure workspace directory exists (soul files are auto-scaffolded by gateway startup)
mkdir -p "$WORKSPACE"

exec gomikrobot "$@"
