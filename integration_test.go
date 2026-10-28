//go:build integration

// Integration tests: real MySQL / MariaDB servers in Docker, the real client
// tools, the whole bkupkit pipeline through cli.App.Run.
//
// The client tools run in containers too: shim scripts named mysql, mysqldump,
// mariadb and mariadb-dump are put first in PATH and `docker run -i` the
// official images (or IT_CLIENT_IMAGE, e.g. the mysql-bkup image itself). The
// private TMP_DIR holding the option files is bind-mounted at the same path.
//
//	go test -tags integration -v -run . ./...
//	IT_CLIENT_IMAGE=jkaninda/mysql-bkup:test go test -tags integration ./...
//	IT_SKIP_MYSQL80=1 ...   skip the MySQL 8.0 server
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/jkaninda/bkupkit/cli"
	"github.com/jkaninda/mysql-bkup/internal/mysql"
)

const (
	rootPW = "r00t-pw"
	// appPW exercises option-file quoting.
	appPW = `p"a#s s\w'd$x`
)

var (
	itNetwork string
	itTmp     string // TMP_DIR shared with the client containers
	itShims   string
	itSkip    string
	servers   []*dbServer
	minioPort string
)

type dbServer struct {
	name    string // container name and DB_HOST
	image   string
	client  string // client binary inside the server image
	mariadb bool
}

// Miabi takes the LAST match of this regex in the output as the artifact.
var miabiDB = regexp.MustCompile(`[\w.\-]+\.(?:sql|archive)\.gz(?:\.gpg)?`)

func docker(args ...string) (string, error) {
	// Stdout only on success: "docker run" prints image pull progress on stderr,
	// which would otherwise end up in the container id on a runner without the image.
	cmd := exec.Command("docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		out = append(out, stderr.Bytes()...)
	}
	return strings.TrimSpace(string(out)), err
}

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("docker"); err != nil {
		itSkip = "docker not installed"
		os.Exit(m.Run())
	}
	if _, err := docker("info"); err != nil {
		itSkip = "docker not running"
		os.Exit(m.Run())
	}
	code := func() int {
		suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e9)
		itNetwork = "mysqlbkup-it-" + suffix
		if out, err := docker("network", "create", itNetwork); err != nil {
			fmt.Fprintln(os.Stderr, out)
			return 1
		}
		defer docker("network", "rm", itNetwork)
		var err error
		if itTmp, err = os.MkdirTemp("", "mysqlbkup-it-tmp-"); err != nil {
			return 1
		}
		defer os.RemoveAll(itTmp)
		// macOS: /var/folders is a symlink to /private/var/folders; mount the real path.
		if p, err := filepath.EvalSymlinks(itTmp); err == nil {
			itTmp = p
		}
		if itShims, err = os.MkdirTemp("", "mysqlbkup-it-bin-"); err != nil {
			return 1
		}
		defer os.RemoveAll(itShims)
		writeShims()

		servers = []*dbServer{
			{name: "it-mysql84-" + suffix, image: "mysql:8.4", client: "mysql"},
			{name: "it-mariadb118-" + suffix, image: "mariadb:11.8", client: "mariadb", mariadb: true},
		}
		if os.Getenv("IT_SKIP_MYSQL80") == "" {
			servers = append(servers, &dbServer{name: "it-mysql80-" + suffix, image: "mysql:8.0", client: "mysql"})
		}
		for _, s := range servers {
			args := []string{"run", "-d", "--rm", "--name", s.name, "--network", itNetwork}
			if s.mariadb {
				args = append(args, "-e", "MARIADB_ROOT_PASSWORD="+rootPW)
			} else {
				args = append(args, "-e", "MYSQL_ROOT_PASSWORD="+rootPW)
			}
			if out, err := docker(append(args, s.image)...); err != nil {
				fmt.Fprintln(os.Stderr, out)
				return 1
			}
			defer docker("rm", "-f", s.name)
		}
		minio := "it-minio-" + suffix
		if out, err := docker("run", "-d", "--rm", "--name", minio, "-p", "127.0.0.1::9000",
			"-e", "MINIO_ROOT_USER=minioadmin", "-e", "MINIO_ROOT_PASSWORD=minioadmin",
			"pgsty/minio:latest", "server", "/data"); err != nil {
			fmt.Fprintln(os.Stderr, out)
			return 1
		}
		defer docker("rm", "-f", minio)
		for _, s := range servers {
			if err := s.wait(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		out, err := docker("port", minio, "9000/tcp")
		if err != nil {
			fmt.Fprintln(os.Stderr, out)
			return 1
		}
		minioPort = out[strings.LastIndex(out, ":")+1:]
		deadline := time.Now().Add(60 * time.Second)
		for {
			out, err = docker("exec", minio, "sh", "-c", "mc alias set l http://127.0.0.1:9000 minioadmin minioadmin && mc mb --ignore-existing l/backups")
			if err == nil {
				break
			}
			if time.Now().After(deadline) {
				fmt.Fprintln(os.Stderr, "minio:", out)
				return 1
			}
			time.Sleep(time.Second)
		}
		minioContainer = minio
		return m.Run()
	}()
	os.Exit(code)
}

