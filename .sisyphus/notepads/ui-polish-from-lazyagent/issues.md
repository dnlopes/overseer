## Task 20 — tmux QA and ANSI color goldens

- First `go test ./internal/adapters/primary/tui/styles/... -run TestColorGolden -update` failed because the styles package had not registered the `-update` test flag. Added a guarded `flag.Bool("update", false, ...)` in the color golden test file.
