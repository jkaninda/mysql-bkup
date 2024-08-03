package pkg

const cronLogFile = "/var/log/mysql-bkup.log"
const tmpPath = "/tmp/backup"
const backupCronFile = "/usr/local/bin/backup_cron.sh"
const algorithm = "aes256"
const gpgExtension = "gpg"

var (
	storage            = "local"
	file               = ""
	dbPassword         = ""
	dbUserName         = ""
	dbName             = ""
	dbHost             = ""
	dbPort             = "3306"
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

// sshHVars Required environment variables for SSH remote server storage
var sshHVars = []string{
	"SSH_USER",
	"SSH_REMOTE_PATH",
	"SSH_HOST_NAME",
	"SSH_PORT",
}