var minioContainer string

func writeShims() {
	imgs := map[string]string{"mysql": "mysql:8.4", "mysqldump": "mysql:8.4", "mariadb": "mariadb:11.8", "mariadb-dump": "mariadb:11.8"}
	for bin, img := range imgs {
		if ci := os.Getenv("IT_CLIENT_IMAGE"); ci != "" {
			img = ci
		}
		sh := fmt.Sprintf("#!/bin/sh\nexec docker run --rm -i --network %s -v %s:%s --entrypoint %s %s \"$@\"\n", itNetwork, itTmp, itTmp, bin, img)
		if err := os.WriteFile(filepath.Join(itShims, bin), []byte(sh), 0o755); err != nil {
			panic(err)
		}
	}
}

// sql runs statements as root inside the server container.
func (s *dbServer) sql(stmts string) (string, error) {
	cmd := exec.Command("docker", "exec", "-i", "-e", "MYSQL_PWD="+rootPW, s.name, s.client, "-uroot", "-h127.0.0.1", "-N", "-B")
	cmd.Stdin = strings.NewReader(stmts)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (s *dbServer) wait() error {
	deadline := time.Now().Add(3 * time.Minute)
	for {
		// TCP only answers once the image's init server is gone.
		if _, err := s.sql("SELECT 1"); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			logs, _ := docker("logs", s.name)
			return fmt.Errorf("%s not ready:\n%s", s.name, logs)
		}
		time.Sleep(time.Second)
	}
}

func (s *dbServer) mustSQL(t *testing.T, stmts string) string {
	t.Helper()
	out, err := s.sql(stmts)
	if err != nil {
		t.Fatalf("%s: %v\n%s", s.name, err, out)
	}
	return out
}

const fixture = `
DROP DATABASE IF EXISTS %[1]s;
CREATE DATABASE %[1]s CHARACTER SET utf8mb4;
USE %[1]s;
CREATE TABLE products (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), price DECIMAL(10,2), data BLOB, created DATETIME);
CREATE TABLE audit (id INT PRIMARY KEY AUTO_INCREMENT, product_id INT, note TEXT);
CREATE TRIGGER trg_products AFTER INSERT ON products FOR EACH ROW INSERT INTO audit (product_id, note) VALUES (NEW.id, 'created');
CREATE PROCEDURE p_count() SELECT COUNT(*) FROM products;
CREATE VIEW v_products AS SELECT id, name FROM products;
CREATE EVENT ev_cleanup ON SCHEDULE EVERY 1 DAY DO DELETE FROM audit WHERE id < 0;
INSERT INTO products (name, price, data, created) VALUES
  ('café 😀', 9.99, X'00FF0A0D27225C', '2026-01-02 03:04:05'),
  ('it''s "quoted" \\ back', 0.01, NULL, NULL),
  ('tab\tnew\nline', 12345678.90, REPEAT(X'AB', 60000), '1999-12-31 23:59:59');
INSERT INTO products (name, price) SELECT CONCAT('bulk-', a.n * 10 + b.n), a.n + b.n FROM
  (SELECT 0 n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
  (SELECT 0 n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b;
`

