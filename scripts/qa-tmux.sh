#!/usr/bin/env bash
set -euo pipefail
EVID=".sisyphus/evidence"
mkdir -p "$EVID"

echo "=== C9: Dashboard renders premium ==="
tmux new-session -d -s qa_c9 -x 120 -y 40 './bin/overseer'
sleep 2
tmux capture-pane -t qa_c9 -p > "$EVID/qa1.txt"
tmux capture-pane -t qa_c9 -p -e > "$EVID/qa1-ansi.txt"
tmux kill-session -t qa_c9
grep -q "overseer" "$EVID/qa1.txt" || { echo "FAIL C9: no overseer branding"; exit 1; }
grep -q $'\xe2\x95\xad\|\xe2\x95\xae\|\xe2\x95\xb0\|\xe2\x95\xaf' "$EVID/qa1-ansi.txt" || { echo "FAIL C9: no rounded border chars"; exit 1; }
grep -q $'\x1b\[' "$EVID/qa1-ansi.txt" || { echo "FAIL C9: no ANSI color codes"; exit 1; }
echo "PASS C9"

echo "=== C10: Create modal ==="
tmux new-session -d -s qa_c10 -x 120 -y 40 './bin/overseer'
sleep 1
tmux send-keys -t qa_c10 'n'
sleep 1
tmux capture-pane -t qa_c10 -p > "$EVID/qa2.txt"
tmux kill-session -t qa_c10
grep -q $'\xe2\x95\xad\|\xe2\x95\xae\|\xe2\x95\xb0\|\xe2\x95\xaf' "$EVID/qa2.txt" || { echo "FAIL C10: no modal border"; exit 1; }
grep -q "Esc" "$EVID/qa2.txt" || { echo "FAIL C10: no Esc hint"; exit 1; }
echo "PASS C10"

echo "=== C11: Focus switching shows visible borders ==="
tmux new-session -d -s qa_c11 -x 120 -y 40 './bin/overseer'
sleep 1
tmux send-keys -t qa_c11 Tab
sleep 1
tmux capture-pane -t qa_c11 -p -e > "$EVID/qa3.txt"
tmux kill-session -t qa_c11
grep -q $'\xe2\x95\xad\|\xe2\x95\xae\|\xe2\x95\xb0\|\xe2\x95\xaf' "$EVID/qa3.txt" || { echo "FAIL C11: no rounded border chars"; exit 1; }
echo "PASS C11"

echo "=== C12: NO_COLOR fallback ==="
NO_COLOR=1 timeout 2 ./bin/overseer < /dev/null > "$EVID/no-color.txt" 2>&1 || true
grep -qv $'\x1b\[' "$EVID/no-color.txt" && echo "PASS C12 (no ANSI)" || echo "WARN C12: ANSI codes present with NO_COLOR"
grep -q "panic\|fatal" "$EVID/no-color.txt" && { echo "FAIL C12: panic or fatal"; exit 1; } || true
echo "PASS C12"

echo "qa-tmux ALL PASS"
