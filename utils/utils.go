package utils

/*****
*   MySQL Backup & Restore
* @author    Jonas Kaninda
* @license   MIT License <https://opensource.org/licenses/MIT>
* @link      https://github.com/jkaninda/mysql-bkup
**/
import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"io"
	"io/fs"
	"os"
	"os/exec"
)

func Info(msg string, args ...any) {
	if len(args) == 0 {
		slog.Info(msg)
	} else {
		slog.Info(fmt.Sprintf(msg, args...))
	}
}
func Worn(msg string, args ...any) {
	if len(args) == 0 {
		slog.Warn(msg)
	} else {
		slog.Warn(fmt.Sprintf(msg, args...))
	}
}
func Error(msg string, args ...any) {
	if len(args) == 0 {
		slog.Error(msg)
	} else {
		slog.Error(fmt.Sprintf(msg, args...))
	}
}
func Done(msg string, args ...any) {
	if len(args) == 0 {
		slog.Info(msg)
	} else {
		slog.Info(fmt.Sprintf(msg, args...))
	}
}
func Fatal(msg string, args ...any) {
	// Fatal logs an error message and exits the program.
	if len(args) == 0 {
		slog.Error(msg)
	} else {
		slog.Error(fmt.Sprintf(msg, args...))
	}
	os.Exit(1)
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func WriteToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}
func CopyFile(src, dst string) error {
	// Open the source file for reading
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destinationFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// Flush the buffer to ensure all data is written
	err = destinationFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %v", err)
	}

	return nil
}
func ChangePermission(filePath string, mod int) {
	if err := os.Chmod(filePath, fs.FileMode(mod)); err != nil {
		Fatal("Error changing permissions of %s: %v\n", filePath, err)
	}

}
func IsDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil
	}
	return true, nil
}

// TestDatabaseConnection  tests the database connection
func TestDatabaseConnection() {
	dbHost := os.Getenv("DB_HOST")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbUserName := os.Getenv("DB_USERNAME")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")

	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_NAME") == "" || os.Getenv("DB_USERNAME") == "" || os.Getenv("DB_PASSWORD") == "" {
		Fatal("Please make sure all required database environment variables are set")
	} else {
		Info("Connecting to database ...")

		cmd := exec.Command("mysql", "-h", dbHost, "-P", dbPort, "-u", dbUserName, "--password="+dbPassword, dbName, "-e", "quit")

		// Capture the output
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		if err != nil {
			slog.Error(fmt.Sprintf("Error testing database connection: %v\nOutput: %s\n", err, out.String()))
			os.Exit(1)

		}
		Info("Successfully connected to database")
	}
}
func GetEnv(cmd *cobra.Command, flagName, envName string) string {
	value, _ := cmd.Flags().GetString(flagName)
	if value != "" {
		err := os.Setenv(envName, value)
		if err != nil {
			return value
		}
	}
	return os.Getenv(envName)
}
func FlagGetString(cmd *cobra.Command, flagName string) string {
	value, _ := cmd.Flags().GetString(flagName)
	if value != "" {
		return value

	}
	return ""
}
func FlagGetBool(cmd *cobra.Command, flagName string) bool {
	value, _ := cmd.Flags().GetBool(flagName)
	return value
}

func SetEnv(key, value string) {

	err := os.Setenv(key, value)
	if err != nil {
		return
	}
}
func GetEnvVariable(envName, oldEnvName string) string {
	value := os.Getenv(envName)
	if value == "" {
		value = os.Getenv(oldEnvName)
		if value != "" {
			err := os.Setenv(envName, value)
			if err != nil {
				return value
			}
			Worn("%s is deprecated, please use %s instead!\n", oldEnvName, envName)

		}
	}
	return value
}
func ShowHistory() {
}

// CheckEnvVars checks if all the specified environment variables are set
func CheckEnvVars(vars []string) error {
	missingVars := []string{}

	for _, v := range vars {
		if os.Getenv(v) == "" {
			missingVars = append(missingVars, v)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing environment variables: %v", missingVars)
	}

	return nil
}
