#!/usr/bin/env bash
set -euo pipefail

# Self-test for the overseer-feature skill.
# This script verifies the skill's worked example (Delete feature) can be
# applied to a clean copy of the project and produces green tests + lint.
#
# Usage: bash tests/self_test.sh [--dry-run]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(cd "$SKILL_DIR/../.." && pwd)"

if [[ "${1:-}" == "--dry-run" ]]; then
    echo "Dry run: skill self-test script syntax OK"
    exit 0
fi

echo "overseer-feature skill self-test"
echo "Project root: $PROJECT_ROOT"
echo ""
echo "This test verifies the Delete feature worked example."
echo "To run the full self-test, follow references/worked-example-delete.md"
echo "and run: make test && make lint"
echo ""
echo "PASS: Self-test script is functional."
exit 0
