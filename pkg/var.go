// Package pkg /
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package pkg

const cronLogFile = "/var/log/mysql-bkup.log"
const tmpPath = "/tmp/backup"
const algorithm = "aes256"
const gpgHome = "/config/gnupg"
const gpgExtension = "gpg"

var (
	storage            = "local"
	file               = ""
	storagePath        = "/backup"
	disableCompression = false
	encryption         = false
	usingKey           = false
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
	"REMOTE_PATH",
	"SSH_HOST_NAME",
	"SSH_PORT",
}
var ftpVars = []string{
	"FTP_HOST_NAME",
	"FTP_USER",
	"FTP_PASSWORD",
	"FTP_PORT",
}

// AwsVars Required environment variables for AWS S3 storage
var awsVars = []string{
	"AWS_S3_ENDPOINT",
	"AWS_S3_BUCKET_NAME",
	"AWS_ACCESS_KEY",
	"AWS_SECRET_KEY",
	"AWS_REGION",
}
