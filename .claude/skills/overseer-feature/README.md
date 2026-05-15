# overseer-feature Skill

A Claude skill for adding new features to the Overseer TUI application.

## Installation

This skill is pre-installed in the Overseer project at `.claude/skills/overseer-feature/`.

## Quick Start

Load this skill when you want to add a new feature to Overseer. It guides you through the full hexagonal architecture path: domain → use case → adapter → TUI.

## Features

- Step-by-step feature creation procedure
- Feature Shape Catalog (5 shapes)
- Worked example: Delete session feature
- Layer-specific code templates

## Self-Test

To verify the skill works:

```bash
bash tests/self_test.sh --dry-run
```

## Files

- `SKILL.md` — Main skill instructions
- `references/worked-example-delete.md` — Complete Delete feature walkthrough
- `tests/self_test.sh` — Self-test script
- `DECISIONS.md` — Links to relevant ADRs
