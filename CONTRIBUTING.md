# Contributing Guide

## Development cycle
1. Pull latest `main`.
2. Implement one logical change set.
3. Run:
   - `go test ./...`
   - `go build ./...`
4. Commit with a concise title and 1-2 lines of rationale.
5. Push to `main`.

## Definition of done
- Feature/fix has tests where practical.
- No failing CI checks.
- README/docs updated if behavior or commands changed.

## Commit size policy
- Prefer medium commits:
  - not tiny one-line commits for every edit,
  - not huge mixed commits touching unrelated concerns.
- Target: 1-3 commits per completed task block.

## Reliability rules
- Validate all external input (HTTP payloads, scenario configs).
- Keep cancellation paths explicit via `context.Context`.
- Preserve observability: new runtime features should expose metrics when possible.
