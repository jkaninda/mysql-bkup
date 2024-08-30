package pkg

const cronLogFile = "/var/log/mysql-bkup.log"
const tmpPath = "/tmp/backup"
const backupCronFile = "/usr/local/bin/backup_cron.sh"
const algorithm = "aes256"
const gpgHome = "gnupg"
const gpgExtension = "gpg"

var (
	storage            = "local"
	file               = ""
	executionMode      = "default"
	storagePath        = "/backup"
	disableCompression = false
	encryption         = false
)

// dbHVars Required environment variables for database
var dbHVars = []string{
	"DB_HOST",
	"DB_PASSWORD",
	"DB_USERNAME",
	"DB_NAME",
}
var sdbRVars = []string{
	"SOURCE_DB_HOST",
	"SOURCE_DB_PORT",
	"SOURCE_DB_NAME",
	"SOURCE_DB_USERNAME",
	"SOURCE_DB_PASSWORD",
}

var dbConf *dbConfig
var sDbConf *dbSourceConfig

// sshHVars Required environment variables for SSH remote server storage
var sshHVars = []string{
	"SSH_USER",
	"SSH_REMOTE_PATH",
	"SSH_HOST_NAME",
	"SSH_PORT",
}
