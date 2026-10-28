// Command mysql-bkup backs up and restores MySQL and MariaDB databases to
// local storage, S3, GCS, Azure Blob, SFTP or FTP, with optional GPG or age
// encryption. See https://github.com/jkaninda/mysql-bkup.
package main

import (
	"github.com/jkaninda/bkupkit/cli"
	"github.com/jkaninda/mysql-bkup/internal/mysql"
)

var Version = "development"

func main() {
	(&cli.App{
		Engine:  mysql.New(),
		Version: Version,
		Short:   "Back up and restore MySQL and MariaDB databases (local, S3, GCS, Azure, SFTP, FTP; GPG or age encryption)",
	}).Main()
}
