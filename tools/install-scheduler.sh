#!/usr/bin/env bash
# Install/uninstall the nightly lab run as a per-user launchd job (macOS).
# This script never installs anything unless you run --install explicitly.
#
#   tools/install-scheduler.sh --dry-run    # print the generated plist only
#   tools/install-scheduler.sh --install    # write ~/Library/LaunchAgents + bootstrap
#   tools/install-scheduler.sh --uninstall  # bootout + remove the plist
#   tools/install-scheduler.sh --status     # show whether the job is loaded
#
# Schedule (local time): SIM_SCHEDULE_HOUR (default 10), SIM_SCHEDULE_MINUTE (default 0).
# Frame gate in the nightly run: SIM_SCHEDULE_FRAME=0 disables it.
set -euo pipefail
cd "$(dirname "$0")/.."

REPO="$(pwd)"
LABEL="com.jutland.sim-periodic"
PLIST_DIR="$HOME/Library/LaunchAgents"
PLIST="$PLIST_DIR/$LABEL.plist"
MODE="${1:---dry-run}"
HOUR="${SIM_SCHEDULE_HOUR:-10}"
MINUTE="${SIM_SCHEDULE_MINUTE:-0}"
FRAME="${SIM_SCHEDULE_FRAME:-1}"

usage() {
  sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
}

require_lab() {
  : "${JUTLAND_SIM_HOME:?JUTLAND_SIM_HOME is required (e.g. export JUTLAND_SIM_HOME=\"\$HOME/jutland-sim-lab\")}"
}

render_plist() {
  cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>$LABEL</string>
	<key>ProgramArguments</key>
	<array>
		<string>/bin/bash</string>
		<string>$REPO/tools/sim-nightly.sh</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>JUTLAND_SIM_HOME</key>
		<string>$JUTLAND_SIM_HOME</string>
		<key>SIM_NIGHTLY_FRAME</key>
		<string>$FRAME</string>
	</dict>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Hour</key>
		<integer>$HOUR</integer>
		<key>Minute</key>
		<integer>$MINUTE</integer>
	</dict>
	<key>StandardOutPath</key>
	<string>$JUTLAND_SIM_HOME/reports/scheduler.log</string>
	<key>StandardErrorPath</key>
	<string>$JUTLAND_SIM_HOME/reports/scheduler.log</string>
	<key>RunAtLoad</key>
	<false/>
</dict>
</plist>
EOF
}

case "$MODE" in
  --dry-run)
    require_lab
    render_plist
    ;;
  --install)
    require_lab
    mkdir -p "$PLIST_DIR"
    render_plist >"$PLIST"
    plutil -lint "$PLIST" >/dev/null
    launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
    launchctl bootstrap "gui/$(id -u)" "$PLIST"
    echo "installed: $PLIST"
    echo "schedule:  daily ${HOUR}:$(printf '%02d' "$MINUTE") local time"
    echo "log:       $JUTLAND_SIM_HOME/reports/scheduler.log"
    ;;
  --uninstall)
    launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
    rm -f "$PLIST"
    echo "removed: $PLIST"
    ;;
  --status)
    if launchctl print "gui/$(id -u)/$LABEL" >/dev/null 2>&1; then
      launchctl print "gui/$(id -u)/$LABEL" | sed -n '1,20p'
    else
      echo "not installed: $LABEL"
    fi
    ;;
  *)
    usage
    exit 2
    ;;
esac
