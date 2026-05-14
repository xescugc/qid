package backends_test

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/cycloidio/sqlr"
	"github.com/xescugc/pikoci/pikoci/mysql"
	"github.com/xescugc/pikoci/pikoci/mysql/migrate"

	_ "gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/natspubsub"
	_ "gocloud.dev/pubsub/rabbitpubsub"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func dbSystems() []string {
	v := envOr("PIKOCI_TEST_DB_SYSTEMS", "mem")
	return strings.Split(v, ",")
}

func pubsubSystems() []string {
	v := envOr("PIKOCI_TEST_PUBSUB_SYSTEMS", "mem")
	return strings.Split(v, ",")
}

type dbSetup struct {
	db      *sql.DB
	querier sqlr.Querier
	system  string
}

func openDB(t *testing.T, system string) *dbSetup {
	t.Helper()

	var (
		host, user, password, dbName string
		port                         int
	)

	switch system {
	case "mem":
		// no config needed
	case "sqlite":
		// Use temp file
	case "mysql":
		hp := envOr("PIKOCI_TEST_MYSQL_HOST", "127.0.0.1:3306")
		parts := strings.Split(hp, ":")
		host = parts[0]
		port, _ = strconv.Atoi(parts[1])
		user = envOr("PIKOCI_TEST_MYSQL_USER", "root")
		password = envOr("PIKOCI_TEST_MYSQL_PASSWORD", "root123")
		dbName = fmt.Sprintf("qid_test_%s_%d", system, os.Getpid())
	case "postgresql":
		hp := envOr("PIKOCI_TEST_PG_HOST", "127.0.0.1:5432")
		parts := strings.Split(hp, ":")
		host = parts[0]
		port, _ = strconv.Atoi(parts[1])
		user = envOr("PIKOCI_TEST_PG_USER", "postgres")
		password = envOr("PIKOCI_TEST_PG_PASSWORD", "postgres123")
		dbName = fmt.Sprintf("qid_test_%s_%d", system, os.Getpid())
	case "cockroachdb":
		hp := envOr("PIKOCI_TEST_COCKROACH_HOST", "127.0.0.1:26257")
		parts := strings.Split(hp, ":")
		host = parts[0]
		port, _ = strconv.Atoi(parts[1])
		user = "root"
		password = ""
		dbName = fmt.Sprintf("qid_test_%s_%d", system, os.Getpid())
	case "tidb":
		hp := envOr("PIKOCI_TEST_TIDB_HOST", "127.0.0.1:4001")
		parts := strings.Split(hp, ":")
		host = parts[0]
		port, _ = strconv.Atoi(parts[1])
		user = "root"
		password = ""
		dbName = fmt.Sprintf("qid_test_%s_%d", system, os.Getpid())
	default:
		t.Fatalf("unknown db system: %s", system)
	}

	// Map compatible systems to their underlying driver system
	driverSystem := system
	switch system {
	case "cockroachdb":
		driverSystem = mysql.PostgreSQL
	case "tidb":
		driverSystem = mysql.MySQL
	}

	opts := mysql.Options{
		DBName:          dbName,
		MultiStatements: true,
		ClientFoundRows: true,
		System:          driverSystem,
	}

	if system == "sqlite" {
		tmpFile := t.TempDir() + "/test.db"
		opts.DBFile = tmpFile
	}

	db, err := mysql.New(host, port, user, password, opts)
	if err != nil {
		t.Fatalf("failed to open db for system %s: %v", system, err)
	}

	t.Cleanup(func() {
		// Drop test database for non-memory/sqlite backends
		if system != "mem" && system != "sqlite" {
			db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		}
		db.Close()
	})

	var querier sqlr.Querier = db
	if mysql.IsPostgreSQL(driverSystem) {
		querier = mysql.NewPGQuerier(db)
	}

	return &dbSetup{
		db:      db,
		querier: querier,
		system:  driverSystem,
	}
}

func migrateDB(t *testing.T, setup *dbSetup) {
	t.Helper()
	err := migrate.Migrate(setup.db, setup.system)
	if err != nil {
		t.Fatalf("migration failed for system %s: %v", setup.system, err)
	}
}
