// Package pkg /
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package pkg

import (
	"bytes"
	"fmt"
	"github.com/jkaninda/mysql-bkup/utils"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func copyToTmp(sourcePath string, backupFileName string) {
	//Copy backup from storage to /tmp
	err := utils.CopyFile(filepath.Join(sourcePath, backupFileName), filepath.Join(tmpPath, backupFileName))
	if err != nil {
		utils.Fatal(fmt.Sprintf("Error copying file %s %s", backupFileName, err))

	}
}
func moveToBackup(backupFileName string, destinationPath string) {
	//Copy backup from tmp folder to storage destination
	err := utils.CopyFile(filepath.Join(tmpPath, backupFileName), filepath.Join(destinationPath, backupFileName))
	if err != nil {
		utils.Fatal(fmt.Sprintf("Error copying file %s %s", backupFileName, err))

	}
	//Delete backup file from tmp folder
	err = utils.DeleteFile(filepath.Join(tmpPath, backupFileName))
	if err != nil {
		fmt.Println("Error deleting file:", err)

	}
	utils.Done("Database has been backed up and copied to %s", filepath.Join(destinationPath, backupFileName))
}
func deleteOldBackup(retentionDays int) {
	utils.Info("Deleting old backups...")
	storagePath = os.Getenv("STORAGE_PATH")
	// Define the directory path
	backupDir := storagePath + "/"
	// Get current time
	currentTime := time.Now()
	// Delete file
	deleteFile := func(filePath string) error {
		err := os.Remove(filePath)
		if err != nil {
			utils.Fatal(fmt.Sprintf("Error: %s", err))
		} else {
			utils.Done("File  %s  has been deleted successfully", filePath)
		}
		return err
	}

	// Walk through the directory and delete files modified more than specified days ago
	err := filepath.Walk(backupDir, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if it's a regular file and if it was modified more than specified days ago
		if fileInfo.Mode().IsRegular() {
			timeDiff := currentTime.Sub(fileInfo.ModTime())
			if timeDiff.Hours() > 24*float64(retentionDays) {
				err := deleteFile(filePath)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		utils.Fatal(fmt.Sprintf("Error: %s", err))
		return
	}
	utils.Done("Deleting old backups...done")

}
func deleteTemp() {
	utils.Info("Deleting %s ...", tmpPath)
	err := filepath.Walk(tmpPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the current item is a file
		if !info.IsDir() {
			// Delete the file
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.Error("Error deleting files: %v", err)
	} else {
		utils.Info("Deleting %s ... done", tmpPath)
	}
}

// TestDatabaseConnection  tests the database connection
func testDatabaseConnection(db *dbConfig) {
	err := os.Setenv("MYSQL_PWD", db.dbPassword)
	if err != nil {
		return
	}
	utils.Info("Connecting to %s database ...", db.dbName)
	cmd := exec.Command("mysql", "-h", db.dbHost, "-P", db.dbPort, "-u", db.dbUserName, db.dbName, "-e", "quit")
	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		utils.Fatal("Error testing database connection: %v\nOutput: %s", err, out.String())

	}
	utils.Info("Successfully connected to %s database", db.dbName)

}
func intro() {
	utils.Info("Starting MySQL Backup...")
	utils.Info("Copyright (c) 2024 Jonas Kaninda ")
}
func checkPubKeyFile(pubKey string) (string, error) {
	// Define possible key file names
	keyFiles := []string{filepath.Join(gpgHome, "public_key.asc"), filepath.Join(gpgHome, "public_key.gpg"), pubKey}

	// Loop through key file names and check if they exist
	for _, keyFile := range keyFiles {
		if _, err := os.Stat(keyFile); err == nil {
			// File exists
			return keyFile, nil
		} else if os.IsNotExist(err) {
			// File does not exist, continue to the next one
			continue
		} else {
			// An unexpected error occurred
			return "", err
		}
	}

	// Return an error if neither file exists
	return "", fmt.Errorf("no public key file found")
}
func checkPrKeyFile(prKey string) (string, error) {
	// Define possible key file names
	keyFiles := []string{filepath.Join(gpgHome, "private_key.asc"), filepath.Join(gpgHome, "private_key.gpg"), prKey}

	// Loop through key file names and check if they exist
	for _, keyFile := range keyFiles {
		if _, err := os.Stat(keyFile); err == nil {
			// File exists
			return keyFile, nil
		} else if os.IsNotExist(err) {
			// File does not exist, continue to the next one
			continue
		} else {
			// An unexpected error occurred
			return "", err
		}
	}

	// Return an error if neither file exists
	return "", fmt.Errorf("no public key file found")
}
func readConf(configFile string) (*Config, error) {
	//configFile := filepath.Join("./", filename)
	if utils.FileExists(configFile) {
		buf, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		c := &Config{}
		err = yaml.Unmarshal(buf, c)
		if err != nil {
			return nil, fmt.Errorf("in file %q: %w", configFile, err)
		}

		return c, err
	}
	return nil, fmt.Errorf("config file %q not found", configFile)
}
func checkConfigFile(filePath string) (string, error) {
	// Define possible config file names
	configFiles := []string{filepath.Join(workingDir, "config.yaml"), filepath.Join(workingDir, "config.yml"), filePath}

	// Loop through config file names and check if they exist
	for _, configFile := range configFiles {
		if _, err := os.Stat(configFile); err == nil {
			// File exists
			return configFile, nil
		} else if os.IsNotExist(err) {
			// File does not exist, continue to the next one
			continue
		} else {
			// An unexpected error occurred
			return "", err
		}
	}

	// Return an error if neither file exists
	return "", fmt.Errorf("no config file found")
}
