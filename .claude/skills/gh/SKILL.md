---
name: gh
description: Use this skill when the user asks to interact with GitHub using the gh CLI — including creating or reviewing PRs, managing issues, checking CI/CD runs, viewing repo info, managing releases, or any other GitHub operation. Trigger phrases include "create a PR", "open a pull request", "check CI", "gh pr", "gh issue", "list issues", "view run", "merge PR", "close issue", "gh release".
version: 1.0.0
allowed-tools: Bash(gh:*)
---

# GitHub CLI Skill

Use the `gh` CLI (version 2.89.0, authenticated as `brandonapol`) for all GitHub operations.

## Current Repo Context
- Repo: !`gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || git remote get-url origin`
- Current branch: !`git branch --show-current`
- Open PRs: !`gh pr list --limit 5 2>/dev/null`

## Common Operations

### Pull Requests
```bash
gh pr create --title "..." --body "..."   # create PR
gh pr list                                # list open PRs
gh pr view [number]                       # view PR details
gh pr diff [number]                       # show PR diff
gh pr merge [number] --squash             # merge PR
gh pr checkout [number]                   # check out PR locally
gh pr review [number] --approve           # approve PR
gh pr review [number] --request-changes -b "feedback"
gh pr comment [number] -b "comment"
gh pr close [number]
```

### Issues
```bash
gh issue list                             # list open issues
gh issue view [number]                    # view issue
gh issue create --title "..." --body "..."
gh issue close [number]
gh issue comment [number] -b "comment"
gh issue assign [number] --assignee @me
```

### CI / Workflow Runs
```bash
gh run list                               # recent workflow runs
gh run view [run-id]                      # view run details
gh run watch [run-id]                     # watch a run live
gh run rerun [run-id] --failed            # re-run failed jobs
gh run download [run-id]                  # download artifacts
```

### Releases
```bash
gh release list
gh release view [tag]
gh release create [tag] --title "..." --notes "..."
```

### Repo
```bash
gh repo view                              # repo overview
gh repo clone owner/repo
gh api repos/{owner}/{repo}/...           # raw API access
```

## Tips
- Use `--json` flag with `-q` (jq path) for scripting: `gh pr list --json number,title -q '.[].title'`
- Use `gh api` for any endpoint not covered by named commands
- Pipe `gh pr diff` into review workflows
- `$ARGUMENTS` will contain any extra user-provided context (e.g. a PR number or issue title)

## Task

$ARGUMENTS
