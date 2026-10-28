package mysql

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jkaninda/bkupkit/config"
)

// TLS modes accepted in DB_SSL_MODE (MySQL --ssl-mode names, lower-case).
const (
	SSLDisabled       = "disabled"
	SSLPreferred      = "preferred"
	SSLRequired       = "required"
	SSLVerifyCA       = "verify_ca"
	SSLVerifyIdentity = "verify_identity"
)

// ParseSSLMode normalizes DB_SSL_MODE. mysql-bkup 1.x wrote the value into
// my.cnf as "ssl=<value>", so "0"/"1" (and booleans) are still accepted.
// An empty value means disabled, the 1.x default.
func ParseSSLMode(s string) (string, error) {
	m := strings.ToLower(strings.TrimSpace(s))
	m = strings.ReplaceAll(m, "-", "_")
	switch m {
	case "", "0", "false", "off", "no", SSLDisabled, "disable":
		return SSLDisabled, nil
	case "1", "true", "on", "yes", SSLRequired, "require":
		return SSLRequired, nil
	case SSLPreferred, "prefer":
		return SSLPreferred, nil
	case SSLVerifyCA:
		return SSLVerifyCA, nil
	case SSLVerifyIdentity, "verify_full":
		return SSLVerifyIdentity, nil
	}
	return "", fmt.Errorf("invalid TLS mode %q: use disabled, preferred, required, verify_ca or verify_identity", s)
}

// Conn is a resolved connection.
type Conn struct {
	Host     string
	Port     int
	User     string
	Password string
	DB       string
	SSLMode  string
	SSLCA    string
	SSLCert  string
	SSLKey   string
}

// Addr is host:port (or the socket path), for logs and cache keys.
func (c Conn) Addr() string {
	if strings.HasPrefix(c.Host, "/") {
		return c.Host
	}
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// ParseURL parses mysql://user:pass@host:port/db?ssl-mode=required (also
// mariadb://). Query parameters: ssl-mode (sslmode, tls), ssl-ca, ssl-cert, ssl-key.
func ParseURL(raw string) (Conn, error) {
	var c Conn
	u, err := url.Parse(raw)
	if err != nil {
		return c, errors.New("invalid database URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "mysql", "mariadb", "mysqlx":
	default:
		return c, fmt.Errorf("unsupported database URL scheme %q (use mysql://user:password@host:port/database)", u.Scheme)
	}
	c.Host = u.Hostname()
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 65535 {
			return c, fmt.Errorf("invalid port %q in database URL", p)
		}
		c.Port = n
	}
	if u.User != nil {
		c.User = u.User.Username()
		c.Password, _ = u.User.Password()
	}
	c.DB = strings.TrimPrefix(u.Path, "/")
	q := u.Query()
	for _, k := range []string{"ssl-mode", "sslmode", "ssl_mode", "tls"} {
		if v := q.Get(k); v != "" {
			if c.SSLMode, err = ParseSSLMode(v); err != nil {
				return c, err
			}
			break
		}
	}
	c.SSLCA = first(q.Get("ssl-ca"), q.Get("sslrootcert"))
	c.SSLCert = first(q.Get("ssl-cert"), q.Get("sslcert"))
	c.SSLKey = first(q.Get("ssl-key"), q.Get("sslkey"))
	return c, nil
}

func first(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// connKeys names the settings of one side (source or migrate target).
type connKeys struct {
	URL, Host, Port, Name, User, Pass, SSLMode, SSLCA, SSLCert, SSLKey string
}

var (
	sourceKeys = connKeys{config.DBURL, config.DBHost, config.DBPort, config.DBName, config.DBUser, config.DBPassword,
		config.DBSSLMode, config.DBSSLCA, config.DBSSLCert, config.DBSSLKey}
	targetKeys = connKeys{config.TargetDBURL, config.TargetDBHost, config.TargetDBPort, config.TargetDBName, config.TargetDBUser,
		config.TargetDBPass, config.TargetDBSSL, config.TargetDBSSLCA, "", ""}
)

// connFrom resolves a connection: DB_URL first, then the individual settings
// (which also fill in what the URL leaves out).
func connFrom(v *config.Values, k connKeys) (Conn, error) {
	var c Conn
	if raw := v.String(k.URL); raw != "" {
		var err error
		if c, err = ParseURL(raw); err != nil {
			return c, fmt.Errorf("%s: %w", k.URL, err)
		}
	}
	str := func(dst *string, key string) {
		if *dst == "" && key != "" {
			*dst = v.String(key)
		}
	}
	str(&c.Host, k.Host)
	str(&c.User, k.User)
	str(&c.Password, k.Pass)
	str(&c.DB, k.Name)
	str(&c.SSLCA, k.SSLCA)
	str(&c.SSLCert, k.SSLCert)
	str(&c.SSLKey, k.SSLKey)
	if c.Port == 0 {
		c.Port = v.Int(k.Port)
	}
	if c.Port == 0 {
		c.Port = 3306
	}
	if c.SSLMode == "" {
		m, err := ParseSSLMode(v.String(k.SSLMode))
		if err != nil {
			return c, fmt.Errorf("%s: %w", k.SSLMode, err)
		}
		c.SSLMode = m
	}
	return c, nil
}

// SourceConn returns the DB_* connection of v.
func SourceConn(v *config.Values) (Conn, error) { return connFrom(v, sourceKeys) }

// TargetConn returns the TARGET_DB_* connection of v (migrate).
func TargetConn(v *config.Values) (Conn, error) {
	c, err := connFrom(v, targetKeys)
	if err != nil {
		return c, err
	}
	if v.String(config.TargetDBURL) == "" && v.String(config.TargetDBHost) == "" {
		return c, errors.New("missing required configuration: TARGET_DB_HOST (or TARGET_DB_URL)")
	}
	return c, nil
}

func (c Conn) validate(prefix string) error {
	var missing []string
	if c.Host == "" {
		missing = append(missing, prefix+"DB_HOST")
	}
	if c.User == "" {
		missing = append(missing, prefix+"DB_USER")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

// checkDBName rejects names that a client would parse as an option.
func checkDBName(name string) error {
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid database name %q", name)
	}
	return nil
}
