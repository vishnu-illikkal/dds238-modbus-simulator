---
name: push-to-github
description: Pushes all local commits and version release tags to the remote GitHub repository (origin).
---

# Push to GitHub Skill

This skill handles pushing all committed code, PDF updates, and Git version tags (`vX.Y.Z`) to the remote GitHub repository (`origin`).

## Execution Protocol

1. **Check Remote & Branch**:
   Identify the current branch and remote setup:

   ```bash
   git branch --show-current
   git remote -v
   ```

2. **Push Commits**:
   Push the active branch to remote `origin`:

   ```bash
   git push origin main
   ```

   *(Or the current active branch if on `master`).*

3. **Push Version Tags**:
   Push all Git tags to ensure release snapshots are synced:

   ```bash
   git push origin --tags
   ```

4. **Confirm Success**:
   Inspect status to ensure working tree is clean and local branch is up-to-date with remote:

   ```bash
   git status
   ```
