// Package mysql is the mysql-bkup engine for bkupkit: MySQL and MariaDB
// logical backups with mysqldump / mariadb-dump, restored with mysql / mariadb.
package mysql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jkaninda/bkupkit/cli"
	"github.com/jkaninda/bkupkit/config"
	"github.com/jkaninda/bkupkit/engine"
	"github.com/jkaninda/bkupkit/execx"
)

// Engine-specific settings.
const (
	SettingClient = "MYSQL_CLIENT"
)

// AllInOneName is the target (and default artifact) name of --all-in-one backups.
const AllInOneName = "all_databases"

// DefaultCA is the 1.x default of DB_SSL_CA.
const DefaultCA = "/etc/ssl/certs/ca-certificates.crt"

// Engine implements engine.Engine and engine.Migrator.
type Engine struct {
	// LookPath finds client binaries (default exec.LookPath); tests replace it.
	LookPath func(string) (string, error)

	mu      sync.Mutex
	tools   map[string]tool
	servers map[string]server
	warned  map[string]bool
}

var (
	_ engine.Engine   = (*Engine)(nil)
	_ engine.Migrator = (*Engine)(nil)
)

// New returns the engine.
func New() *Engine {
	return &Engine{tools: map[string]tool{}, servers: map[string]server{}, warned: map[string]bool{}}
}

// Tool implements engine.Engine.
func (e *Engine) Tool() string { return "mysql-bkup" }

// Kind implements engine.Engine. It is "mysql" for MySQL and MariaDB alike so
// that {engine} in the index name does not depend on the server.
func (e *Engine) Kind() string { return "mysql" }

// Settings implements engine.Engine.
func (e *Engine) Settings(r *config.Registry) {
	config.RegisterDatabase(r, "3306")
	r.Override(config.DBURL, func(s *config.Setting) {
		s.Usage = "Connection URL mysql://user:password@host:port/database?ssl-mode=required (overrides host, port, user, password)"
	})
	r.Override(config.DBHost, func(s *config.Setting) {
		s.Usage = "Database host (a path starting with / is a Unix socket)"
	})
	r.Override(config.DBSSLMode, func(s *config.Setting) {
		s.Default = SSLDisabled
		s.Usage = "TLS mode: disabled, preferred, required, verify_ca, verify_identity (1.x values 0 and 1 mean disabled and required)"
	})
	r.Override(config.DBSSLCA, func(s *config.Setting) {
		s.Default = DefaultCA
		s.Usage = "CA certificate file, used with DB_SSL_MODE=verify_ca or verify_identity"
	})
	r.Override(config.TargetDBSSL, func(s *config.Setting) {
		s.Default = SSLDisabled
		s.Usage = "migrate: target TLS mode (same values as DB_SSL_MODE)"
	})
	r.Override(config.TargetDBSSLCA, func(s *config.Setting) {
		s.Default = DefaultCA
	})
	r.Override(config.TargetDBName, func(s *config.Setting) {
		s.Usage = "migrate: target database (default: the source name)"
	})
	// 1.x accepted --storage remote for SFTP.
	r.Override(config.Storage, func(s *config.Setting) {
		s.Enum = append(s.Enum, "remote")
		s.Usage = "Storage backend: local, s3, gcs, azure, sftp (ssh, remote), ftp"
	})
	r.Add(config.Setting{
		Key: SettingClient, Default: ClientAuto, Enum: []string{ClientAuto, ClientMySQL, ClientMariaDB},
		Group: config.GroupDatabase,
		Usage: "Client tools: auto (mariadb-dump for MariaDB, Oracle mysqldump for MySQL), mysql or mariadb",
	})
}

