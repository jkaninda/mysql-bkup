package mysql

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jkaninda/bkupkit/cli"
	"github.com/jkaninda/bkupkit/config"
	"github.com/jkaninda/bkupkit/engine"
)

func resolve(t *testing.T, env config.MapEnv) *config.Values {
	t.Helper()
	r := config.NewRegistry()
	config.RegisterCommon(r)
	New().Settings(r)
	v, err := r.Resolve(config.ResolveOptions{Env: env})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestParseSSLMode(t *testing.T) {
	cases := map[string]string{
		"": SSLDisabled, "0": SSLDisabled, "false": SSLDisabled, "DISABLED": SSLDisabled,
		"1": SSLRequired, "true": SSLRequired, "Required": SSLRequired,
		"preferred": SSLPreferred, "VERIFY_CA": SSLVerifyCA, "verify-identity": SSLVerifyIdentity,
	}
	for in, want := range cases {
		got, err := ParseSSLMode(in)
		if err != nil || got != want {
			t.Errorf("ParseSSLMode(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseSSLMode("2"); err == nil {
		t.Error("expected an error for 2")
	}
}

func TestParseURL(t *testing.T) {
	c, err := ParseURL("mysql://bob:p%40ss%3Aw%2Frd@db.example:3307/shop?ssl-mode=verify_identity&ssl-ca=/ca.pem")
	if err != nil {
		t.Fatal(err)
	}
	want := Conn{Host: "db.example", Port: 3307, User: "bob", Password: "p@ss:w/rd", DB: "shop", SSLMode: SSLVerifyIdentity, SSLCA: "/ca.pem"}
	if c != want {
		t.Fatalf("got %+v\nwant %+v", c, want)
	}
	if _, err := ParseURL("postgres://a@b/c"); err == nil {
		t.Fatal("expected scheme error")
	}
	if _, err := ParseURL("mysql://a@b:99999/c"); err == nil {
		t.Fatal("expected port error")
	}
}

func TestSourceConnPrecedence(t *testing.T) {
	v := resolve(t, config.MapEnv{
		"DB_URL": "mariadb://u:pw@h1/db1", "DB_HOST": "ignored", "DB_PORT": "3310", "DB_NAME": "ignored",
		"DB_SSL_MODE": "1",
	})
	c, err := SourceConn(v)
	if err != nil {
		t.Fatal(err)
	}
	if c.Host != "h1" || c.Port != 3310 || c.User != "u" || c.Password != "pw" || c.DB != "db1" || c.SSLMode != SSLRequired {
		t.Fatalf("%+v", c)
	}
	if c.SSLCA != DefaultCA {
		t.Fatalf("default CA = %q", c.SSLCA)
	}

	v = resolve(t, config.MapEnv{"DB_HOST": "h", "DB_USER": "legacy", "DB_PASSWORD": "x", "DB_NAME": "d"})
	c, _ = SourceConn(v)
	if c.User != "legacy" || c.Port != 3306 || c.SSLMode != SSLDisabled {
		t.Fatalf("%+v", c)
	}

	v = resolve(t, config.MapEnv{"DB_SSL_MODE": "sometimes"})
	if err := New().Configure(context.Background(), v); err == nil {
		t.Fatal("invalid DB_SSL_MODE accepted")
	}
}

func TestTargetConn(t *testing.T) {
	v := resolve(t, config.MapEnv{"DB_NAME": "a"})
	if _, err := TargetConn(v); err == nil {
		t.Fatal("expected missing TARGET_DB_HOST")
	}
	v = resolve(t, config.MapEnv{"TARGET_DB_HOST": "t", "TARGET_DB_USERNAME": "tu", "TARGET_DB_PASSWORD": "tp", "TARGET_DB_SSL_MODE": "verify_ca"})
	c, err := TargetConn(v)
	if err != nil {
		t.Fatal(err)
	}
	if c.Host != "t" || c.User != "tu" || c.Password != "tp" || c.Port != 3306 || c.SSLMode != SSLVerifyCA {
		t.Fatalf("%+v", c)
	}
}

func TestQuoteOption(t *testing.T) {
	cases := map[string]string{
		`plain`:       `"plain"`,
		`a"b`:         `"a\"b"`,
		`back\slash`:  `"back\\slash"`,
		" # spaced ":  `" # spaced "`,
		"new\nline\t": `"new\nline\t"`,
		`it's $x`:     `"it's $x"`,
	}
	for in, want := range cases {
		if got := quoteOption(in); got != want {
			t.Errorf("quoteOption(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestOptionFile(t *testing.T) {
	c := Conn{Host: "db", Port: 3306, User: "root", Password: `p"w#`, SSLMode: SSLDisabled, SSLCA: DefaultCA}
	got := optionFile(c, false)
	for _, want := range []string{"[client]\n", "protocol=TCP\n", `host="db"` + "\n", "port=3306\n", `user="root"` + "\n",
		`password="p\"w#"` + "\n", "ssl-mode=DISABLED\n", "get-server-public-key=1\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("mysql option file lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ssl-ca") {
		t.Errorf("CA written without verification:\n%s", got)
	}
	got = optionFile(c, true)
	if !strings.Contains(got, "ssl=0\n") || strings.Contains(got, "ssl-mode") || strings.Contains(got, "get-server-public-key") {
		t.Errorf("mariadb option file:\n%s", got)
	}

	c.SSLMode = SSLVerifyIdentity
	if got := optionFile(c, false); !strings.Contains(got, "ssl-mode=VERIFY_IDENTITY\n") || !strings.Contains(got, `ssl-ca="`+DefaultCA+`"`) {
		t.Errorf("verify_identity:\n%s", got)
	}
	if got := optionFile(c, true); !strings.Contains(got, "ssl=1\nssl-verify-server-cert=1\n") || !strings.Contains(got, "ssl-ca=") {
		t.Errorf("mariadb verify:\n%s", got)
	}
	c.SSLMode = SSLRequired
	if got := optionFile(c, true); !strings.Contains(got, "ssl=1\nssl-verify-server-cert=0\n") || strings.Contains(got, "ssl-ca=") {
		t.Errorf("mariadb required:\n%s", got)
	}

	c.Host = "/run/mysqld/mysqld.sock"
	if got := optionFile(c, true); !strings.Contains(got, "protocol=SOCKET\n") || !strings.Contains(got, `socket="/run/mysqld/mysqld.sock"`) || strings.Contains(got, "host=") {
		t.Errorf("socket:\n%s", got)
	}
}

func TestDumpArgs(t *testing.T) {
	base := []string{"--single-transaction", "--routines", "--triggers", "--events", "--hex-blob", "--quick"}
	cases := []struct {
		name                string
		toolMaria, srvMaria bool
		spec                dumpSpec
		want                []string
	}{
		{"oracle-mysql", false, false, dumpSpec{db: "shop"}, append(append([]string{}, base...), "--no-tablespaces", "--set-gtid-purged=OFF", "shop")},
		{"maria-maria", true, true, dumpSpec{db: "shop"}, append(append([]string{}, base...), "shop")},
		{"maria-mysql", true, false, dumpSpec{all: true}, append(append([]string{}, base...), "--no-tablespaces", "--all-databases")},
		{"oracle-maria", false, true, dumpSpec{databases: []string{"a", "b"}}, append(append([]string{}, base...), "--column-statistics=0", "--databases", "a", "b")},
	}
	for _, c := range cases {
		if got := DumpArgs(c.toolMaria, c.srvMaria, c.spec); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}

func TestFilterDatabases(t *testing.T) {
	got := filterDatabases([]string{"information_schema", "shop", "mysql", "Performance_Schema", "sys", "blog", "", "tmp\r"}, []string{"tmp"})
	if !reflect.DeepEqual(got, []string{"shop", "blog"}) {
		t.Fatal(got)
	}
}

func TestIsMariaDB(t *testing.T) {
	if !IsMariaDB("11.8.3-MariaDB-ubu2404") || IsMariaDB("8.4.6") {
		t.Fatal("flavour detection")
	}
}

func TestQuoteIdent(t *testing.T) {
	if got := quoteIdent("we`ird"); got != "`we``ird`" {
		t.Fatal(got)
	}
}

func TestTargets(t *testing.T) {
	e := New()
	ctx := context.Background()
	for _, c := range []struct{ storagePath, file, wantRoot, wantFile string }{
		{"/backup", "/backup/x.sql.gz", "/backup", "x.sql.gz"},
		{"/backup", "/backup/sub/x.sql.gz", "/backup", "sub/x.sql.gz"},
		{"/backup", "/data/x.sql", "/data", "x.sql"},
		{"/backup", "x.sql", "/backup", "x.sql"},
	} {
		v := resolve(t, config.MapEnv{"STORAGE_PATH": c.storagePath, "FILE_NAME": c.file, "DB_NAME": "x"})
		if _, err := e.Targets(ctx, v); err != nil || v.String(config.StoragePath) != c.wantRoot || v.String(config.FileName) != c.wantFile {
			t.Errorf("%s: %q %q %v", c.file, v.String(config.StoragePath), v.String(config.FileName), err)
		}
	}
	v0 := resolve(t, config.MapEnv{"STORAGE": "Remote", "DB_NAME": "x"})
	if _, err := e.Targets(ctx, v0); err != nil || v0.String(config.Storage) != "sftp" {
		t.Fatalf("storage remote: %q %v", v0.String(config.Storage), err)
	}
	v := resolve(t, config.MapEnv{"DB_NAME": "shop"})
	ts, err := e.Targets(ctx, v)
	if err != nil || len(ts) != 1 || ts[0].Name != "shop" {
		t.Fatalf("%v %v", ts, err)
	}
	v = resolve(t, config.MapEnv{})
	ts, err = e.Targets(ctx, v)
	if err != nil || len(ts) != 1 || ts[0].Name != "" {
		t.Fatalf("no name: %v %v", ts, err)
	}
	var ee *cli.ExitError
	if err := e.Ping(ctx, ts[0]); !errors.As(err, &ee) || ee.Code != cli.ExitConfig || !strings.Contains(err.Error(), "DB_NAME") {
		t.Fatalf("backup without a database: %v", err)
	}
	// restore without a database (all-in-one dumps carry their own USE).
	v = resolve(t, config.MapEnv{"FILE_NAME": "all_databases_20260101_000000.sql.gz"})
	if ts, err := e.Targets(ctx, v); err != nil || ts[0].Name != "" {
		t.Fatalf("%v %v", ts, err)
	}
	v = resolve(t, config.MapEnv{"BACKUP_ALL_IN_ONE": "true"})
	if ts, err := e.Targets(ctx, v); err != nil || ts[0].Name != AllInOneName || restoreDB(ts[0]) != "" {
		t.Fatalf("%v %v", ts, err)
	}
	v = resolve(t, config.MapEnv{"DB_NAME": "-x"})
	if _, err := e.Targets(ctx, v); err == nil {
		t.Fatal("option-like database name accepted")
	}
}

// fakeClients installs shell scripts standing in for the client binaries.
// They log their argv and the option file, and answer the queries the engine runs.
func fakeClients(t *testing.T, serverVersion string, oracle bool) (dir, logFile string) {
	t.Helper()
	dir = t.TempDir()
	logFile = filepath.Join(dir, "calls.log")
	script := `#!/bin/sh
log="` + logFile + `"
echo "ARGV $(basename "$0") $*" >> "$log"
case "$1" in --defaults-extra-file=*) f="${1#--defaults-extra-file=}"; echo "MODE $(ls -ln "$f" | cut -c1-10)" >> "$log"; sed 's/^/CNF /' "$f" >> "$log";; esac
for a in "$@"; do
  case "$a" in
    --version) echo "$(basename "$0")  Ver 8.4.6 for Linux (MySQL Community Server - GPL)"; exit 0;;
    "SELECT VERSION()") echo "` + serverVersion + `"; exit 0;;
    "SHOW DATABASES") printf 'information_schema\nmysql\nshop\nblog\nsys\n'; exit 0;;
    SELECT\ COUNT*) echo 0; exit 0;;
    CREATE\ DATABASE*) exit 0;;
  esac
done
case "$(basename "$0")" in
  *dump) echo "-- dump of $*";;
  *) cat > "` + filepath.Join(dir, "restored.sql") + `";;
esac
`
	names := []string{"mariadb", "mariadb-dump"}
	if oracle {
		names = append(names, "mysql", "mysqldump")
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir, logFile
}

func engineWith(dir string) *Engine {
	e := New()
	e.LookPath = func(name string) (string, error) {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			return "", err
		}
		return p, nil
	}
	return e
}

func TestDumpAndRestoreUseOptionFile(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, version    string
		oracle           bool
		wantDump, wantCL string
	}{
		{"mysql", "8.4.6", true, "mysqldump", "mysql"},
		{"mariadb", "11.8.3-MariaDB", true, "mariadb-dump", "mariadb"},
		{"mysql-without-oracle-client", "8.4.6", false, "mariadb-dump", "mariadb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, logFile := fakeClients(t, tc.version, tc.oracle)
			tmp := t.TempDir()
			v := resolve(t, config.MapEnv{"DB_HOST": "db", "DB_USERNAME": "root", "DB_PASSWORD": "s3cr3t pw", "DB_NAME": "shop", "TMP_DIR": tmp})
			e := engineWith(dir)
			tg := engine.Target{Name: "shop", Values: v}
			if err := e.Ping(ctx, tg); err != nil {
				t.Fatal(err)
			}
			if got := e.ServerVersion(ctx, tg); got != tc.version {
				t.Fatalf("version %q", got)
			}
			var out bytes.Buffer
			if err := e.Dump(ctx, tg, &out); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(out.String(), "-- dump of --defaults-extra-file=") || !strings.HasSuffix(strings.TrimSpace(out.String()), " shop") {
				t.Fatalf("dump output %q", out.String())
			}
			if err := e.Restore(ctx, tg, strings.NewReader("CREATE TABLE t (id int);\n"), engine.ArtifactInfo{File: "x.sql"}); err != nil {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(logFile)
			calls := string(b)
			if strings.Contains(strings.Join(argvLines(calls), "\n"), "s3cr3t") {
				t.Fatalf("password in argv:\n%s", calls)
			}
			if !strings.Contains(calls, `CNF password="s3cr3t pw"`) {
				t.Fatalf("password not in option file:\n%s", calls)
			}
			if !strings.Contains(calls, "MODE -rw-------") || strings.Contains(calls, "MODE -rw-r") {
				t.Fatalf("option file mode:\n%s", calls)
			}
			for _, l := range argvLines(calls) {
				f := strings.Fields(l)
				if len(f) < 3 || !strings.HasPrefix(f[2], "--defaults-extra-file=") {
					t.Fatalf("--defaults-extra-file is not the first argument: %q", l)
				}
			}
			if !strings.Contains(calls, "ARGV "+tc.wantDump+" ") || !strings.Contains(calls, "ARGV "+tc.wantCL+" --defaults-extra-file=") {
				t.Fatalf("clients:\n%s", calls)
			}
			if !strings.Contains(calls, "--binary-mode shop") {
				t.Fatalf("restore args:\n%s", calls)
			}
			if r, _ := os.ReadFile(filepath.Join(dir, "restored.sql")); string(r) != "CREATE TABLE t (id int);\n" {
				t.Fatalf("restored %q", r)
			}
			if ents, _ := os.ReadDir(tmp); len(ents) != 0 {
				t.Fatalf("TMP_DIR not cleaned: %v", ents)
			}
		})
	}
}

func argvLines(log string) []string {
	var out []string
	for _, l := range strings.Split(log, "\n") {
		if strings.HasPrefix(l, "ARGV ") && !strings.Contains(l, "--version") {
			out = append(out, l)
		}
	}
	return out
}

func TestAllDatabasesTargets(t *testing.T) {
	dir, _ := fakeClients(t, "8.4.6", true)
	v := resolve(t, config.MapEnv{"DB_HOST": "db", "DB_USERNAME": "root", "BACKUP_ALL_DATABASES": "1", "BACKUP_EXCLUDE_DATABASES": "blog", "TMP_DIR": t.TempDir()})
	ts, err := engineWith(dir).Targets(context.Background(), v)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 1 || ts[0].Name != "shop" {
		t.Fatalf("%v", ts)
	}
}

func TestMigratePipes(t *testing.T) {
	dir, logFile := fakeClients(t, "11.8.3-MariaDB", true)
	v := resolve(t, config.MapEnv{"DB_HOST": "src", "DB_USERNAME": "root", "DB_PASSWORD": "a", "DB_NAME": "shop",
		"TARGET_DB_HOST": "dst", "TARGET_DB_USERNAME": "root", "TARGET_DB_PASSWORD": "b", "TARGET_DB_NAME": "shop2", "TMP_DIR": t.TempDir()})
	e := engineWith(dir)
	pairs, err := e.MigrateTargets(context.Background(), v)
	if err != nil || len(pairs) != 1 || pairs[0][0].Name != "shop" || pairs[0][1].Name != "shop2" {
		t.Fatalf("%v %v", pairs, err)
	}
	if err := e.Migrate(context.Background(), pairs[0][0], pairs[0][1]); err != nil {
		t.Fatal(err)
	}
	r, _ := os.ReadFile(filepath.Join(dir, "restored.sql"))
	if !strings.HasPrefix(string(r), "-- dump of --defaults-extra-file=") {
		t.Fatalf("restored %q", r)
	}
	b, _ := os.ReadFile(logFile)
	if !strings.Contains(string(b), `CNF host="dst"`) || !strings.Contains(string(b), "--binary-mode shop2") {
		t.Fatalf("calls:\n%s", b)
	}
}

func TestVerifiesBeforeRestore(t *testing.T) {
	// MySQL cannot roll back DDL: a damaged dump must be caught first.
	if !New().VerifyBeforeRestore(engine.Target{}, engine.ArtifactInfo{}) {
		t.Fatal("restores must verify the dump first")
	}
}
