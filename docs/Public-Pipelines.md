# Public Pipelines

Public pipelines let anyone view a pipeline's status without authentication. Useful for open-source projects that want to display build status.

## Making a pipeline public

Use the `--public` flag when updating a pipeline:

```bash
pikoci client -u localhost:8080 pipelines update -n my-pipeline -c pipeline.hcl --public
```

## What is exposed

Public pipeline endpoints return a sanitized view of the pipeline. The following data is **included**:

- Pipeline name and structure (jobs, resources, resource types)
- Job names and plan steps
- Resource names and types
- Resource type names and IDs
- Build status

The following data is **removed** for security:

| Field | Reason |
|-------|--------|
| `raw` (pipeline config) | May contain secrets in variable values |
| Resource `params` | May contain URLs, credentials |
| Resource `webhook_token` | Would allow unauthorized triggers |
| Resource `logs` | May contain sensitive output |
| Resource `cron_id` | Internal implementation detail |
| Resource type `check`/`pull`/`push` commands | May contain secrets in args |
| Resource type `params` list | Only `id` and `name` are retained |

## Public API endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /teams/{team}/pipelines/{pipeline}/public` | Sanitized pipeline data |
| `GET /teams/{team}/pipelines/{pipeline}/public/image?format=dot` | Pipeline graph in DOT format |

## Example

```bash
# View public pipeline data
curl http://localhost:8080/teams/main/pipelines/my-pipeline/public

# Get the pipeline graph as SVG
curl http://localhost:8080/teams/main/pipelines/my-pipeline/public/image?format=dot | dot -Tsvg > status.svg
```