// Configure implements engine.Engine. It only validates values: commands that
// do not connect (list, prune, verify) must work without credentials.
func (e *Engine) Configure(_ context.Context, v *config.Values) error {
	normalizeStorage(v)
	var errs []error
	if _, err := SourceConn(v); err != nil {
		errs = append(errs, err)
	}
	if v.Has(config.TargetDBURL) || v.Has(config.TargetDBHost) {
		if _, err := TargetConn(v); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// normalizeStorage maps the 1.x storage name "remote" to sftp, and an
// absolute local FILE_NAME (1.x: restore -f /backup/x.sql.gz) to a file name
// relative to STORAGE_PATH (or to its own directory, as 1.x did).
func normalizeStorage(v *config.Values) {
	if v.String(config.Storage) == "remote" {
		v.Set(config.Storage, "sftp")
	}
	f := v.String(config.FileName)
	if v.String(config.Storage) != "local" || !filepath.IsAbs(f) || v.Has(config.RemotePath) {
		return
	}
	root := filepath.Clean(v.String(config.StoragePath))
	if rel, err := filepath.Rel(root, filepath.Clean(f)); err == nil && !strings.HasPrefix(rel, "..") {
		v.Set(config.FileName, filepath.ToSlash(rel))
		return
	}
	v.Set(config.StoragePath, filepath.Dir(f))
	v.Set(config.FileName, filepath.Base(f))
}

// restoreContext reports whether v belongs to a restore/verify/delete run,
// where one database at most is involved. Outside the CLI (no command), a
// FILE_NAME means the same.
func restoreContext(v *config.Values) bool {
	switch v.Command {
	case "restore", "verify", "delete":
		return true
	case "":
		return v.Has(config.FileName) && !v.Bool(config.AllDatabases) && !v.Bool(config.AllInOne)
	}
	return false
}

// Targets implements engine.Engine.
func (e *Engine) Targets(ctx context.Context, v *config.Values) ([]engine.Target, error) {
	normalizeStorage(v) // per-job values of a config file are resolved again
	c, err := SourceConn(v)
	if err != nil {
		return nil, err
	}
	if restoreContext(v) {
		// restore -f FILE [-d DB]: without a database, the dump's own USE
		// statements decide (--all-in-one dumps).
		if err := checkDBName(c.DB); err != nil {
			return nil, err
		}
		return []engine.Target{{Name: c.DB, Values: v}}, nil
	}
	switch {
	case v.Bool(config.AllInOne):
		return []engine.Target{{Name: AllInOneName, Values: v}}, nil
	case v.Bool(config.AllDatabases):
		if err := c.validate(""); err != nil {
			return nil, err
		}
		dbs, err := e.listDatabases(ctx, v, c, v.List(config.ExcludeDBs))
		if err != nil {
			return nil, err
		}
		if len(dbs) == 0 {
			return nil, errors.New("no databases found")
		}
		ts := make([]engine.Target, 0, len(dbs))
		for _, db := range dbs {
			ts = append(ts, engine.Target{Name: db, Values: v})
		}
		return ts, nil
	}
	// No database: fine for list, prune, verify and delete; Ping rejects it
	// for a backup.
	if err := checkDBName(c.DB); err != nil {
		return nil, err
	}
	return []engine.Target{{Name: c.DB, Values: v}}, nil
}

// errNoDatabase is a configuration error (exit code 2).
var errNoDatabase = &cli.ExitError{Code: cli.ExitConfig, Err: errors.New("database name is required, use DB_NAME environment variable or -d flag")}

// Ext implements engine.Engine.
func (e *Engine) Ext(engine.Target) string { return ".sql" }

// NativeCompression implements engine.Engine.
func (e *Engine) NativeCompression(engine.Target) bool { return false }

// VerifyBeforeRestore makes RESTORE_VERIFY_FIRST=auto read and check the whole
// dump before the client runs: MySQL cannot roll back the dump's DROP/CREATE
// TABLE statements, so a backup that turns out to be damaged halfway would
// otherwise leave the database half-restored.
func (e *Engine) VerifyBeforeRestore(engine.Target, engine.ArtifactInfo) bool { return true }

var _ engine.RestoreVerifier = (*Engine)(nil)

// Ping implements engine.Engine: it connects, and detects the server flavour.
func (e *Engine) Ping(ctx context.Context, t engine.Target) error {
	c, err := SourceConn(t.Values)
	if err != nil {
		return err
	}
	if t.Name == "" && !restoreContext(t.Values) {
		return errNoDatabase
	}
	if err := c.validate(""); err != nil {
		return &cli.ExitError{Code: cli.ExitConfig, Err: err}
	}
	srv, err := e.server(ctx, t.Values, c)
	if err != nil {
		return err
	}
	flavour := "MySQL"
	if srv.mariadb {
		flavour = "MariaDB"
	}
	slog.Default().Info("Connected to database server", "server", c.Addr(), "flavour", flavour, "version", srv.version)
	return nil
}

// ServerVersion implements engine.Engine.
func (e *Engine) ServerVersion(ctx context.Context, t engine.Target) string {
	c, err := SourceConn(t.Values)
	if err != nil {
		return ""
	}
	srv, err := e.server(ctx, t.Values, c)
	if err != nil {
		return ""
	}
	return srv.version
}

// dumpSpec describes what a dump covers.
type dumpSpec struct {
	db        string   // one database (no CREATE DATABASE / USE: restorable under another name)
	all       bool     // --all-databases
	databases []string // --databases list (all-in-one with exclusions)
}

// DumpArgs returns the dump arguments after --defaults-extra-file.
// mariadbTool is the client family, mariadbServer the server flavour.
func DumpArgs(mariadbTool, mariadbServer bool, s dumpSpec) []string {
	args := []string{"--single-transaction", "--routines", "--triggers", "--events", "--hex-blob", "--quick"}
	if !mariadbServer {
		// MySQL: no tablespace statements (they need the PROCESS privilege
		// and are useless in logical backups).
		args = append(args, "--no-tablespaces")
		if !mariadbTool {
			args = append(args, "--set-gtid-purged=OFF")
		}
	} else if !mariadbTool {
		// Oracle mysqldump against MariaDB: no column_statistics table there.
		args = append(args, "--column-statistics=0")
	}
	switch {
	case s.all:
		args = append(args, "--all-databases")
	case len(s.databases) > 0:
		args = append(args, "--databases")
		args = append(args, s.databases...)
	default:
		args = append(args, s.db)
	}
	return args
}

func (e *Engine) dumpCmd(ctx context.Context, t engine.Target, c Conn) (tool, []string, error) {
	v := t.Values
	srv, err := e.server(ctx, v, c)
	if err != nil {
		return tool{}, nil, err
	}
	tl, err := e.pick(ctx, v.String(SettingClient), srv, true)
	if err != nil {
		return tool{}, nil, err
	}
	spec := dumpSpec{db: t.Name}
	if t.Name == AllInOneName && v.Bool(config.AllInOne) {
		spec = dumpSpec{all: true}
		if ex := v.List(config.ExcludeDBs); len(ex) > 0 {
			dbs, err := e.listDatabases(ctx, v, c, ex)
			if err != nil {
				return tool{}, nil, err
			}
			if len(dbs) == 0 {
				return tool{}, nil, errors.New("no databases left after BACKUP_EXCLUDE_DATABASES")
			}
			spec = dumpSpec{databases: dbs}
		}
	} else if err := checkDBName(t.Name); err != nil || t.Name == "" {
		return tool{}, nil, fmt.Errorf("invalid database name %q", t.Name)
	}
	return tl, DumpArgs(tl.mariadb, srv.mariadb, spec), nil
}

// Dump implements engine.Engine: the dump goes straight to w.
func (e *Engine) Dump(ctx context.Context, t engine.Target, w io.Writer) error {
	c, err := SourceConn(t.Values)
	if err != nil {
		return err
	}
	tl, args, err := e.dumpCmd(ctx, t, c)
	if err != nil {
		return err
	}
	slog.Default().Info("Dumping database", "database", t.Name, "client", tl.name)
	return withOptionFile(t.Values, c, tl, func(cnf string) error {
		return execx.Run(ctx, execx.Cmd{
			Name: tl.path, Args: append([]string{"--defaults-extra-file=" + cnf}, args...),
			Stdout: w, Log: slog.Default(), StderrLevel: slog.LevelWarn,
		})
	})
}

// restoreDB returns the database a dump is replayed into ("" = none: the
// dump's USE statements decide).
func restoreDB(t engine.Target) string {
	if t.Name == AllInOneName && t.Values.Bool(config.AllInOne) {
		return ""
	}
	return t.Name
}

func (e *Engine) restoreCmd(ctx context.Context, v *config.Values, c Conn, db string) (tool, []string, error) {
	if err := checkDBName(db); err != nil {
		return tool{}, nil, err
	}
	srv, err := e.server(ctx, v, c)
	if err != nil {
		return tool{}, nil, err
	}
	tl, err := e.pick(ctx, v.String(SettingClient), srv, false)
	if err != nil {
		return tool{}, nil, err
	}
	if db != "" {
		if err := ensureDatabase(ctx, v, c, tl, db); err != nil {
			return tool{}, nil, err
		}
	}
	// --binary-mode: dumps may contain \0 and \r in string literals.
	args := []string{"--binary-mode"}
	if db != "" {
		args = append(args, db)
	}
	return tl, args, nil
}

// Restore implements engine.Engine. The client reads the dump from stdin and
// stops at the first failing statement (non-zero exit). If r fails, execx
// kills the client before it sees EOF. MySQL cannot roll back DDL, so a
// failed restore can leave the statements before the failure applied.
func (e *Engine) Restore(ctx context.Context, t engine.Target, r io.Reader, info engine.ArtifactInfo) error {
	c, err := SourceConn(t.Values)
	if err != nil {
		return err
	}
	db := restoreDB(t)
	tl, args, err := e.restoreCmd(ctx, t.Values, c, db)
	if err != nil {
		return err
	}
	slog.Default().Info("Restoring database", "database", db, "file", info.File, "client", tl.name)
	return withOptionFile(t.Values, c, tl, func(cnf string) error {
		return execx.Run(ctx, execx.Cmd{
			Name: tl.path, Args: append([]string{"--defaults-extra-file=" + cnf}, args...),
			Stdin: r, Log: slog.Default(), StderrLevel: slog.LevelWarn,
		})
	})
}

// MigrateTargets implements engine.Migrator: DB_NAME → TARGET_DB_NAME (default
// the same name), or every database with --all-databases.
func (e *Engine) MigrateTargets(ctx context.Context, v *config.Values) ([][2]engine.Target, error) {
	tc, err := TargetConn(v)
	if err != nil {
		return nil, err
	}
	if err := tc.validate("TARGET_"); err != nil {
		return nil, err
	}
	sc, err := SourceConn(v)
	if err != nil {
		return nil, err
	}
	if err := sc.validate(""); err != nil {
		return nil, err
	}
	if v.Bool(config.AllDatabases) || v.Bool(config.AllInOne) {
		dbs, err := e.listDatabases(ctx, v, sc, v.List(config.ExcludeDBs))
		if err != nil {
			return nil, err
		}
		pairs := make([][2]engine.Target, 0, len(dbs))
		for _, db := range dbs {
			pairs = append(pairs, [2]engine.Target{{Name: db, Values: v}, {Name: db, Values: v}})
		}
		return pairs, nil
	}
	if sc.DB == "" {
		return nil, errNoDatabase
	}
	name := tc.DB
	if name == "" {
		name = sc.DB
	}
	for _, n := range []string{sc.DB, name} {
		if err := checkDBName(n); err != nil {
			return nil, err
		}
	}
	return [][2]engine.Target{{{Name: sc.DB, Values: v}, {Name: name, Values: v}}}, nil
}

// Migrate implements engine.Migrator: dump | restore through a pipe, no file.
func (e *Engine) Migrate(ctx context.Context, source, target engine.Target) error {
	sc, err := SourceConn(source.Values)
	if err != nil {
		return err
	}
	tc, err := TargetConn(target.Values)
	if err != nil {
		return err
	}
	dumpTool, dumpArgs, err := e.dumpCmd(ctx, source, sc)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	restTool, restArgs, err := e.restoreCmd(ctx, target.Values, tc, target.Name)
	if err != nil {
		return fmt.Errorf("target: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// The first failure is the cause; the other side then fails because of it.
	var (
		once  sync.Once
		cause error
	)
	fail := func(err error) {
		once.Do(func() { cause = err })
		cancel()
	}
	pr, pw := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := withOptionFile(source.Values, sc, dumpTool, func(cnf string) error {
			return execx.Run(ctx, execx.Cmd{
				Name: dumpTool.path, Args: append([]string{"--defaults-extra-file=" + cnf}, dumpArgs...),
				Stdout: pw, Log: slog.Default(), StderrLevel: slog.LevelWarn,
			})
		})
		if err != nil {
			err = fmt.Errorf("dump %s: %w", source.Name, err)
			fail(err)
			// The restore side must never see a clean EOF.
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()
	err = withOptionFile(target.Values, tc, restTool, func(cnf string) error {
		return execx.Run(ctx, execx.Cmd{
			Name: restTool.path, Args: append([]string{"--defaults-extra-file=" + cnf}, restArgs...),
			Stdin: pr, Log: slog.Default(), StderrLevel: slog.LevelWarn,
		})
	})
	if err != nil {
		fail(fmt.Errorf("restore into %s: %w", target.Name, err))
		_ = pr.CloseWithError(err)
	}
	<-done
	return cause
}
