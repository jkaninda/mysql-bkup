// Package cmd /*
/*
Copyright © 2024 Jonas Kaninda
*/
package cmd

import (
	"github.com/jkaninda/mysql-bkup/utils"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "mysql-bkup [Command]",
	Short:   "MySQL Backup tool, backup database to S3 or Object Storage",
	Long:    `MySQL Database backup and restoration tool. Backup database to AWS S3 storage or any S3 Alternatives for Object Storage.`,
	Example: utils.MainExample,
	Version: appVersion,
}
var operation = ""

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("dbname", "d", "", "Database name")
	rootCmd.PersistentFlags().IntP("port", "p", 3306, "Database port")
	rootCmd.PersistentFlags().StringVarP(&operation, "operation", "o", "", "Set operation, for old version only")
	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(BackupCmd)
	rootCmd.AddCommand(RestoreCmd)
	rootCmd.AddCommand(MigrateCmd)

}
