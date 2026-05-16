# Database Backends

PikoCI supports multiple database backends via the `--db-system` flag.

## mem (default)

In-memory storage. All data is lost when the server stops. Useful for development and testing.

```bash
pikoci server --jwt-secret my-secret --db-system mem
```

No additional flags required.

## sqlite

Embedded SQLite database. Data persists in a single file. Good for single-node deployments.

```bash
pikoci server --jwt-secret my-secret --db-system sqlite --db-name pikoci.db
```

| Flag | Description |
|------|-------------|
| `--db-name` | Path to the SQLite database file |

## mysql

MySQL or MariaDB.

```bash
pikoci server \
  --jwt-secret my-secret \
  --db-system mysql \
  --db-host 127.0.0.1 \
  --db-port 3306 \
  --db-user pikoci \
  --db-password secret \
  --db-name pikoci
```

| Flag | Description |
|------|-------------|
| `--db-host` | MySQL host |
| `--db-port` | MySQL port |
| `--db-user` | MySQL user |
| `--db-password` | MySQL password |
| `--db-name` | Database name |

## postgresql

PostgreSQL.

```bash
pikoci server \
  --jwt-secret my-secret \
  --db-system postgresql \
  --db-host 127.0.0.1 \
  --db-port 5432 \
  --db-user pikoci \
  --db-password secret \
  --db-name pikoci
```

| Flag | Description |
|------|-------------|
| `--db-host` | PostgreSQL host |
| `--db-port` | PostgreSQL port |
| `--db-user` | PostgreSQL user |
| `--db-password` | PostgreSQL password |
| `--db-name` | Database name |

## Migrations

Migrations run automatically on startup by default. To disable:

```bash
pikoci server --run-migrations=false ...
```

The initial migration (V8) seeds the default user `admin` / `admin123`.
