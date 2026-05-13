# Changelog

## Unreleased

- Fix user permission changes not reflected until re-login: backend signals stale JWT via `X-Refresh-Token` header, frontend auto-refreshes session ([#126](https://github.com/xescugc/qid/pull/126))
- Add users and teams with role-based access control, refactor HTTP transport from go-kit to direct handlers ([#124](https://github.com/xescugc/qid/pull/124))
