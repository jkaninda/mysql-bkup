package pkg

const s3MountPath string = "/s3mnt"
const s3fsPasswdFile string = "/etc/passwd-s3fs"

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