func (s *dbServer) loadFixture(t *testing.T, db string) {
	t.Helper()
	s.mustSQL(t, fmt.Sprintf(fixture, db))
}

// fingerprint summarises the data and objects of db.
func (s *dbServer) fingerprint(t *testing.T, db string) string {
	t.Helper()
	q := fmt.Sprintf(`SET SESSION group_concat_max_len = 100000000;
SELECT COUNT(*), MD5(GROUP_CONCAT(CONCAT_WS('|', id, name, price, HEX(data), created) ORDER BY id SEPARATOR ';')) FROM %[1]s.products;
SELECT COUNT(*) FROM %[1]s.audit;
SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = '%[1]s';
SELECT COUNT(*) FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = '%[1]s';
SELECT COUNT(*) FROM information_schema.VIEWS WHERE TABLE_SCHEMA = '%[1]s';
SELECT COUNT(*) FROM information_schema.EVENTS WHERE EVENT_SCHEMA = '%[1]s';`, db)
	return s.mustSQL(t, q)
}

func needDocker(t *testing.T) {
	t.Helper()
	if itSkip != "" {
		t.Skip(itSkip)
	}
}

// run executes the CLI in-process and returns its combined output.
func run(t *testing.T, args ...string) (string, int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = io.Copy(&buf, r); close(done) }()
	code := (&cli.App{Engine: mysql.New(), Version: "it"}).Run(args)
	_ = w.Close()
	<-done
	os.Stdout, os.Stderr = oldOut, oldErr
	return buf.String(), code
}

func mustRun(t *testing.T, args ...string) string {
	t.Helper()
	out, code := run(t, args...)
	if code != 0 {
		t.Fatalf("%v: exit %d\n%s", args, code, out)
	}
	return out
}

