# Code Review Instructions

This file provides additional context to the code review instructions.

## Stay focused on the pull request diff

- Only comment on lines and logic that are part of the pull request changes.
- Do not suggest improvements, refactors, or style changes to surrounding code that was not modified in the pull request.
- Exception: Critical security vulnerabilities may be flagged regardless of whether the code was changed.

## Suppress known false-positives

Do not produce comments about the following patterns:

- `time.Tick` and `time.After` leaking a goroutine if the codebase uses Go version 1.23 or newer
- Keeping reference of a loop variable after the end of its iteration if the codebase uses Go version 1.22 or newer
- `new(...)` expecting a type not a value if the codebase uses Go version 1.26
