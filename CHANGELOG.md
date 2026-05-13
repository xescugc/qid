# Changelog

## Unreleased

### Added
- Users and Teams with role-based access control (global admin, team admin, team member)
- JWT-based authentication with per-route authorization middleware
- User login, creation and listing endpoints
- Team CRUD with canonical name handling
- Team member management (add, update admin role, remove) with orphan protection
- Frontend authorization guards: UI elements hidden based on user role
- Frontend route guards preventing direct URL navigation to unauthorized pages
- HTTP client methods for all user and team endpoints
- Integration tests for member role restrictions (no create/edit/delete, can trigger jobs and resources)

### Changed
- All resource endpoints now scoped under `/teams/{team_canonical}/`
- Refactored HTTP transport from go-kit endpoints to direct handler functions
- Split monolithic service file into per-entity files (pipelines, jobs, builds, resources, teams, users)
- Split HTTP handlers into per-entity files
- Simplified HTTP client by removing go-kit transport/endpoint boilerplate
- Members can now trigger jobs and resources (previously admin-only in the UI)

### Removed
- `qid/transport/endpoints.go` (go-kit endpoint definitions)
- `qid/transport/http/client/endpoints.go` (go-kit client endpoints)
- `qid/transport/http/client/transport.go` (go-kit client transports)
