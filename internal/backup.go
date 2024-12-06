// Package internal /
/*
MIT License

Copyright (c) 2023 Jonas Kaninda

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package internal

import (
	"fmt"
	"github.com/jkaninda/encryptor"
	"github.com/jkaninda/go-storage/pkg/ftp"
	"github.com/jkaninda/go-storage/pkg/local"
	"github.com/jkaninda/go-storage/pkg/s3"
	"github.com/jkaninda/go-storage/pkg/ssh"
	"github.com/jkaninda/mysql-bkup/pkg/logger"
	"github.com/jkaninda/mysql-bkup/utils"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func StartBackup(cmd *cobra.Command) {
	intro()
	// Initialize backup configs
	config := initBackupConfig(cmd)
	// Load backup configuration file
	configFile, err := loadConfigFile()
	if err != nil {
		dbConf = initDbConfig(cmd)
		if config.cronExpression == "" {
			BackupTask(dbConf, config)
		} else {
			if utils.IsValidCronExpression(config.cronExpression) {
				scheduledMode(dbConf, config)
			} else {
				logger.Fatal("Cron expression is not valid: %s", config.cronExpression)
			}
		}
	} else {
		startMultiBackup(config, configFile)
	}

}

// scheduledMode Runs backup in scheduled mode
func scheduledMode(db *dbConfig, config *BackupConfig) {
	logger.Info("Running in Scheduled mode")
	logger.Info("Backup cron expression:  %s", config.cronExpression)
	logger.Info("The next scheduled time is: %v", utils.CronNextTime(config.cronExpression).Format(timeFormat))
	logger.Info("Storage type %s ", config.storage)

	// Test backup
	logger.Info("Testing backup configurations...")
	testDatabaseConnection(db)
	logger.Info("Testing backup configurations...done")
	logger.Info("Creating backup job...")
	// Create a new cron instance
	c := cron.New()

	_, err := c.AddFunc(config.cronExpression, func() {
		BackupTask(db, config)
		logger.Info("Next backup time is: %v", utils.CronNextTime(config.cronExpression).Format(timeFormat))

	})
	if err != nil {
		return
	}
	// Start the cron scheduler
	c.Start()
	logger.Info("Creating backup job...done")
	logger.Info("Backup job started")
	defer c.Stop()
	select {}
}

// multiBackupTask backup multi database
func multiBackupTask(databases []Database, bkConfig *BackupConfig) {
	for _, db := range databases {
		// Check if path is defined in config file
		if db.Path != "" {
			bkConfig.remotePath = db.Path
		}
		BackupTask(getDatabase(db), bkConfig)
	}
}

// BackupTask backups database
func BackupTask(db *dbConfig, config *BackupConfig) {
	logger.Info("Starting backup task...")
	// Generate file name
	backupFileName := fmt.Sprintf("%s_%s.sql.gz", db.dbName, time.Now().Format("20060102_150405"))
	if config.disableCompression {
		backupFileName = fmt.Sprintf("%s_%s.sql", db.dbName, time.Now().Format("20060102_150405"))
	}
	config.backupFileName = backupFileName
	switch config.storage {
	case "local":
		localBackup(db, config)
	case "s3", "S3":
		s3Backup(db, config)
	case "ssh", "SSH", "remote":
		sshBackup(db, config)
	case "ftp", "FTP":
		ftpBackup(db, config)
	default:
		localBackup(db, config)
	}
}
func startMultiBackup(bkConfig *BackupConfig, configFile string) {
	logger.Info("Starting backup task...")
	conf, err := readConf(configFile)
	if err != nil {
		logger.Fatal("Error reading config file: %s", err)
	}
	// Check if cronExpression is defined in config file
	if conf.CronExpression != "" {
		bkConfig.cronExpression = conf.CronExpression
	}
	if len(conf.Databases) == 0 {
		logger.Fatal("No databases found")
	}
	// Check if cronExpression is defined
	if bkConfig.cronExpression == "" {
		multiBackupTask(conf.Databases, bkConfig)
	} else {
		// Check if cronExpression is valid
		if utils.IsValidCronExpression(bkConfig.cronExpression) {
			logger.Info("Running backup in Scheduled mode")
			logger.Info("Backup cron expression:  %s", bkConfig.cronExpression)
			logger.Info("The next scheduled time is: %v", utils.CronNextTime(bkConfig.cronExpression).Format(timeFormat))
			logger.Info("Storage type %s ", bkConfig.storage)

			// Test backup
			logger.Info("Testing backup configurations...")
			for _, db := range conf.Databases {
				testDatabaseConnection(getDatabase(db))
			}
			logger.Info("Testing backup configurations...done")
			logger.Info("Creating backup job...")
			// Create a new cron instance
			c := cron.New()

			_, err := c.AddFunc(bkConfig.cronExpression, func() {
				multiBackupTask(conf.Databases, bkConfig)
				logger.Info("Next backup time is: %v", utils.CronNextTime(bkConfig.cronExpression).Format(timeFormat))

			})
			if err != nil {
				return
			}
			// Start the cron scheduler
			c.Start()
			logger.Info("Creating backup job...done")
			logger.Info("Backup job started")
			defer c.Stop()
			select {}

		} else {
			logger.Fatal("Cron expression is not valid: %s", bkConfig.cronExpression)
		}
	}

}

// BackupDatabase backup database
func BackupDatabase(db *dbConfig, backupFileName string, disableCompression bool) {
	storagePath = os.Getenv("STORAGE_PATH")

	logger.Info("Starting database backup...")

	err := os.Setenv("MYSQL_PWD", db.dbPassword)
	if err != nil {
		return
	}
	testDatabaseConnection(db)
	// Backup Database database
	logger.Info("Backing up database...")

	// Verify is compression is disabled
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
			logger.Fatal(err.Error())
		}

		// save output
		file, err := os.Create(filepath.Join(tmpPath, backupFileName))
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				logger.Fatal(err.Error())
			}
		}(file)

		_, err = file.Write(output)
		if err != nil {
			logger.Fatal(err.Error())
		}
		logger.Info("Database has been backed up")

	} else {
		// Execute mysqldump
		cmd := exec.Command("mysqldump", "-h", db.dbHost, "-P", db.dbPort, "-u", db.dbUserName, db.dbName)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		gzipCmd := exec.Command("gzip")
		gzipCmd.Stdin = stdout
		gzipCmd.Stdout, err = os.Create(filepath.Join(tmpPath, backupFileName))
		err = gzipCmd.Start()
		if err != nil {
			return
		}
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
		if err := gzipCmd.Wait(); err != nil {
			log.Fatal(err)
		}
		logger.Info("Database has been backed up")

	}
}
func localBackup(db *dbConfig, config *BackupConfig) {
	logger.Info("Backup database to local storage")
	startTime = time.Now().Format(utils.TimeFormat())
	BackupDatabase(db, config.backupFileName, disableCompression)
	finalFileName := config.backupFileName
	if config.encryption {
		encryptBackup(config)
		finalFileName = fmt.Sprintf("%s.%s", config.backupFileName, gpgExtension)
	}
	fileInfo, err := os.Stat(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		logger.Error("Error: %s", err)
	}
	backupSize = fileInfo.Size()
	logger.Info("Backup name is %s", finalFileName)
	localStorage := local.NewStorage(local.Config{
		LocalPath:  tmpPath,
		RemotePath: storagePath,
	})
	err = localStorage.Copy(finalFileName)
	if err != nil {
		logger.Fatal("Error copying backup file: %s", err)
	}
	logger.Info("Backup saved in %s", filepath.Join(storagePath, finalFileName))
	// Send notification
	utils.NotifySuccess(&utils.NotificationData{
		File:           finalFileName,
		BackupSize:     backupSize,
		Database:       db.dbName,
		Storage:        config.storage,
		BackupLocation: filepath.Join(storagePath, finalFileName),
		StartTime:      startTime,
		EndTime:        time.Now().Format(utils.TimeFormat()),
	})
	// Delete old backup
	if config.prune {
		err = localStorage.Prune(config.backupRetention)
		if err != nil {
			logger.Fatal("Error deleting old backup from %s storage: %s ", config.storage, err)
		}

	}
	// Delete temp
	deleteTemp()
	logger.Info("Backup completed successfully")
}

func s3Backup(db *dbConfig, config *BackupConfig) {

	logger.Info("Backup database to s3 storage")
	startTime = time.Now().Format(utils.TimeFormat())
	// Backup database
	BackupDatabase(db, config.backupFileName, disableCompression)
	finalFileName := config.backupFileName
	if config.encryption {
		encryptBackup(config)
		finalFileName = fmt.Sprintf("%s.%s", config.backupFileName, "gpg")
	}
	logger.Info("Uploading backup archive to remote storage S3 ... ")
	awsConfig := initAWSConfig()
	if config.remotePath == "" {
		config.remotePath = awsConfig.remotePath
	}
	logger.Info("Backup name is %s", finalFileName)
	s3Storage, err := s3.NewStorage(s3.Config{
		Endpoint:       awsConfig.endpoint,
		Bucket:         awsConfig.bucket,
		AccessKey:      awsConfig.accessKey,
		SecretKey:      awsConfig.secretKey,
		Region:         awsConfig.region,
		DisableSsl:     awsConfig.disableSsl,
		ForcePathStyle: awsConfig.forcePathStyle,
		RemotePath:     awsConfig.remotePath,
		LocalPath:      tmpPath,
	})
	if err != nil {
		logger.Fatal("Error creating s3 storage: %s", err)
	}
	err = s3Storage.Copy(finalFileName)
	if err != nil {
		logger.Fatal("Error copying backup file: %s", err)
	}
	// Get backup info
	fileInfo, err := os.Stat(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		logger.Error("Error: %s", err)
	}
	backupSize = fileInfo.Size()

	// Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, config.backupFileName))
	if err != nil {
		fmt.Println("Error deleting file: ", err)

	}
	// Delete old backup
	if config.prune {
		err := s3Storage.Prune(config.backupRetention)
		if err != nil {
			logger.Fatal("Error deleting old backup from %s storage: %s ", config.storage, err)
		}
	}
	logger.Info("Backup saved in %s", filepath.Join(config.remotePath, finalFileName))
	logger.Info("Uploading backup archive to remote storage S3 ... done ")
	// Send notification
	utils.NotifySuccess(&utils.NotificationData{
		File:           finalFileName,
		BackupSize:     backupSize,
		Database:       db.dbName,
		Storage:        config.storage,
		BackupLocation: filepath.Join(config.remotePath, finalFileName),
		StartTime:      startTime,
		EndTime:        time.Now().Format(utils.TimeFormat()),
	})
	// Delete temp
	deleteTemp()
	logger.Info("Backup completed successfully")

}
func sshBackup(db *dbConfig, config *BackupConfig) {
	logger.Info("Backup database to Remote server")
	startTime = time.Now().Format(utils.TimeFormat())
	// Backup database
	BackupDatabase(db, config.backupFileName, disableCompression)
	finalFileName := config.backupFileName
	if config.encryption {
		encryptBackup(config)
		finalFileName = fmt.Sprintf("%s.%s", config.backupFileName, "gpg")
	}
	logger.Info("Uploading backup archive to remote storage ... ")
	logger.Info("Backup name is %s", finalFileName)
	sshConfig, err := loadSSHConfig()
	if err != nil {
		logger.Fatal("Error loading ssh config: %s", err)
	}

	sshStorage, err := ssh.NewStorage(ssh.Config{
		Host:       sshConfig.hostName,
		Port:       sshConfig.port,
		User:       sshConfig.user,
		Password:   sshConfig.password,
		RemotePath: config.remotePath,
		LocalPath:  tmpPath,
	})
	if err != nil {
		logger.Fatal("Error creating SSH storage: %s", err)
	}
	err = sshStorage.Copy(finalFileName)
	if err != nil {
		logger.Fatal("Error copying backup file: %s", err)
	}
	// Get backup info
	fileInfo, err := os.Stat(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		logger.Error("Error: %s", err)
	}
	backupSize = fileInfo.Size()
	logger.Info("Backup saved in %s", filepath.Join(config.remotePath, finalFileName))

	// Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		logger.Error("Error deleting file: %v", err)

	}
	if config.prune {
		err := sshStorage.Prune(config.backupRetention)
		if err != nil {
			logger.Fatal("Error deleting old backup from %s storage: %s ", config.storage, err)
		}

	}
	logger.Info("Uploading backup archive to remote storage ... done ")
	// Send notification
	utils.NotifySuccess(&utils.NotificationData{
		File:           finalFileName,
		BackupSize:     backupSize,
		Database:       db.dbName,
		Storage:        config.storage,
		BackupLocation: filepath.Join(config.remotePath, finalFileName),
		StartTime:      startTime,
		EndTime:        time.Now().Format(utils.TimeFormat()),
	})
	// Delete temp
	deleteTemp()
	logger.Info("Backup completed successfully")

}
func ftpBackup(db *dbConfig, config *BackupConfig) {
	logger.Info("Backup database to the remote FTP server")
	startTime = time.Now().Format(utils.TimeFormat())

	// Backup database
	BackupDatabase(db, config.backupFileName, disableCompression)
	finalFileName := config.backupFileName
	if config.encryption {
		encryptBackup(config)
		finalFileName = fmt.Sprintf("%s.%s", config.backupFileName, "gpg")
	}
	logger.Info("Uploading backup archive to the remote FTP server ... ")
	logger.Info("Backup name is %s", finalFileName)
	ftpConfig := loadFtpConfig()
	ftpStorage, err := ftp.NewStorage(ftp.Config{
		Host:       ftpConfig.host,
		Port:       ftpConfig.port,
		User:       ftpConfig.user,
		Password:   ftpConfig.password,
		RemotePath: config.remotePath,
		LocalPath:  tmpPath,
	})
	if err != nil {
		logger.Fatal("Error creating SSH storage: %s", err)
	}
	err = ftpStorage.Copy(finalFileName)
	if err != nil {
		logger.Fatal("Error copying backup file: %s", err)
	}
	logger.Info("Backup saved in %s", filepath.Join(config.remotePath, finalFileName))
	// Get backup info
	fileInfo, err := os.Stat(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		logger.Error("Error: %s", err)
	}
	backupSize = fileInfo.Size()
	// Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		logger.Error("Error deleting file: %v", err)

	}
	if config.prune {
		err := ftpStorage.Prune(config.backupRetention)
		if err != nil {
			logger.Fatal("Error deleting old backup from %s storage: %s ", config.storage, err)
		}

	}

	logger.Info("Uploading backup archive to the remote FTP server ... done ")

	// Send notification
	utils.NotifySuccess(&utils.NotificationData{
		File:           finalFileName,
		BackupSize:     backupSize,
		Database:       db.dbName,
		Storage:        config.storage,
		BackupLocation: filepath.Join(config.remotePath, finalFileName),
		StartTime:      startTime,
		EndTime:        time.Now().Format(utils.TimeFormat()),
	})
	// Delete temp
	deleteTemp()
	logger.Info("Backup completed successfully")
}

func encryptBackup(config *BackupConfig) {
	backupFile, err := os.ReadFile(filepath.Join(tmpPath, config.backupFileName))
	outputFile := fmt.Sprintf("%s.%s", filepath.Join(tmpPath, config.backupFileName), gpgExtension)
	if err != nil {
		logger.Fatal("Error reading backup file: %s ", err)
	}
	if config.usingKey {
		logger.Info("Encrypting backup using public key...")
		pubKey, err := os.ReadFile(config.publicKey)
		if err != nil {
			logger.Fatal("Error reading public key: %s ", err)
		}
		err = encryptor.EncryptWithPublicKey(backupFile, fmt.Sprintf("%s.%s", filepath.Join(tmpPath, config.backupFileName), gpgExtension), pubKey)
		if err != nil {
			logger.Fatal("Error encrypting backup file: %v ", err)
		}
		logger.Info("Encrypting backup using public key...done")

	} else if config.passphrase != "" {
		logger.Info("Encrypting backup using passphrase...")
		err := encryptor.Encrypt(backupFile, outputFile, config.passphrase)
		if err != nil {
			logger.Fatal("error during encrypting backup %v", err)
		}
		logger.Info("Encrypting backup using passphrase...done")

	}

}
