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

package pkg

import (
	"bytes"
	"fmt"
	"github.com/jkaninda/encryptor"
	"github.com/jkaninda/go-storage/pkg/local"
	goutils "github.com/jkaninda/go-utils"
	"github.com/jkaninda/mysql-bkup/utils"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
			createBackupTask(dbConf, config)
		} else {
			if utils.IsValidCronExpression(config.cronExpression) {
				scheduledMode(dbConf, config)
			} else {
				utils.Fatal("Cron expression is not valid: %s", config.cronExpression)
			}
		}
	} else {
		startMultiBackup(config, configFile)
	}

}

// scheduledMode Runs backup in scheduled mode
func scheduledMode(db *dbConfig, config *BackupConfig) {
	utils.Info("Running in Scheduled mode")
	utils.Info("Backup cron expression:  %s", config.cronExpression)
	utils.Info("The next scheduled time is: %v", utils.CronNextTime(config.cronExpression).Format(timeFormat))
	utils.Info("Storage type %s ", config.storage)

	// Test backup
	utils.Info("Testing backup configurations...")
	err := testDatabaseConnection(db)
	if err != nil {
		utils.Error("Error connecting to database: %s", db.dbName)
		utils.Fatal("Error: %s", err)
	}
	utils.Info("Testing backup configurations...done")
	utils.Info("Creating backup job...")
	// Create a new cron instance
	c := cron.New()

	_, err = c.AddFunc(config.cronExpression, func() {
		createBackupTask(db, config)
		utils.Info("Next backup time is: %v", utils.CronNextTime(config.cronExpression).Format(timeFormat))

	})
	if err != nil {
		return
	}
	// Start the cron scheduler
	c.Start()
	utils.Info("Creating backup job...done")
	utils.Info("Backup job started")
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
		createBackupTask(getDatabase(db), bkConfig)
	}
}

// createBackupTask backup task
func createBackupTask(db *dbConfig, config *BackupConfig) {
	if config.all && !config.singleFile {
		backupAll(db, config)
	} else {
		backupTask(db, config)
	}
}

// backupAll backup all databases
func backupAll(db *dbConfig, config *BackupConfig) {
	databases, err := listDatabases(*db)
	if err != nil {
		utils.Fatal("Error listing databases: %s", err)
	}
	for _, dbName := range databases {
		if dbName == "information_schema" || dbName == "performance_schema" || dbName == "mysql" || dbName == "sys" || dbName == "innodb" || dbName == "Database" {
			continue
		}
		db.dbName = dbName
		config.backupFileName = fmt.Sprintf("%s_%s.sql.gz", dbName, time.Now().Format("20060102_150405"))
		backupTask(db, config)
	}

}

