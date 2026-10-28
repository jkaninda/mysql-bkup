package mysql

import (
	"strconv"
	"strings"
)

// quoteOption quotes a value for a MySQL/MariaDB option file. Both client
// families parse option files with the same mysys code: a value wrapped in
// double quotes keeps leading/trailing spaces and '#', and the escapes
// \\ \" \n \r \t are decoded inside it.
func quoteOption(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// optionFile returns the [client] option file for c. mariadbClient selects
// the MariaDB client option names (ssl, ssl-verify-server-cert) instead of
// the Oracle MySQL ones (ssl-mode, get-server-public-key).
//
// The file holds the password; it is written 0600 in a private directory and
// passed as --defaults-extra-file, never on the command line.
func optionFile(c Conn, mariadbClient bool) string {
	var b strings.Builder
	line := func(k, v string) {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
		b.WriteByte('\n')
	}
	b.WriteString("[client]\n")
	if strings.HasPrefix(c.Host, "/") {
		line("protocol", "SOCKET")
		line("socket", quoteOption(c.Host))
	} else {
		// TCP even for "localhost" (a client would otherwise look for a socket).
		line("protocol", "TCP")
		line("host", quoteOption(c.Host))
		line("port", strconv.Itoa(c.Port))
	}
	line("user", quoteOption(c.User))
	line("password", quoteOption(c.Password))
	line("max-allowed-packet", "1G")

	verify := c.SSLMode == SSLVerifyCA || c.SSLMode == SSLVerifyIdentity
	if mariadbClient {
		switch c.SSLMode {
		case SSLDisabled:
			line("ssl", "0")
		case SSLPreferred:
			// Leave TLS to the client default, without certificate checks.
			line("ssl-verify-server-cert", "0")
		case SSLRequired:
			line("ssl", "1")
			line("ssl-verify-server-cert", "0")
		default: // verify_ca, verify_identity: MariaDB always checks the host name too
			line("ssl", "1")
			line("ssl-verify-server-cert", "1")
		}
	} else {
		line("ssl-mode", strings.ToUpper(c.SSLMode))
		if c.SSLMode != SSLVerifyIdentity && c.SSLMode != SSLVerifyCA {
			// caching_sha2_password over a non-TLS connection needs the
			// server's RSA key (what the MariaDB client does by itself).
			line("get-server-public-key", "1")
		}
	}
	if c.SSLMode != SSLDisabled {
		// The CA is only used when certificates are verified: with the MySQL
		// client, a CA would otherwise silently turn "required" into "verify_ca".
		if verify && c.SSLCA != "" {
			line("ssl-ca", quoteOption(c.SSLCA))
		}
		if c.SSLCert != "" {
			line("ssl-cert", quoteOption(c.SSLCert))
		}
		if c.SSLKey != "" {
			line("ssl-key", quoteOption(c.SSLKey))
		}
	}
	return b.String()
}
