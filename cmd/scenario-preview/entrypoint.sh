#!/bin/sh
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -eu

export SCENARIO_ID="${SCENARIO_ID:-python-to-uds}"
export PREVIEW_WORKSPACE="${PREVIEW_WORKSPACE:-/root/myapp}"
export PATH="/opt/scenario-preview/bin:${PATH}"

# The scenario argument is also used while preparing the fixture for ttyd.
next_is_scenario=false
for arg in "$@"; do
  if [ "$next_is_scenario" = true ]; then
    SCENARIO_ID="$arg"
    next_is_scenario=false
    continue
  fi
  case "$arg" in
    --scenario) next_is_scenario=true ;;
    --scenario=*) SCENARIO_ID="${arg#--scenario=}" ;;
  esac
done

scenario-preview prepare --scenarios /scenarios --scenario "$SCENARIO_ID" --workspace "$PREVIEW_WORKSPACE"

# ttyd and click-to-run blocks attach to one tmux session. This gives the
# preview the same shared interactive-terminal behavior as a live lab.
tmux new-session -d -s preview -c "$PREVIEW_WORKSPACE" /bin/sh -l
ttyd --interface 127.0.0.1 --port 7681 --base-path /terminal --writable \
  --cwd "$PREVIEW_WORKSPACE" tmux attach-session -t preview &

exec scenario-preview serve --scenarios /scenarios --workspace "$PREVIEW_WORKSPACE" --reset=false "$@"