func backupTask(db *dbConfig, config *BackupConfig) {
	utils.Info("Starting backup task...")
	startTime = time.Now()
	prefix := db.dbName
	if config.all && config.singleFile {
		prefix = "all_databases"
	}
	// Generate file name
	backupFileName := fmt.Sprintf("%s_%s.sql.gz", prefix, time.Now().Format("20060102_150405"))
	if config.disableCompression {
		backupFileName = fmt.Sprintf("%s_%s.sql", prefix, time.Now().Format("20060102_150405"))
	}
	config.backupFileName = backupFileName
	switch config.storage {
	case "local":
		localBackup(db, config)
	case "s3", "S3":
		s3Backup(db, config)
	case "ssh", "SSH", "remote", "sftp":
		sshBackup(db, config)
	case "ftp", "FTP":
		ftpBackup(db, config)
	case "azure":
		azureBackup(db, config)
	default:
		localBackup(db, config)
	}
}
func startMultiBackup(bkConfig *BackupConfig, configFile string) {
	utils.Info("Starting Multi backup task...")
	conf, err := readConf(configFile)
	if err != nil {
		utils.Fatal("Error reading config file: %s", err)
	}
	// Check if cronExpression is defined in config file
	if conf.CronExpression != "" {
		bkConfig.cronExpression = conf.CronExpression
	}
	if len(conf.Databases) == 0 {
		utils.Fatal("No databases found")
	}
	// Check if cronExpression is defined
	if bkConfig.cronExpression == "" {
		multiBackupTask(conf.Databases, bkConfig)
	} else {
		backupRescueMode = conf.BackupRescueMode
		// Check if cronExpression is valid
		if utils.IsValidCronExpression(bkConfig.cronExpression) {
			utils.Info("Running backup in Scheduled mode")
			utils.Info("Backup cron expression:  %s", bkConfig.cronExpression)
			utils.Info("The next scheduled time is: %v", utils.CronNextTime(bkConfig.cronExpression).Format(timeFormat))
			utils.Info("Storage type %s ", bkConfig.storage)

			// Test backup
			utils.Info("Testing backup configurations...")
			for _, db := range conf.Databases {
				err = testDatabaseConnection(getDatabase(db))
				if err != nil {
					recoverMode(err, fmt.Sprintf("Error connecting to database: %s", db.Name))
					continue
				}
			}
			utils.Info("Testing backup configurations...done")
			utils.Info("Creating backup job...")
			// Create a new cron instance
			c := cron.New()

			_, err := c.AddFunc(bkConfig.cronExpression, func() {
				multiBackupTask(conf.Databases, bkConfig)
				utils.Info("Next backup time is: %v", utils.CronNextTime(bkConfig.cronExpression).Format(timeFormat))

			})
			if err != nil {
				return
			}
			// Start the cron scheduler
			c.Start()
			utils.Info("Creating backup job...done")
			utils.Info("Backup job started")
			defer c.Stop()
			select {}

		} else {
			utils.Fatal("Cron expression is not valid: %s", bkConfig.cronExpression)
		}
	}

}

// BackupDatabase backup database
func BackupDatabase(db *dbConfig, backupFileName string, disableCompression, all, singleFile bool) error {
	storagePath = os.Getenv("STORAGE_PATH")
	utils.Info("Starting database backup...")

	if err := testDatabaseConnection(db); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	dumpArgs := []string{fmt.Sprintf("--defaults-file=%s", mysqlClientConfig)}
	if all && singleFile {
		dumpArgs = append(dumpArgs, "--all-databases", "--single-transaction", "--routines", "--triggers")
	} else {
		dumpArgs = append(dumpArgs, db.dbName)
	}

	backupPath := filepath.Join(tmpPath, backupFileName)
	if disableCompression {
		return runCommandAndSaveOutput("mysqldump", dumpArgs, backupPath)
	}
	return runCommandWithCompression("mysqldump", dumpArgs, backupPath)
}

func runCommandAndSaveOutput(command string, args []string, outputPath string) error {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s: %v, output: %s", command, err, string(output))
	}

	return os.WriteFile(outputPath, output, 0644)
}

