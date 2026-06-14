---
name: release
description: Cut a new logsqueeze release — verify tests pass, update CHANGELOG, tag, push, and create a GitHub Release. Usage: /release v0.2.0
disable-model-invocation: true
---

Release workflow for logsqueeze. $ARGUMENTS is the version tag (e.g. `v0.2.0`).

Follow these steps in order. Stop and report clearly on the first failure.

**1. Verify clean working tree**
```bash
git status
```
Abort if there are uncommitted changes. Ask the user to commit or stash first.

**2. Run the full test suite**
```bash
go build ./...
go vet ./...
go test -race ./...
```
Abort if any step fails.

**3. Update CHANGELOG.md**
- Find the previous tag: `git describe --tags --abbrev=0 2>/dev/null || echo "(none)"`
- List commits since the last tag: `git log PREVTAG..HEAD --oneline` (or `git log --oneline` if no previous tag)
- Insert a new section above the previous `## [` entry:
  ```
  ## [$ARGUMENTS] - YYYY-MM-DD
  
  ### Added
  - ...
  
  ### Fixed
  - ...
  ```
  Use today's date. Summarize commits into Added / Changed / Fixed / Removed. Skip pure chore commits.
- Append a comparison link at the bottom:
  - If there is a previous tag: `[$ARGUMENTS]: https://github.com/frontdoorrr/logsqueeze/compare/PREVTAG...$ARGUMENTS`
  - If this is the first tag: `[$ARGUMENTS]: https://github.com/frontdoorrr/logsqueeze/releases/tag/$ARGUMENTS`

**4. Commit the CHANGELOG**
```bash
git add CHANGELOG.md
git commit -m "chore: release $ARGUMENTS"
```

**5. Create an annotated tag**
```bash
git tag -a $ARGUMENTS -m "Release $ARGUMENTS"
```

**6. Push commit and tag**
```bash
git push
git push origin $ARGUMENTS
```

**7. Create a GitHub Release**
Extract the changelog section for this version and use it as release notes:
```bash
gh release create $ARGUMENTS \
  --title "$ARGUMENTS" \
  --notes "$(awk '/^## \['"$ARGUMENTS"'\]/{found=1; next} found && /^## \[/{exit} found{print}' CHANGELOG.md)"
```

Report success with the GitHub Release URL.
