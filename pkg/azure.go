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
	"fmt"
	"github.com/jkaninda/go-storage/pkg/azure"
	"github.com/jkaninda/mysql-bkup/utils"

	"os"
	"path/filepath"
	"time"
)

func azureBackup(db *dbConfig, config *BackupConfig) {
	utils.Info("Backup database to Azure Blob Storage")
	startTime = time.Now().Format(utils.TimeFormat())

	// Backup database
	BackupDatabase(db, config.backupFileName, disableCompression)
	finalFileName := config.backupFileName
	if config.encryption {
		encryptBackup(config)
		finalFileName = fmt.Sprintf("%s.%s", config.backupFileName, "gpg")
	}
	utils.Info("Uploading backup archive to Azure Blob storage ...")
	utils.Info("Backup name is %s", finalFileName)
	azureConfig := loadAzureConfig()
	azureStorage, err := azure.NewStorage(azure.Config{
		ContainerName: azureConfig.containerName,
		AccountName:   azureConfig.accountName,
		AccountKey:    azureConfig.accountKey,
		RemotePath:    config.remotePath,
		LocalPath:     tmpPath,
	})
	if err != nil {
		utils.Fatal("Error creating Azure storage: %s", err)
	}
	err = azureStorage.Copy(finalFileName)
	if err != nil {
		utils.Fatal("Error copying backup file: %s", err)
	}
	utils.Info("Backup saved in %s", filepath.Join(config.remotePath, finalFileName))
	// Get backup info
	fileInfo, err := os.Stat(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		utils.Error("Error: %s", err)
	}
	backupSize = fileInfo.Size()
	// Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, finalFileName))
	if err != nil {
		utils.Error("Error deleting file: %v", err)

	}
	if config.prune {
		err := azureStorage.Prune(config.backupRetention)
		if err != nil {
			utils.Fatal("Error deleting old backup from %s storage: %s ", config.storage, err)
		}

	}

	utils.Info("Backup name is %s", finalFileName)
	utils.Info("Backup size: %s", utils.ConvertBytes(uint64(backupSize)))
	utils.Info("Uploading backup archive to Azure Blob storage ... done ")

	// Send notification
	utils.NotifySuccess(&utils.NotificationData{
		File:           finalFileName,
		BackupSize:     utils.ConvertBytes(uint64(backupSize)),
		Database:       db.dbName,
		Storage:        config.storage,
		BackupLocation: filepath.Join(config.remotePath, finalFileName),
		StartTime:      startTime,
		EndTime:        time.Now().Format(utils.TimeFormat()),
	})
	// Delete temp
	deleteTemp()
	utils.Info("Backup completed successfully")
}
func azureRestore(db *dbConfig, conf *RestoreConfig) {
	utils.Info("Restore database from Azure Blob storage")
	azureConfig := loadAzureConfig()
	azureStorage, err := azure.NewStorage(azure.Config{
		ContainerName: azureConfig.containerName,
		AccountName:   azureConfig.accountName,
		AccountKey:    azureConfig.accountKey,
		RemotePath:    conf.remotePath,
		LocalPath:     tmpPath,
	})
	if err != nil {
		utils.Fatal("Error creating SSH storage: %s", err)
	}

	err = azureStorage.CopyFrom(conf.file)
	if err != nil {
		utils.Fatal("Error downloading backup file: %s", err)
	}
	RestoreDatabase(db, conf)
}
