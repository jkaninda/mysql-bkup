// Package pkg /*
/*
Copyright © 2024 Jonas Kaninda  <jonaskaninda.gmail.com>
*/
package pkg

import (
	"fmt"
	"github.com/jkaninda/mysql-bkup/utils"
	"log"
	"os"
	"os/exec"
	"time"
)

var (
	dbName      = ""
	dbHost      = ""
	dbPort      = ""
	dbPassword  = ""
	dbUserName  = ""
	storagePath = "/backup"
)

// BackupDatabase backup database
func BackupDatabase(disableCompression bool) {
	dbHost = os.Getenv("DB_HOST")
	dbPassword = os.Getenv("DB_PASSWORD")
	dbUserName = os.Getenv("DB_USERNAME")
	dbName = os.Getenv("DB_NAME")
	dbPort = os.Getenv("DB_PORT")
	storagePath = os.Getenv("STORAGE_PATH")

	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_NAME") == "" || os.Getenv("DB_USERNAME") == "" || os.Getenv("DB_PASSWORD") == "" {
		utils.Fatal("Please make sure all required environment variables for database are set")
	} else {
		utils.TestDatabaseConnection()
		// Backup Database database
		utils.Info("Backing up database...")
		bkFileName := fmt.Sprintf("%s_%s.sql.gz", dbName, time.Now().Format("20060102_150405"))

		if disableCompression {
			bkFileName = fmt.Sprintf("%s_%s.sql", dbName, time.Now().Format("20060102_150405"))
			cmd := exec.Command("mysqldump",
				"-h", dbHost,
				"-P", dbPort,
				"-u", dbUserName,
				"--password="+dbPassword,
				dbName,
			)
			output, err := cmd.Output()
			if err != nil {
				log.Fatal(err)
			}

			file, err := os.Create(fmt.Sprintf("%s/%s", storagePath, bkFileName))
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			_, err = file.Write(output)
			if err != nil {
				log.Fatal(err)
			}
			utils.Done("Database has been backed up")

		} else {
			cmd := exec.Command("mysqldump", "-h", dbHost, "-P", dbPort, "-u", dbUserName, "--password="+dbPassword, dbName)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Fatal(err)
			}
			gzipCmd := exec.Command("gzip")
			gzipCmd.Stdin = stdout
			gzipCmd.Stdout, err = os.Create(fmt.Sprintf("%s/%s", storagePath, bkFileName))
			gzipCmd.Start()
			if err != nil {
				log.Fatal(err)
			}
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
			if err := gzipCmd.Wait(); err != nil {
				log.Fatal(err)
			}
			utils.Done("Database has been backed up")

		}

		historyFile, err := os.OpenFile(fmt.Sprintf("%s/history.txt", storagePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer historyFile.Close()
		if _, err := historyFile.WriteString(bkFileName + "\n"); err != nil {
			log.Fatal(err)
		}
	}

}
