package pkg

import (
	"fmt"
	"github.com/jkaninda/mysql-bkup/utils"
	"os"
	"os/exec"
	"path/filepath"
)

// Restore restore database
func Restore(file string) {
	dbHost = os.Getenv("DB_HOST")
	dbPassword = os.Getenv("DB_PASSWORD")
	dbUserName = os.Getenv("DB_USERNAME")
	dbName = os.Getenv("DB_NAME")
	dbPort = os.Getenv("DB_PORT")
	storagePath = os.Getenv("STORAGE_PATH")

	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_NAME") == "" || os.Getenv("DB_USERNAME") == "" || os.Getenv("DB_PASSWORD") == "" || file == "" {
		utils.Fatal("Please make sure all required environment variables are set")
	} else {

		if utils.FileExists(fmt.Sprintf("%s/%s", storagePath, file)) {
			utils.TestDatabaseConnection()

			extension := filepath.Ext(fmt.Sprintf("%s/%s", storagePath, file))
			// GZ compressed file
			if extension == ".gz" {
				str := "zcat " + fmt.Sprintf("%s/%s", storagePath, file) + " | mysql -h " + os.Getenv("DB_HOST") + " -P " + os.Getenv("DB_PORT") + " -u " + os.Getenv("DB_USERNAME") + " --password=" + os.Getenv("DB_PASSWORD") + " " + os.Getenv("DB_NAME")
				_, err := exec.Command("bash", "-c", str).Output()
				if err != nil {
					utils.Fatal("Error, in restoring the database")
				}

				utils.Info("Database has been restored")

			} else if extension == ".sql" {
				//SQL file
				str := "cat " + fmt.Sprintf("%s/%s", storagePath, file) + " | mysql -h " + os.Getenv("DB_HOST") + " -P " + os.Getenv("DB_PORT") + " -u " + os.Getenv("DB_USERNAME") + " --password=" + os.Getenv("DB_PASSWORD") + " " + os.Getenv("DB_NAME")
				_, err := exec.Command("bash", "-c", str).Output()
				if err != nil {
					utils.Fatal("Error, in restoring the database", err)
				}

				utils.Info("Database has been restored")
			} else {
				utils.Fatal("Unknown file extension ", extension)
			}

		} else {
			utils.Fatal("File not found in ", fmt.Sprintf("%s/%s", storagePath, file))
		}

	}
}
