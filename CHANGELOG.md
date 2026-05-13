# Changelog

## Unreleased

- CLI auto-refreshes stale JWT when backend returns `X-Refresh-Token` header, persisting the new token to disk ([#130](https://github.com/xescugc/qid/issues/130))
- Fix entity still shown in collection when backend creation fails: add Unit of Work (transaction) pattern for multi-step DB operations and `wait: true` to frontend `collection.create()` calls ([#123](https://github.com/xescugc/qid/issues/123))
- Fix user permission changes not reflected until re-login: backend signals stale JWT via `X-Refresh-Token` header, frontend auto-refreshes session ([#126](https://github.com/xescugc/qid/pull/126))
- Add users and teams with role-based access control, refactor HTTP transport from go-kit to direct handlers ([#124](https://github.com/xescugc/qid/pull/124))
