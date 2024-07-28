package pkg

const s3MountPath string = "/s3mnt"
const s3fsPasswdFile string = "/etc/passwd-s3fs"
const cronLogFile = "/var/log/mysql-bkup.log"
const backupCronFile = "/usr/local/bin/backup_cron.sh"

var (
	storage            = "local"
	file               = ""
	s3Path             = "/mysql-bkup"
	dbName             = ""
	dbHost             = ""
	dbPort             = "3306"
	executionMode      = "default"
	storagePath        = "/backup"
	disableCompression = false
)
