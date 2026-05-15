# Changelog

## Unreleased

- Replace in-memory cron scheduler with database-polling scheduler for horizontal scaling support. Multiple server instances can now run concurrently with PostgreSQL or MySQL. Minimum check_interval is now 10 seconds. Manual and webhook triggers reset the check interval timer ([#213](https://github.com/xescugc/pikoci/issues/213), [#215](https://github.com/xescugc/pikoci/issues/215))
- Add `source` field on `resource_type` and `runner` blocks for URL-based definition sharing (`pikoci://` and `https://` schemes), built-in `git` resource type with API-aware check for GitHub/GitLab, built-in `docker` runner, HCL standard functions (string, collection, numeric, encoding, regex), and optional `params` on resource types ([#11](https://github.com/xescugc/pikoci/issues/11), [#143](https://github.com/xescugc/pikoci/issues/143), [#206](https://github.com/xescugc/pikoci/issues/206), [#104](https://github.com/xescugc/pikoci/issues/104))
- Add `docs/` folder with GitHub Actions workflow to sync wiki on push ([#211](https://github.com/xescugc/pikoci/issues/211))
- Add public pipelines support: pipelines can be marked public, exposing read-only views of the graph, jobs, builds, and resources without authentication ([#100](https://github.com/xescugc/pikoci/issues/100))
- Add webhook triggers for resources: each resource gets a webhook token for external trigger via `POST /webhooks/<token>`, with token regeneration endpoint ([#144](https://github.com/xescugc/pikoci/issues/144), [#181](https://github.com/xescugc/pikoci/issues/181))
- Replace shellquote splitting with native HCL list args for runner command arguments ([#201](https://github.com/xescugc/pikoci/issues/201))
- Replace pixel-art PNG logo/favicon with new hexagonal SVG logo using PICO-8 brand colors, remove old `aseprite/` folder ([#202](https://github.com/xescugc/pikoci/issues/202))
- Add ordered plan execution and `put` step support: jobs now execute `get`, `task`, and `put` steps in the order they appear in HCL, enabling CD workflows like build, push, deploy. The `put` step invokes `resource_type.push` to push content to resources ([#169](https://github.com/xescugc/pikoci/issues/169), [#72](https://github.com/xescugc/pikoci/issues/72))
- Fix SPA catch-all intercepting API requests ([#140](https://github.com/xescugc/pikoci/issues/140))
- Rename QID to PikoCI ([#136](https://github.com/xescugc/pikoci/issues/136))
- Redesign UI with PICO-8 color palette, Plus Jakarta Sans / JetBrains Mono fonts, dark mode toggle, improved pipeline graph styling, and modernized layout for all views ([#133](https://github.com/xescugc/pikoci/issues/133))
- CLI auto-refreshes stale JWT when backend returns `X-Refresh-Token` header, persisting the new token to disk ([#130](https://github.com/xescugc/pikoci/issues/130))
- Add PostgreSQL, RabbitMQ, and Kafka backend support with integration tests ([#128](https://github.com/xescugc/pikoci/pull/128))
- Fix entity still shown in collection when backend creation fails: add Unit of Work (transaction) pattern for multi-step DB operations and `wait: true` to frontend `collection.create()` calls ([#123](https://github.com/xescugc/pikoci/issues/123))
- Fix user permission changes not reflected until re-login: backend signals stale JWT via `X-Refresh-Token` header, frontend auto-refreshes session ([#126](https://github.com/xescugc/pikoci/pull/126))
- Add users and teams with role-based access control, refactor HTTP transport from go-kit to direct handlers ([#124](https://github.com/xescugc/pikoci/pull/124))
