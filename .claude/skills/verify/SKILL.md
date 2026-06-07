---
name: verify
description: Build, vet, and test the entire logsqueeze module. Run this before committing or sharing a diff to confirm nothing is broken.
---

Run the following three commands in sequence. Stop and report clearly on the first failure — include the full error output and which command failed.

```bash
go build ./...
go vet ./...
go test ./...
```

If all three pass, report: "Build, vet, and tests all pass." — nothing more.

If any step fails:
1. Show the exact error output.
2. Identify which package or file is the likely source.
3. If the cause is obvious, explain it in plain terms (useful for someone still learning Go idioms).
