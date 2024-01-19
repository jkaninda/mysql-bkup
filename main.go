package main

/*****
*   MySQL Backup & Restore
* @author    Jonas Kaninda
* @license   MIT License <https://opensource.org/licenses/MIT>
* @link      https://github.com/jkaninda/mysql-bkup
**/
import (
	"fmt"
	"github.com/jkaninda/mysql-bkup/pkg"
	"github.com/jkaninda/mysql-bkup/utils"
	flag "github.com/spf13/pflag"
	"os"
	"os/exec"
)

var appVersion string = os.Getenv("VERSION")

const s3MountPath string = "/s3mnt"

var (
	operation          string = "backup"
	storage            string = "local"
	file               string = ""
	s3Path             string = "/mysql-bkup"
	dbName             string = ""
	dbHost             string = ""
	dbPort             string = ""
	dbPassword         string = ""
	dbUserName         string = ""
	executionMode      string = "default"
	storagePath        string = "/backup"
	accessKey          string = ""
	secretKey          string = ""
	bucketName         string = ""
	s3Endpoint         string = ""
	s3fsPasswdFile     string = "/etc/passwd-s3fs"
	disableCompression bool   = false
	startBackup        bool   = true

	timeout int    = 30
	period  string = "0 1 * * *"
)

func init() {
	var (
		operationFlag          = flag.StringP("operation", "o", "backup", "Operation")
		storageFlag            = flag.StringP("storage", "s", "local", "Storage, local or s3")
		fileFlag               = flag.StringP("file", "f", "", "File name")
		pathFlag               = flag.StringP("path", "P", "/mysql-bkup", "S3 path, without file name")
		dbnameFlag             = flag.StringP("dbname", "d", "", "Database name")
		modeFlag               = flag.StringP("mode", "m", "default", "Execution mode. default or scheduled")
		periodFlag             = flag.StringP("period", "", "0 1 * * *", "Schedule period time")
		timeoutFlag            = flag.IntP("timeout", "t", 30, "Timeout (in seconds) to stop database connexion")
		disableCompressionFlag = flag.BoolP("disable-compression", "", false, "Disable backup compression")
		portFlag               = flag.IntP("port", "p", 3306, "Database port")
		helpFlag               = flag.BoolP("help", "h", false, "Print this help message")
		versionFlag            = flag.BoolP("version", "v", false, "Version information")
	)
	flag.Parse()

	operation = *operationFlag
	storage = *storageFlag
	file = *fileFlag
	s3Path = *pathFlag
	dbName = *dbnameFlag
	executionMode = *modeFlag
	dbPort = fmt.Sprint(*portFlag)
	timeout = *timeoutFlag
	period = *periodFlag
	disableCompression = *disableCompressionFlag

	flag.Usage = func() {
		fmt.Print("MySQL BackupDatabase and Restoration tool. BackupDatabase database to AWS S3 storage or any S3 Alternatives for Object Storage.\n\n")
		fmt.Print("Usage: bkup --operation backup -storage s3 --dbname databasename --path /my_path ...\n")
		fmt.Print("       bkup -o backup -d databasename --disable-compression ...\n")
		fmt.Print("       RestoreDatabase: bkup -o restore -d databasename -f db_20231217_051339.sql.gz ...\n\n")
		flag.PrintDefaults()
	}

	if *helpFlag {
		startBackup = false
		flag.Usage()
		os.Exit(0)
	}
	if *versionFlag {
		startBackup = false
		version()
		os.Exit(0)
	}
	if *dbnameFlag != "" {
		err := os.Setenv("DB_NAME", dbName)
		if err != nil {
			return
		}
	}
	if *pathFlag != "" {
		s3Path = *pathFlag
		err := os.Setenv("S3_PATH", fmt.Sprint(*pathFlag))
		if err != nil {
			return
		}

	}
	if *fileFlag != "" {
		file = *fileFlag
		err := os.Setenv("FILE_NAME", fmt.Sprint(*fileFlag))
		if err != nil {
			return
		}

	}
	if *portFlag != 3306 {
		err := os.Setenv("DB_PORT", fmt.Sprint(*portFlag))
		if err != nil {
			return
		}
	}
	if *periodFlag != "" {
		err := os.Setenv("SCHEDULE_PERIOD", fmt.Sprint(*periodFlag))
		if err != nil {
			return
		}
	}
	if *storageFlag != "" {
		err := os.Setenv("STORAGE", fmt.Sprint(*storageFlag))
		if err != nil {
			return
		}
	}
	storage = os.Getenv("STORAGE")
	err := os.Setenv("STORAGE_PATH", storagePath)
	if err != nil {
		return
	}

}

func version() {
	fmt.Printf("Version: %s \n", appVersion)
	fmt.Print()
}
func main() {
	//cmd.Execute()

	err := os.Setenv("STORAGE_PATH", storagePath)
	if err != nil {
		return
	}

	if startBackup {
		start()
	}

}
func start() {

	if executionMode == "default" {
		if operation != "backup" {
			if storage != "s3" {
				utils.Info("RestoreDatabase from local")
				pkg.RestoreDatabase(file)
			} else {
				utils.Info("RestoreDatabase from s3")
				s3Restore()
			}
		} else {
			if storage != "s3" {
				utils.Info("BackupDatabase to local storage")
				pkg.BackupDatabase(disableCompression)
			} else {
				utils.Info("BackupDatabase to s3 storage")
				s3Backup()
			}
		}
	} else if executionMode == "scheduled" {
		scheduledMode()
	} else {
		utils.Fatal("Error, unknown execution mode!")
	}
}

func s3Backup() {
	// Backup Database to S3 storage
	pkg.MountS3Storage(s3Path)
	pkg.BackupDatabase(disableCompression)
}

// Run in scheduled mode
func scheduledMode() {
	// Verify operation
	if operation == "backup" {

		fmt.Println()
		fmt.Println("**********************************")
		fmt.Println("     Starting MySQL Bkup...       ")
		fmt.Println("***********************************")
		utils.Info("Running in Scheduled mode")
		utils.Info("Log file in /var/log/mysql-bkup.log")
		utils.Info("Execution period ", os.Getenv("SCHEDULE_PERIOD"))

		//Test database connexion
		utils.TestDatabaseConnection()

		utils.Info("Creating backup job...")
		pkg.CreateCrontabScript(disableCompression, storage)

		supervisordCmd := exec.Command("supervisord", "-c", "/etc/supervisor/supervisord.conf")
		if err := supervisordCmd.Run(); err != nil {
			utils.Fatalf("Error starting supervisord: %v\n", err)
		}
	} else {
		utils.Fatal("Scheduled mode supports only backup operation")
	}
}

func s3Restore() {
	// Restore database from S3
	pkg.MountS3Storage(s3Path)
	pkg.RestoreDatabase(file)
}
