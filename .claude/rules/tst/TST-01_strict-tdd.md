---
paths:
  - "**/*_test.go"
---
# TST-01: Strict TDD

## Rule
Every feature task follows RED-GREEN-REFACTOR: write a failing test FIRST, implement only after the test fails for the right reason, then refactor.

## Why
TDD prevents over-engineering, ensures tests actually test the right thing, and produces a safety net for refactoring.

## Example
✅ Good:
```
1. Write TestSessionService_Create_DuplicateName_ReturnsError (RED — fails)
2. Implement duplicate-check in Create (GREEN — passes)
3. Clean up implementation (REFACTOR)
```

❌ Bad:
```
1. Implement Create fully
2. Write tests to match the implementation
```
