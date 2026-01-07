# ADR-0005: Sandbox

## Context and Problem Statement

We keep having security incidents due to Chromium's many security vulnerabilities.
These are patched rapidly, but also so often that it still incurs a lot of work on us.

We need a way to reduce the impact of vulnerabilities to a level where we can either not care, or not care _as much_.

## Considered Options

- Do nothing: keep patching Chromium as vulnerabilities are disclosed.
- Use Chromium's built-in sandboxing features.
- Implement our own "just enough" sandbox to run Chromium inside.

## Decision Outcome

We will build our own "just enough" sandbox, and if possible, also implement the Chromium sandbox inside there:

- The current rate of incidents is too high to be sustainable.
- Chromium's sandbox is strong, but there is still a lot of talk back and forth between the renderer and browser processes.
  This means that it is not _that_ uncommon to have a sandbox escape.
- We do not _need_ Chromium's full sandboxing, we only want to ensure that:
  - Persistence is impossible as processes are killed after use, and the data is gone.
  - Processes cannot access other customers' data.
  - Processes should not, but can, be used for utilising our resources maliciously (e.g. cryptocurrency mining).
