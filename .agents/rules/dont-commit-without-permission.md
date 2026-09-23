# Strict Git Permission Gate: Explicit Instruction Only

> [!CAUTION]
> **NEVER AUTO-COMMIT OR AUTO-PUSH**: Under NO circumstances should the agent run `git commit`, `git tag`, or `git push` autonomously as part of normal code edits, fixes, refactorings, or documentation changes.

### Mandatory Rules
1. **No Implicit Commits / Pushes**: When editing code, docs, or fixing bugs, finish the work in the working tree and report the result to the user. **DO NOT** commit or push automatically.
2. **Explicit Trigger Only**:
   - Only execute `git commit` when the user explicitly sends `/commit` or writes "please commit" in the current turn.
   - Only execute `git push` when the user explicitly sends `/push` or writes "please push" in the current turn.
3. **Decoupled Workflow**: Even after a `/commit`, **DO NOT push automatically** unless `/push` was also requested. Always ask the user if they would like to push.
