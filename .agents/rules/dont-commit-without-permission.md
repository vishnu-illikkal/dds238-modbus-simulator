---
trigger: always_on
description: Do not perform git commits or push changes unless explicitly instructed by the user.
---

# Do Not Commit Without Permission

- **Rule**: Never run `git commit` or `git push` commands automatically or on behalf of the user unless the user has explicitly requested to commit or run the commit workflow in the current message turn.
- **Exceptions**: Only run commit commands when the user explicitly uses slash commands like `/commit`, `/commit-viewer`, or requests a commit/push directly.
