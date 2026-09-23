---
description: Pushes all local commits and version release tags to the remote GitHub repository.
---

# Push Workflow

// turbo-all

> [!IMPORTANT]
> **NO PLANNING REQUIRED**: This workflow is considered **trivially simple**. Proceed directly to execution without creating an Implementation Plan or Task list.

This workflow handles pushing all local commits and version release tags (`vX.Y.Z`) to the remote GitHub repository (`origin`).

## Usage

- `/push`: Pushes all local commits on the active branch and all Git tags to `origin`.

## Execution Steps

1. **Verify Active Branch & Remotes**:

   ```bash
   git branch --show-current
   git remote -v
   ```

2. **Push Branch Commits to Remote**:

   ```bash
   git push origin <active_branch>
   ```

3. **Push All Git Release Tags to Remote**:

   ```bash
   git push origin --tags
   ```

4. **Verify Remote Status**:

   ```bash
   git status
   ```