func runCommandWithCompression(command string, args []string, outputPath string) error {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	gzipCmd := exec.Command("gzip")
	gzipCmd.Stdin = stdout
	gzipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create gzip file: %w", err)
	}
	defer func(gzipFile *os.File) {
		err := gzipFile.Close()
		if err != nil {
			utils.Error("Error closing gzip file: %v", err)
		}
	}(gzipFile)
	gzipCmd.Stdout = gzipFile

	if err := gzipCmd.Start(); err != nil {
		return fmt.Errorf("failed to start gzip: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %w", command, err)
	}
	if err := gzipCmd.Wait(); err != nil {
		return fmt.Errorf("failed to wait for gzip completion: %w", err)
	}

	utils.Info("Database has been backed up")
	return nil
}
func localBackup(db *dbConfig, config *BackupConfig) {
	utils.Info("Backup database to local storage")
	err := BackupDatabase(db, config.backupFileName, disableCompression, config.all, config.singleFile)
	if err != nil {
		recoverMode(err, "Error backing up database")
		return
	}
	finalFileName := config.backupFileName
	if config.encryption {
		encryptBackup(config)
		finalFileName = fmt.Sprintf("%s.%s", config.backupFileName, gpgExtension)
	}
	fileInfo, err := os.Stat(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		utils.Error("Error: %s", err)
	}
	backupSize = fileInfo.Size()
	localStorage := local.NewStorage(local.Config{
		LocalPath:  tmpPath,
		RemotePath: storagePath,
	})
	err = localStorage.Copy(finalFileName)
	if err != nil {
		utils.Fatal("Error copying backup file: %s", err)
	}
	utils.Info("Backup name is %s", finalFileName)
	utils.Info("Backup size: %s", utils.ConvertBytes(uint64(backupSize)))
	utils.Info("Backup saved in %s", filepath.Join(storagePath, finalFileName))
	duration := goutils.FormatDuration(time.Since(startTime), 0)

	// Send notification
	utils.NotifySuccess(&utils.NotificationData{
		File:           finalFileName,
		BackupSize:     utils.ConvertBytes(uint64(backupSize)),
		Database:       db.dbName,
		Storage:        config.storage,
		BackupLocation: filepath.Join(storagePath, finalFileName),
		Duration:       duration,
	})
	// Delete old backup
	if config.prune {
		err = localStorage.Prune(config.backupRetention)
		if err != nil {
			utils.Fatal("Error deleting old backup from %s storage: %s ", config.storage, err)
		}

	}
	// Delete temp
	deleteTemp()
	utils.Info("Backup successfully completed in %s", duration)
}

func encryptBackup(config *BackupConfig) {
	backupFile, err := os.ReadFile(filepath.Join(tmpPath, config.backupFileName))
	outputFile := fmt.Sprintf("%s.%s", filepath.Join(tmpPath, config.backupFileName), gpgExtension)
	if err != nil {
		utils.Fatal("Error reading backup file: %s ", err)
	}
	if config.usingKey {
		utils.Info("Encrypting backup using public key...")
		pubKey, err := os.ReadFile(config.publicKey)
		if err != nil {
			utils.Fatal("Error reading public key: %s ", err)
		}
		err = encryptor.EncryptWithPublicKey(backupFile, fmt.Sprintf("%s.%s", filepath.Join(tmpPath, config.backupFileName), gpgExtension), pubKey)
		if err != nil {
			utils.Fatal("Error encrypting backup file: %v ", err)
		}
		utils.Info("Encrypting backup using public key...done")

	} else if config.passphrase != "" {
		utils.Info("Encrypting backup using passphrase...")
		err := encryptor.Encrypt(backupFile, outputFile, config.passphrase)
		if err != nil {
			utils.Fatal("error during encrypting backup %v", err)
		}
		utils.Info("Encrypting backup using passphrase...done")

	}

}

// listDatabases list all databases
func listDatabases(db dbConfig) ([]string, error) {
	databases := []string{}
	// Create the mysql client config file
	if err := createMysqlClientConfigFile(db); err != nil {
		return databases, fmt.Errorf(err.Error())
	}
	utils.Info("Listing databases...")
	// Step 1: List all databases
	cmd := exec.Command("mariadb", fmt.Sprintf("--defaults-file=%s", mysqlClientConfig), "-e", "SHOW DATABASES;")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return databases, fmt.Errorf("failed to list databases: %s", err)
	}
	// Step 2: Parse the output
	for _, _db := range strings.Split(out.String(), "\n") {
		if _db != "" {
			databases = append(databases, _db)
		}
	}
	return databases, nil
}
func recoverMode(err error, msg string) {
	if err != nil {
		if backupRescueMode {
			utils.NotifyError(fmt.Sprintf("%s : %v", msg, err))
			utils.Error("Error: %s", msg)
			utils.Error("Backup rescue mode is enabled")
			utils.Error("Backup will continue")
		} else {
			utils.Error("Error: %s", msg)
			utils.Fatal("Error: %v", err)
			return
		}
	}

}