// setEnv sets the common environment for s and returns the local storage dir.
func setEnv(t *testing.T, s *dbServer, user, pw string) string {
	t.Helper()
	store := t.TempDir()
	t.Setenv("PATH", itShims+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMP_DIR", itTmp)
	t.Setenv("STORAGE_PATH", store)
	t.Setenv("HEALTH_FILE", filepath.Join(t.TempDir(), "health.json"))
	t.Setenv("TEMPLATES_DIR", "")
	t.Setenv("DB_HOST", s.name)
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USERNAME", user)
	t.Setenv("DB_PASSWORD", pw)
	return store
}

// artifact returns the last Miabi regex match and checks the file exists in dir.
func artifact(t *testing.T, out, dir string) string {
	t.Helper()
	m := miabiDB.FindAllString(out, -1)
	if len(m) == 0 {
		t.Fatalf("no artifact name in output:\n%s", out)
	}
	last := m[len(m)-1]
	if dir != "" {
		if _, err := os.Stat(filepath.Join(dir, last)); err != nil {
			t.Fatalf("last match %q is not the stored artifact: %v\n%s", last, err, out)
		}
	}
	return last
}

func assertTmpClean(t *testing.T) {
	t.Helper()
	if ents, _ := os.ReadDir(itTmp); len(ents) != 0 {
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		t.Fatalf("TMP_DIR not empty: %v", names)
	}
}

func forEachServer(t *testing.T, fn func(t *testing.T, s *dbServer)) {
	needDocker(t)
	for _, s := range servers {
		t.Run(strings.SplitN(strings.TrimPrefix(s.name, "it-"), "-", 2)[0], func(t *testing.T) { fn(t, s) })
	}
}

// TestSingleDatabase: Miabi's argv (backup -d db, restore -d db -f file),
// gzip default, restore after data loss, the client chosen per flavour.
func TestSingleDatabase(t *testing.T) {
	forEachServer(t, func(t *testing.T, s *dbServer) {
		s.loadFixture(t, "shop")
		want := s.fingerprint(t, "shop")
		store := setEnv(t, s, "root", rootPW)

		out := mustRun(t, "backup", "-d", "shop")
		file := artifact(t, out, store)
		if !strings.HasSuffix(file, ".sql.gz") || !strings.HasPrefix(file, "shop_") {
			t.Fatalf("artifact %q", file)
		}
		wantClient := "client=mysqldump"
		if s.mariadb {
			wantClient = "client=mariadb-dump"
		}
		if !strings.Contains(out, wantClient) {
			t.Fatalf("expected %s:\n%s", wantClient, out)
		}
		assertTmpClean(t)

		s.mustSQL(t, "DROP DATABASE shop")
		mustRun(t, "restore", "-d", "shop", "-f", file)
		if got := s.fingerprint(t, "shop"); got != want {
			t.Fatalf("restored data differs:\n got %s\nwant %s", got, want)
		}
		assertTmpClean(t)

		// verify and list work without a database connection.
		mustRun(t, "verify", "-f", file)
		if out := mustRun(t, "list"); !strings.Contains(out, file) {
			t.Fatalf("list:\n%s", out)
		}
	})
}

// TestNonRootUser: a user with only database privileges (no PROCESS, no
// global SELECT) and a password that needs option-file quoting.
func TestNonRootUser(t *testing.T) {
	forEachServer(t, func(t *testing.T, s *dbServer) {
		s.mustSQL(t, fmt.Sprintf(`DROP USER IF EXISTS 'app'@'%%'; CREATE USER 'app'@'%%' IDENTIFIED BY '%s';
DROP DATABASE IF EXISTS appdb; CREATE DATABASE appdb; GRANT ALL ON appdb.* TO 'app'@'%%';
CREATE TABLE appdb.items (id INT PRIMARY KEY, v VARCHAR(20)); INSERT INTO appdb.items VALUES (1,'a'),(2,'b');`,
			strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(appPW)))
		store := setEnv(t, s, "app", appPW)
		out := mustRun(t, "backup", "-d", "appdb")
		file := artifact(t, out, store)
		s.mustSQL(t, "DELETE FROM appdb.items")
		mustRun(t, "restore", "-d", "appdb", "-f", file)
		if got := s.mustSQL(t, "SELECT COUNT(*) FROM appdb.items"); got != "2" {
			t.Fatalf("rows after restore: %s", got)
		}
	})
}

// TestEncryption: GPG passphrase and age recipients, restored under another name.
func TestEncryption(t *testing.T) {
	forEachServer(t, func(t *testing.T, s *dbServer) {
		s.loadFixture(t, "shop")
		want := s.fingerprint(t, "shop")
		store := setEnv(t, s, "root", rootPW)

		t.Run("gpg", func(t *testing.T) {
			t.Setenv("GPG_PASSPHRASE", "correct horse")
			out := mustRun(t, "backup", "-d", "shop")
			file := artifact(t, out, store)
			if !strings.HasSuffix(file, ".sql.gz.gpg") {
				t.Fatalf("artifact %q", file)
			}
			s.mustSQL(t, "DROP DATABASE IF EXISTS shop_gpg")
			mustRun(t, "restore", "-d", "shop_gpg", "-f", file)
			if got := s.fingerprint(t, "shop_gpg"); got != want {
				t.Fatalf("restored data differs:\n got %s\nwant %s", got, want)
			}
			t.Setenv("GPG_PASSPHRASE", "wrong")
			if out, code := run(t, "restore", "-d", "shop_gpg", "-f", file); code == 0 {
				t.Fatalf("restore with a wrong passphrase succeeded:\n%s", out)
			}
		})
		t.Run("age", func(t *testing.T) {
			id, err := age.GenerateX25519Identity()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("AGE_RECIPIENTS", id.Recipient().String())
			out := mustRun(t, "backup", "-d", "shop")
			m := regexp.MustCompile(`shop_[\w.\-]+\.sql\.gz\.age`).FindAllString(out, -1)
			if len(m) == 0 {
				t.Fatalf("no age artifact:\n%s", out)
			}
			file := m[len(m)-1]
			s.mustSQL(t, "DROP DATABASE IF EXISTS shop_age")
			t.Setenv("AGE_IDENTITY", id.String())
			mustRun(t, "restore", "-d", "shop_age", "-f", file)
			if got := s.fingerprint(t, "shop_age"); got != want {
				t.Fatalf("restored data differs:\n got %s\nwant %s", got, want)
			}
		})
		assertTmpClean(t)
	})
}

// TestAllDatabases: one file per database (-a, --exclude-db) and one file for
// everything (-A), restored without -d.
func TestAllDatabases(t *testing.T) {
	forEachServer(t, func(t *testing.T, s *dbServer) {
		s.mustSQL(t, "DROP DATABASE IF EXISTS shop_gpg; DROP DATABASE IF EXISTS shop_age; DROP DATABASE IF EXISTS appdb; DROP DATABASE IF EXISTS skipme; CREATE DATABASE skipme;")
		s.loadFixture(t, "shop")
		s.loadFixture(t, "blog")
		wantShop, wantBlog := s.fingerprint(t, "shop"), s.fingerprint(t, "blog")
		store := setEnv(t, s, "root", rootPW)

		out := mustRun(t, "backup", "-a", "--exclude-db", "skipme")
		ents, _ := os.ReadDir(store)
		seen := map[string]bool{}
		for _, e := range ents {
			if strings.HasSuffix(e.Name(), ".sql.gz") {
				seen[e.Name()[:strings.Index(e.Name(), "_2")]] = true
			}
		}
		if !seen["shop"] || !seen["blog"] || seen["skipme"] || seen["mysql"] || seen["sys"] {
			t.Fatalf("artifacts %v\n%s", seen, out)
		}
		artifact(t, out, store)

		out = mustRun(t, "backup", "-A")
		file := artifact(t, out, store)
		if !strings.HasPrefix(file, mysql.AllInOneName+"_") {
			t.Fatalf("all-in-one artifact %q", file)
		}
		s.mustSQL(t, "DROP DATABASE shop; DROP DATABASE blog")
		t.Setenv("DB_NAME", "")
		mustRun(t, "restore", "-f", file)
		if got := s.fingerprint(t, "shop"); got != wantShop {
			t.Fatalf("shop differs:\n got %s\nwant %s", got, wantShop)
		}
		if got := s.fingerprint(t, "blog"); got != wantBlog {
			t.Fatalf("blog differs:\n got %s\nwant %s", got, wantBlog)
		}
	})
}

// TestUploadsAndCorruption: Miabi uploads of plain .sql and gzip .sql.gz
// restore; corrupted inputs fail with a non-zero exit, and damaged or tampered
// archives are caught before the client runs, so the database is untouched.
func TestUploadsAndCorruption(t *testing.T) {
	forEachServer(t, func(t *testing.T, s *dbServer) {
		s.loadFixture(t, "shop")
		want := s.fingerprint(t, "shop")
		store := setEnv(t, s, "root", rootPW)
		out := mustRun(t, "backup", "-d", "shop", "--disable-compression")
		m := regexp.MustCompile(`shop_\d{8}_\d{6}(?:_\d+)?\.sql\b`).FindAllString(out, -1)
		if len(m) == 0 {
			t.Fatalf("no .sql artifact:\n%s", out)
		}
		plain, err := os.ReadFile(filepath.Join(store, m[len(m)-1]))
		if err != nil {
			t.Fatal(err)
		}
		write := func(name string, b []byte) {
			if err := os.WriteFile(filepath.Join(store, name), b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		write("upload.sql", plain)
		var gz bytes.Buffer
		zw := gzip.NewWriter(&gz)
		_, _ = zw.Write(plain)
		_ = zw.Close()
		write("upload.sql.gz", gz.Bytes())
		write("truncated.sql.gz", gz.Bytes()[:gz.Len()/2])
		write("bad.sql", append(append([]byte{}, plain...), []byte("\nTHIS IS NOT SQL;\n")...))

		// 1.x style absolute path: restore -f /backup/upload.sql.gz
		for _, f := range []string{"upload.sql", "upload.sql.gz", filepath.Join(store, "upload.sql.gz")} {
			s.mustSQL(t, "DROP DATABASE IF EXISTS shop_up")
			mustRun(t, "restore", "-d", "shop_up", "-f", f)
			if got := s.fingerprint(t, "shop_up"); got != want {
				t.Fatalf("%s: restored data differs", f)
			}
		}
		// A valid file with a bad statement cannot be caught before it runs.
		for _, f := range []string{"bad.sql", "missing.sql.gz"} {
			if out, code := run(t, "restore", "-d", "shop_bad", "-f", f); code == 0 {
				t.Fatalf("%s: restore succeeded:\n%s", f, out)
			}
		}
		// A damaged archive is verified first and never reaches the database.
		if out, code := run(t, "restore", "-d", "shop", "-f", "truncated.sql.gz"); code == 0 {
			t.Fatalf("truncated.sql.gz: restore succeeded:\n%s", out)
		}
		if got := s.fingerprint(t, "shop"); got != want {
			t.Fatal("truncated.sql.gz changed the database")
		}
		// Tampered GPG artifact.
		t.Setenv("GPG_PASSPHRASE", "pw")
		out = mustRun(t, "backup", "-d", "shop")
		file := artifact(t, out, store)
		b, _ := os.ReadFile(filepath.Join(store, file))
		b[len(b)/2] ^= 0xFF
		write("tampered.sql.gz.gpg", b)
		if out, code := run(t, "restore", "-d", "shop", "-f", "tampered.sql.gz.gpg"); code == 0 {
			t.Fatalf("tampered restore succeeded:\n%s", out)
		}
		if got := s.fingerprint(t, "shop"); got != want {
			t.Fatal("tampered.sql.gz.gpg changed the database")
		}
		assertTmpClean(t)
	})
}

// TestMiabiS3: Miabi's exact S3 argv and env against MinIO.
func TestMiabiS3(t *testing.T) {
	forEachServer(t, func(t *testing.T, s *dbServer) {
		s.loadFixture(t, "shop")
		want := s.fingerprint(t, "shop")
		setEnv(t, s, "root", rootPW)
		t.Setenv("AWS_S3_ENDPOINT", "http://127.0.0.1:"+minioPort)
		t.Setenv("AWS_S3_BUCKET_NAME", "backups")
		t.Setenv("AWS_REGION", "us-east-1")
		t.Setenv("AWS_ACCESS_KEY", "minioadmin")
		t.Setenv("AWS_SECRET_KEY", "minioadmin")
		t.Setenv("AWS_DISABLE_SSL", "true")
		t.Setenv("AWS_FORCE_PATH_STYLE", "true")
		t.Setenv("GPG_PASSPHRASE", "miabi")
		path := "miabi/" + strings.SplitN(s.name, "-", 3)[1]

		var out string
		for i := 0; i < 2; i++ { // two runs: older names are printed too
			out = mustRun(t, "backup", "--storage", "s3", "-d", "shop", "--path", path)
		}
		file := artifact(t, out, "")
		if !strings.HasSuffix(file, ".sql.gz.gpg") {
			t.Fatalf("artifact %q", file)
		}
		if o, err := docker("exec", minioContainer, "mc", "stat", "l/backups/"+path+"/"+file); err != nil {
			t.Fatalf("object %s/%s not in the bucket: %s", path, file, o)
		}
		s.mustSQL(t, "DROP DATABASE shop")
		mustRun(t, "restore", "--storage", "s3", "-d", "shop", "--path", path, "-f", file)
		if got := s.fingerprint(t, "shop"); got != want {
			t.Fatalf("restored data differs")
		}
	})
}

// TestMigrate: dump | restore between two servers, single database and -a.
func TestMigrate(t *testing.T) {
	needDocker(t)
	src := servers[0] // MySQL 8.4
	dst := servers[len(servers)-1]
	if dst.mariadb || dst == src {
		t.Skip("needs a second MySQL server (IT_SKIP_MYSQL80 is set)")
	}
	src.loadFixture(t, "shop")
	src.loadFixture(t, "blog")
	want := src.fingerprint(t, "shop")
	setEnv(t, src, "root", rootPW)
	t.Setenv("TARGET_DB_HOST", dst.name)
	t.Setenv("TARGET_DB_USERNAME", "root")
	t.Setenv("TARGET_DB_PASSWORD", rootPW)

	dst.mustSQL(t, "DROP DATABASE IF EXISTS shop_copy")
	t.Setenv("TARGET_DB_NAME", "shop_copy")
	mustRun(t, "migrate", "-d", "shop")
	if got := dst.fingerprint(t, "shop_copy"); got != want {
		t.Fatalf("migrated data differs:\n got %s\nwant %s", got, want)
	}

	t.Setenv("TARGET_DB_NAME", "")
	dst.mustSQL(t, "DROP DATABASE IF EXISTS shop; DROP DATABASE IF EXISTS blog")
	src.mustSQL(t, "DROP DATABASE IF EXISTS shop_gpg; DROP DATABASE IF EXISTS shop_age; DROP DATABASE IF EXISTS shop_up; DROP DATABASE IF EXISTS appdb; DROP DATABASE IF EXISTS skipme; DROP DATABASE IF EXISTS shop_bad")
	mustRun(t, "migrate", "-a")
	if got := dst.fingerprint(t, "blog"); got != src.fingerprint(t, "blog") {
		t.Fatalf("blog differs")
	}

	// A failing target aborts the migration with a non-zero exit.
	t.Setenv("TARGET_DB_PASSWORD", "wrong")
	if out, code := run(t, "migrate", "-d", "shop"); code == 0 {
		t.Fatalf("migrate with bad target credentials succeeded:\n%s", out)
	}
	assertTmpClean(t)
}

// TestMigrateMySQLToMariaDB: cross-flavour migrate (Oracle mysqldump | mariadb).
func TestMigrateMySQLToMariaDB(t *testing.T) {
	needDocker(t)
	var src, dst *dbServer
	for _, s := range servers {
		if s.mariadb {
			dst = s
		} else if src == nil {
			src = s
		}
	}
	src.mustSQL(t, "DROP DATABASE IF EXISTS plain; CREATE DATABASE plain; CREATE TABLE plain.t (id INT PRIMARY KEY, v VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci); INSERT INTO plain.t VALUES (1,'x'),(2,'y');")
	setEnv(t, src, "root", rootPW)
	t.Setenv("TARGET_DB_HOST", dst.name)
	t.Setenv("TARGET_DB_USERNAME", "root")
	t.Setenv("TARGET_DB_PASSWORD", rootPW)
	t.Setenv("TARGET_DB_NAME", "plain")
	dst.mustSQL(t, "DROP DATABASE IF EXISTS plain")
	mustRun(t, "migrate", "-d", "plain")
	if got := dst.mustSQL(t, "SELECT COUNT(*) FROM plain.t"); got != "2" {
		t.Fatalf("rows: %s", got)
	}
}
