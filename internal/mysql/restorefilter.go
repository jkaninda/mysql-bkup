package mysql

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
)

// FilterOptions say how RestoreFilter adapts a dump.
type FilterOptions struct {
	// Target is the database the dump is restored into ("" for an
	// all-in-one restore, which is left alone).
	Target string
	// NoDefiner removes DEFINER clauses.
	NoDefiner bool
	// OracleClient: Oracle's mysql client replays the dump.
	OracleClient bool
	// MySQLServer: the dump is restored into MySQL (not MariaDB).
	MySQLServer bool
}

// RestoreFilter adapts a dump on its way to the client.
//
// With a Target (a restore into a named database), the dump's CREATE
// DATABASE, DROP DATABASE and USE statements are dropped, so a dump taken with
// --databases (or written by another tool) lands in target whatever its source
// database was called. A dump naming two databases cannot go into one and
// fails the restore.
//
// With NoDefiner, DEFINER clauses are removed from CREATE statements, so
// views, triggers, routines and events are defined by the restoring user: only
// a user with SET_USER_ID (SUPER) may name another one, and the source
// accounts usually do not exist on the target anyway.
//
// With OracleClient, the MariaDB sandbox line that mariadb-dump writes
// first ("/*M!999999\- enable the sandbox mode */") is dropped: Oracle's mysql
// client rejects it, so a MariaDB dump could not be restored into MySQL. The
// MariaDB client keeps it (it disables client shell commands for the dump).
//
// With MySQLServer, the utf8mb4_uca1400 collations MariaDB 11 uses by default
// become their MySQL equivalents (utf8mb4_uca1400_ai_ci -> utf8mb4_0900_ai_ci),
// outside data lines, and sql_mode values keep only the modes MySQL 8 knows
// (NO_AUTO_CREATE_USER from MySQL 5.7 and MariaDB, MariaDB-only modes go), so
// MariaDB and older MySQL dumps restore into MySQL 8.
//
// Lines are kept (an empty line replaces a dropped statement), so the client's
// error line numbers still match the dump.
func RestoreFilter(r io.Reader, o FilterOptions) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		f := &restoreFilter{FilterOptions: o, delimiter: ";"}
		err := f.run(r, pw)
		if f.retargeted > 0 || f.definers > 0 || f.collations > 0 || f.sqlModes > 0 {
			slog.Default().Info("Restore adapted to the target", "source_database", f.source, "database", o.Target,
				"database_statements_skipped", f.retargeted, "definers_removed", f.definers,
				"collations_mapped", f.collations, "sql_modes_adjusted", f.sqlModes)
		}
		_ = pw.CloseWithError(err)
	}()
	return pr
}

const ident = "(`(?:[^`]|``)+`|'(?:[^']|'')+'|\"(?:[^\"]|\"\")+\"|[\\w$]+)"

var (
	useRe       = regexp.MustCompile(`(?i)^USE\s+` + ident + `\s*;?$`)
	createDBRe  = regexp.MustCompile(`(?i)^CREATE\s+(?:DATABASE|SCHEMA)\s*(?:/\*!\d*\s*IF\s+NOT\s+EXISTS\s*\*/|IF\s+NOT\s+EXISTS)?\s*` + ident + `[^;]*;$`)
	dropDBRe    = regexp.MustCompile(`(?i)^(?:/\*!\d*\s*)?DROP\s+(?:DATABASE|SCHEMA)\s+(?:IF\s+EXISTS\s+)?` + ident + `\s*(?:\*/)?\s*;$`)
	delimiterRe = regexp.MustCompile(`(?i)^DELIMITER\s+(\S+)`)
	sandboxRe   = regexp.MustCompile(`^/\*M!999999\\- enable the sandbox mode \*/\s*$`)
	definerPart = "(?:`(?:[^`]|``)*`|'(?:[^']|'')*'|\"(?:[^\"]|\"\")*\"|[\\w.$%-]+)"
	// MariaDB collations with an exact MySQL 8 equivalent.
	uca1400Re = regexp.MustCompile(`\butf8mb4_uca1400_(ai_ci|as_ci|as_cs)\b`)
	dataRe    = regexp.MustCompile(`(?i)^(?:INSERT|REPLACE)\s`)
	sqlModeRe = regexp.MustCompile(`(?i)(\bsql_mode\s*=\s*')([^']*)(')`)
	definerRe = regexp.MustCompile(`(?i)\s*DEFINER\s*=\s*(?:CURRENT_USER(?:\s*\(\s*\))?|` + definerPart + `\s*@\s*` + definerPart + `|` + definerPart + `)`)
)

const lineBuf = 64 * 1024

type restoreFilter struct {
	FilterOptions
	delimiter string
	source    string // the database the dump selects, once seen

	retargeted, definers, collations, sqlModes int
}

