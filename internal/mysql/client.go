package mysql

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/jkaninda/bkupkit/config"
	"github.com/jkaninda/bkupkit/execx"
)

// Client binaries. Debian links mysql/mysqldump to the MariaDB tools, so the
// Oracle ones are recognized by their --version output, not by name.
const (
	mariadbCLI  = "mariadb"
	mariadbDump = "mariadb-dump"
	mysqlCLI    = "mysql"
	mysqlDump   = "mysqldump"
)

// Client preferences (MYSQL_CLIENT).
const (
	ClientAuto    = "auto"
	ClientMySQL   = "mysql"
	ClientMariaDB = "mariadb"
)

// tool is a resolved client binary.
type tool struct {
	name    string // logical name (mysqldump, mariadb-dump, ...)
	path    string
	mariadb bool // MariaDB client (option names differ)
}

// server is what Ping learned about a server.
type server struct {
	version string
	mariadb bool
}

// IsMariaDB reports whether a SELECT VERSION() result is a MariaDB server.
func IsMariaDB(version string) bool {
	return strings.Contains(strings.ToLower(version), "mariadb")
}

// lookTool resolves a binary and its client family (cached).
func (e *Engine) lookTool(ctx context.Context, name string) (tool, bool) {
	e.mu.Lock()
	t, ok := e.tools[name]
	e.mu.Unlock()
	if ok {
		return t, t.path != ""
	}
	t = tool{name: name}
	if p, err := e.lookPath(name); err == nil {
		t.path = p
		switch name {
		case mariadbCLI, mariadbDump:
			t.mariadb = true
		default:
			out, err := execx.Output(ctx, p, []string{"--version"}, nil)
			if err != nil {
				t.path = ""
			} else {
				t.mariadb = IsMariaDB(out)
			}
		}
	}
	e.mu.Lock()
	e.tools[name] = t
	e.mu.Unlock()
	return t, t.path != ""
}

func (e *Engine) lookPath(name string) (string, error) {
	if e.LookPath != nil {
		return e.LookPath(name)
	}
	return exec.LookPath(name)
}

// oracle returns the Oracle MySQL binary called name, if installed.
func (e *Engine) oracle(ctx context.Context, name string) (tool, bool) {
	t, ok := e.lookTool(ctx, name)
	return t, ok && !t.mariadb
}

// maria returns the MariaDB binary (mariadb-dump, or a mysqldump that is MariaDB's).
func (e *Engine) maria(ctx context.Context, name, legacy string) (tool, bool) {
	if t, ok := e.lookTool(ctx, name); ok {
		return t, true
	}
	t, ok := e.lookTool(ctx, legacy)
	return t, ok && t.mariadb
}

// pick chooses the dump (dump=true) or command-line client for a server:
// mariadb-dump/mariadb for MariaDB, Oracle mysqldump/mysql for MySQL.
func (e *Engine) pick(ctx context.Context, pref string, srv server, dump bool) (tool, error) {
	oName, mName := mysqlCLI, mariadbCLI
	if dump {
		oName, mName = mysqlDump, mariadbDump
	}
	switch pref {
	case ClientMySQL:
		if t, ok := e.oracle(ctx, oName); ok {
			return t, nil
		}
		return tool{}, fmt.Errorf("MYSQL_CLIENT=mysql but the Oracle MySQL %s client is not installed", oName)
	case ClientMariaDB:
		if t, ok := e.maria(ctx, mName, oName); ok {
			return t, nil
		}
		return tool{}, fmt.Errorf("MYSQL_CLIENT=mariadb but %s is not installed", mName)
	}
	if srv.mariadb {
		if t, ok := e.maria(ctx, mName, oName); ok {
			return t, nil
		}
		if t, ok := e.oracle(ctx, oName); ok {
			e.warnOnce("oracle-for-mariadb", "MariaDB client not installed; using the Oracle MySQL client against a MariaDB server", "client", t.name)
			return t, nil
		}
		return tool{}, fmt.Errorf("no %s or %s client installed", mName, oName)
	}
	if t, ok := e.oracle(ctx, oName); ok {
		return t, nil
	}
	if t, ok := e.maria(ctx, mName, oName); ok {
		e.warnOnce("mariadb-for-mysql",
			"Oracle MySQL client not available on this platform; using the MariaDB client against a MySQL server "+
				"(dumps of MySQL-only features may be incomplete)",
			"client", t.name, "server_version", srv.version)
		return t, nil
	}
	return tool{}, fmt.Errorf("no %s or %s client installed", oName, mName)
}

