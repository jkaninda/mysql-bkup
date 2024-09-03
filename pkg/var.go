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
var tdbRVars = []string{
	"TARGET_DB_HOST",
	"TARGET_DB_PORT",
	"TARGET_DB_NAME",
	"TARGET_DB_USERNAME",
	"TARGET_DB_PASSWORD",
}

var dbConf *dbConfig
var targetDbConf *targetDbConfig

// sshHVars Required environment variables for SSH remote server storage
var sshHVars = []string{
	"SSH_USER",
	"SSH_REMOTE_PATH",
	"SSH_HOST_NAME",
	"SSH_PORT",
}