func (f *restoreFilter) run(r io.Reader, w io.Writer) error {
	br := bufio.NewReaderSize(r, lineBuf)
	partial := false
	for {
		line, err := br.ReadSlice('\n')
		if len(line) > 0 {
			out := line
			if !partial && !errors.Is(err, bufio.ErrBufferFull) {
				// Long lines are data (extended INSERTs): never rewritten.
				var ferr error
				if out, ferr = f.line(line); ferr != nil {
					return ferr
				}
			}
			if _, werr := w.Write(out); werr != nil {
				return werr
			}
		}
		switch {
		case errors.Is(err, bufio.ErrBufferFull):
			partial = true
		case err == nil:
			partial = false
		case errors.Is(err, io.EOF):
			return nil
		default:
			return err
		}
	}
}

func (f *restoreFilter) line(b []byte) ([]byte, error) {
	text := strings.TrimSpace(string(b))
	nl := b[len(strings.TrimRight(string(b), "\r\n")):]
	if f.OracleClient && sandboxRe.MatchString(text) {
		return nl, nil
	}
	if m := delimiterRe.FindStringSubmatch(text); m != nil {
		f.delimiter = m[1]
		return b, nil
	}
	if f.Target != "" && f.delimiter == ";" {
		for _, re := range []*regexp.Regexp{useRe, createDBRe, dropDBRe} {
			m := re.FindStringSubmatch(text)
			if m == nil {
				continue
			}
			name := unquoteIdent(m[1])
			switch {
			case f.source == "":
				f.source = name
			case name != f.source:
				return nil, fmt.Errorf("the dump holds more than one database (%s, %s): restore it without a database name, or one database at a time", f.source, name)
			}
			f.retargeted++
			return nl, nil
		}
	}
	out := strings.TrimRight(string(b), "\r\n")
	changed := false
	if f.NoDefiner && definesObject(text) {
		if n := len(definerRe.FindAllStringIndex(out, -1)); n > 0 {
			f.definers += n
			out, changed = definerRe.ReplaceAllString(out, ""), true
		}
	}
	if f.MySQLServer && !dataRe.MatchString(text) {
		if n := len(uca1400Re.FindAllStringIndex(out, -1)); n > 0 {
			f.collations += n
			out, changed = uca1400Re.ReplaceAllString(out, "utf8mb4_0900_$1"), true
		}
		if sqlModeRe.MatchString(out) {
			fixed := sqlModeRe.ReplaceAllStringFunc(out, func(m string) string {
				p := sqlModeRe.FindStringSubmatch(m)
				return p[1] + mysqlModes(p[2]) + p[3]
			})
			if fixed != out {
				f.sqlModes++
				out, changed = fixed, true
			}
		}
	}
	if !changed {
		return b, nil
	}
	return append([]byte(out), nl...), nil
}

// definesObject reports whether a line can carry a DEFINER clause: a CREATE
// statement, or mysqldump's versioned comments around one.
func definesObject(s string) bool {
	if len(s) < 6 {
		return false
	}
	head := strings.ToUpper(s[:6])
	return (head == "CREATE" || strings.HasPrefix(head, "/*!")) && strings.Contains(strings.ToUpper(s), "DEFINER")
}

func unquoteIdent(s string) string {
	if len(s) >= 2 {
		switch q := s[0]; q {
		case '`', '\'', '"':
			if s[len(s)-1] == q {
				return strings.ReplaceAll(s[1:len(s)-1], string([]byte{q, q}), string(q))
			}
		}
	}
	return s
}

// mysql8Modes are the sql_mode values MySQL 8 accepts.
var mysql8Modes = map[string]bool{
	"ALLOW_INVALID_DATES": true, "ANSI_QUOTES": true, "ERROR_FOR_DIVISION_BY_ZERO": true, "HIGH_NOT_PRECEDENCE": true,
	"IGNORE_SPACE": true, "NO_AUTO_VALUE_ON_ZERO": true, "NO_BACKSLASH_ESCAPES": true, "NO_DIR_IN_CREATE": true,
	"NO_ENGINE_SUBSTITUTION": true, "NO_UNSIGNED_SUBTRACTION": true, "NO_ZERO_DATE": true, "NO_ZERO_IN_DATE": true,
	"ONLY_FULL_GROUP_BY": true, "PAD_CHAR_TO_FULL_LENGTH": true, "PIPES_AS_CONCAT": true, "REAL_AS_FLOAT": true,
	"STRICT_ALL_TABLES": true, "STRICT_TRANS_TABLES": true, "TIME_TRUNCATE_FRACTIONAL": true,
	"ANSI": true, "TRADITIONAL": true,
}

// mysqlModes keeps the modes of a comma-separated sql_mode MySQL 8 knows.
func mysqlModes(modes string) string {
	var kept []string
	for _, m := range strings.Split(modes, ",") {
		if m = strings.TrimSpace(m); m != "" && mysql8Modes[strings.ToUpper(m)] {
			kept = append(kept, m)
		}
	}
	return strings.Join(kept, ",")
}
