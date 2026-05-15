# Changelog

## Unreleased

- Add ordered plan execution and `put` step support: jobs now execute `get`, `task`, and `put` steps in the order they appear in HCL, enabling CD workflows like build→push→deploy. The `put` step invokes `resource_type.push` to push content to resources ([#169](https://github.com/xescugc/pikoci/issues/169), [#72](https://github.com/xescugc/pikoci/issues/72))
- Redesign UI with PICO-8 color palette, Plus Jakarta Sans / JetBrains Mono fonts, dark mode toggle, improved pipeline graph styling, and modernized layout for all views ([#133](https://github.com/xescugc/pikoci/issues/133))
- CLI auto-refreshes stale JWT when backend returns `X-Refresh-Token` header, persisting the new token to disk ([#130](https://github.com/xescugc/pikoci/issues/130))
- Fix entity still shown in collection when backend creation fails: add Unit of Work (transaction) pattern for multi-step DB operations and `wait: true` to frontend `collection.create()` calls ([#123](https://github.com/xescugc/pikoci/issues/123))
- Fix user permission changes not reflected until re-login: backend signals stale JWT via `X-Refresh-Token` header, frontend auto-refreshes session ([#126](https://github.com/xescugc/pikoci/pull/126))
- Add users and teams with role-based access control, refactor HTTP transport from go-kit to direct handlers ([#124](https://github.com/xescugc/pikoci/pull/124))
