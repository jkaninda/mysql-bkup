// Package pkg /
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package pkg

import (
	"fmt"
	"github.com/jkaninda/mysql-bkup/utils"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func StartBackup(cmd *cobra.Command) {
	intro()
	dbConf = initDbConfig(cmd)
	//Initialize backup configs
	config := initBackupConfig(cmd)

	if config.cronExpression == "" {
		BackupTask(dbConf, config)
	} else {
		if utils.IsValidCronExpression(config.cronExpression) {
			scheduledMode(dbConf, config)
		} else {
			utils.Fatal("Cron expression is not valid: %s", config.cronExpression)
		}
	}

}

// Run in scheduled mode
func scheduledMode(db *dbConfig, config *BackupConfig) {
	utils.Info("Running in Scheduled mode")
	utils.Info("Backup cron expression:  %s", os.Getenv("BACKUP_CRON_EXPRESSION"))
	utils.Info("Storage type %s ", storage)

	//Test database connexion
	testDatabaseConnection(db)

	utils.Info("Creating a new cron instance...")
	// Create a new cron instance
	c := cron.New()

	// Add a cron job that runs every 10 seconds
	c.AddFunc(config.cronExpression, func() {
		BackupTask(db, config)
	})
	// Start the cron scheduler
	c.Start()
	utils.Info("Creating a new cron instance...done")
	defer c.Stop()
	select {}
}
func BackupTask(db *dbConfig, config *BackupConfig) {
	utils.Info("Starting backup task...")
	switch config.storage {
	case "s3":
		s3Backup(db, config.backupFileName, config.disableCompression, config.prune, config.backupRetention, config.encryption)
	case "local":
		localBackup(db, config.backupFileName, config.disableCompression, config.prune, config.backupRetention, config.encryption)
	case "ssh", "remote":
		sshBackup(db, config.backupFileName, config.remotePath, config.disableCompression, config.prune, config.backupRetention, config.encryption)
	case "ftp":
		utils.Fatal("Not supported storage type: %s", config.storage)
	default:
		localBackup(db, config.backupFileName, config.disableCompression, config.prune, config.backupRetention, config.encryption)
	}
}

// BackupDatabase backup database
func BackupDatabase(db *dbConfig, backupFileName string, disableCompression bool) {
	storagePath = os.Getenv("STORAGE_PATH")

	err := utils.CheckEnvVars(dbHVars)
	if err != nil {
		utils.Error("Please make sure all required environment variables for database are set")
		utils.Fatal("Error checking environment variables: %s", err)
	}

	utils.Info("Starting database backup...")
	err = os.Setenv("MYSQL_PWD", db.dbPassword)
	if err != nil {
		return
	}
	testDatabaseConnection(db)

	// Backup Database database
	utils.Info("Backing up database...")

	if disableCompression {
		// Execute mysqldump
		cmd := exec.Command("mysqldump",
			"-h", db.dbHost,
			"-P", db.dbPort,
			"-u", db.dbUserName,
			db.dbName,
		)
		output, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}

		// save output
		file, err := os.Create(fmt.Sprintf("%s/%s", tmpPath, backupFileName))
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
		// Execute mysqldump
		cmd := exec.Command("mysqldump", "-h", db.dbHost, "-P", db.dbPort, "-u", db.dbUserName, db.dbName)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		gzipCmd := exec.Command("gzip")
		gzipCmd.Stdin = stdout
		gzipCmd.Stdout, err = os.Create(fmt.Sprintf("%s/%s", tmpPath, backupFileName))
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

}
func localBackup(db *dbConfig, backupFileName string, disableCompression bool, prune bool, backupRetention int, encrypt bool) {
	utils.Info("Backup database to local storage")
	BackupDatabase(db, backupFileName, disableCompression)
	finalFileName := backupFileName
	if encrypt {
		encryptBackup(backupFileName)
		finalFileName = fmt.Sprintf("%s.%s", backupFileName, gpgExtension)
	}
	utils.Info("Backup name is %s", finalFileName)
	moveToBackup(finalFileName, storagePath)
	//Send notification
	utils.NotifySuccess(finalFileName)
	//Delete old backup
	if prune {
		deleteOldBackup(backupRetention)
	}
	//Delete temp
	deleteTemp()
}

func s3Backup(db *dbConfig, backupFileName string, disableCompression bool, prune bool, backupRetention int, encrypt bool) {
	bucket := utils.GetEnvVariable("AWS_S3_BUCKET_NAME", "BUCKET_NAME")
	s3Path := utils.GetEnvVariable("AWS_S3_PATH", "S3_PATH")
	utils.Info("Backup database to s3 storage")
	//Backup database
	BackupDatabase(db, backupFileName, disableCompression)
	finalFileName := backupFileName
	if encrypt {
		encryptBackup(backupFileName)
		finalFileName = fmt.Sprintf("%s.%s", backupFileName, "gpg")
	}
	utils.Info("Uploading backup archive to remote storage S3 ... ")
	utils.Info("Backup name is %s", finalFileName)
	err := utils.UploadFileToS3(tmpPath, finalFileName, bucket, s3Path)
	if err != nil {
		utils.Fatal("Error uploading file to S3: %s ", err)

	}

	//Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, backupFileName))
	if err != nil {
		fmt.Println("Error deleting file: ", err)

	}
	// Delete old backup
	if prune {
		err := utils.DeleteOldBackup(bucket, s3Path, backupRetention)
		if err != nil {
			utils.Fatal("Error deleting old backup from S3: %s ", err)
		}
	}
	utils.Done("Uploading backup archive to remote storage S3 ... done ")
	//Send notification
	utils.NotifySuccess(finalFileName)
	//Delete temp
	deleteTemp()
}

// sshBackup backup database to SSH remote server
func sshBackup(db *dbConfig, backupFileName, remotePath string, disableCompression bool, prune bool, backupRetention int, encrypt bool) {
	utils.Info("Backup database to Remote server")
	//Backup database
	BackupDatabase(db, backupFileName, disableCompression)
	finalFileName := backupFileName
	if encrypt {
		encryptBackup(backupFileName)
		finalFileName = fmt.Sprintf("%s.%s", backupFileName, "gpg")
	}
	utils.Info("Uploading backup archive to remote storage ... ")
	utils.Info("Backup name is %s", finalFileName)
	err := CopyToRemote(finalFileName, remotePath)
	if err != nil {
		utils.Fatal("Error uploading file to the remote server: %s ", err)

	}

	//Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		fmt.Println("Error deleting file: ", err)

	}
	if prune {
		//TODO: Delete old backup from remote server
		utils.Info("Deleting old backup from a remote server is not implemented yet")

	}

	utils.Done("Uploading backup archive to remote storage ... done ")
	//Send notification
	utils.NotifySuccess(finalFileName)
	//Delete temp
	deleteTemp()
}

// encryptBackup encrypt backup
func encryptBackup(backupFileName string) {
	gpgPassphrase := os.Getenv("GPG_PASSPHRASE")
	err := Encrypt(filepath.Join(tmpPath, backupFileName), gpgPassphrase)
	if err != nil {
		utils.Fatal("Error during encrypting backup %s", err)
	}

}