func (e *Engine) warnOnce(key, msg string, args ...any) {
	e.mu.Lock()
	seen := e.warned[key]
	e.warned[key] = true
	e.mu.Unlock()
	if !seen {
		slog.Default().Warn(msg, args...)
	}
}

// withOptionFile writes c's option file (0600, private directory) and calls fn
// with its path. The file is removed when fn returns.
func withOptionFile(v *config.Values, c Conn, t tool, fn func(path string) error) error {
	dir, cleanup, err := execx.PrivateDir(v.String(config.TmpDir))
	if err != nil {
		return fmt.Errorf("create private directory: %w", err)
	}
	defer cleanup()
	p, err := execx.WriteSecret(dir, "my.cnf", optionFile(c, t.mariadb))
	if err != nil {
		return fmt.Errorf("write client option file: %w", err)
	}
	return fn(p)
}

// query runs one SQL statement with the command-line client and returns the
// raw tab-separated output without headers.
func query(ctx context.Context, v *config.Values, c Conn, t tool, sql string) (string, error) {
	var out string
	err := withOptionFile(v, c, t, func(cnf string) error {
		var err error
		out, err = execx.Output(ctx, t.path, []string{"--defaults-extra-file=" + cnf, "-N", "-B", "-r", "-e", sql}, nil)
		return err
	})
	return out, err
}

// server detects the server version and flavour (cached per address and user).
// It tries the MariaDB client first (it speaks every MySQL authentication
// method), then the Oracle client.
func (e *Engine) server(ctx context.Context, v *config.Values, c Conn) (server, error) {
	key := c.Addr() + "\x00" + c.User
	e.mu.Lock()
	s, ok := e.servers[key]
	e.mu.Unlock()
	if ok {
		return s, nil
	}
	var cands []tool
	if t, ok := e.maria(ctx, mariadbCLI, mysqlCLI); ok {
		cands = append(cands, t)
	}
	if t, ok := e.oracle(ctx, mysqlCLI); ok {
		cands = append(cands, t)
	}
	if len(cands) == 0 {
		return s, errors.New("no mariadb or mysql client installed")
	}
	var firstErr error
	for _, t := range cands {
		out, err := query(ctx, v, c, t, "SELECT VERSION()")
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s = server{version: strings.TrimSpace(out), mariadb: IsMariaDB(out)}
		e.mu.Lock()
		e.servers[key] = s
		e.mu.Unlock()
		return s, nil
	}
	return s, fmt.Errorf("failed to connect to database server %s: %w", c.Addr(), firstErr)
}

// systemDatabases are never backed up one by one (mysql-bkup 1.x list).
var systemDatabases = map[string]bool{
	"information_schema": true, "performance_schema": true, "mysql": true, "sys": true, "innodb": true,
}

// listDatabases returns the user databases, minus exclude.
func (e *Engine) listDatabases(ctx context.Context, v *config.Values, c Conn, exclude []string) ([]string, error) {
	srv, err := e.server(ctx, v, c)
	if err != nil {
		return nil, err
	}
	t, err := e.pick(ctx, v.String(SettingClient), srv, false)
	if err != nil {
		return nil, err
	}
	out, err := query(ctx, v, c, t, "SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	return filterDatabases(strings.Split(out, "\n"), exclude), nil
}

func filterDatabases(names, exclude []string) []string {
	skip := map[string]bool{}
	for _, x := range exclude {
		skip[x] = true
	}
	var dbs []string
	for _, n := range names {
		n = strings.TrimRight(n, "\r")
		if n == "" || systemDatabases[strings.ToLower(n)] || skip[n] {
			continue
		}
		dbs = append(dbs, n)
	}
	return dbs
}

// quoteIdent quotes a MySQL identifier.
func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

// ensureDatabase creates db when it does not exist yet. The existence check
// runs first so that users without CREATE privilege can restore into an
// existing database.
func ensureDatabase(ctx context.Context, v *config.Values, c Conn, t tool, db string) error {
	lit := "CONVERT(X'" + hex.EncodeToString([]byte(db)) + "' USING utf8mb4)"
	out, err := query(ctx, v, c, t, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = "+lit)
	if err != nil {
		return fmt.Errorf("check database %s: %w", db, err)
	}
	if strings.TrimSpace(out) != "0" {
		return nil
	}
	slog.Default().Info("Creating database", "database", db)
	if _, err := query(ctx, v, c, t, "CREATE DATABASE IF NOT EXISTS "+quoteIdent(db)); err != nil {
		return fmt.Errorf("create database %s: %w", db, err)
	}
	return nil
}
