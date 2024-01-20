// Package cmd /*
/*
Copyright © 2024 Jonas Kaninda  <jonaskaninda@gmail.com>
*/
package cmd

import (
	"fmt"
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
	//TODO: To remove
	//For old user || To remove
	Run: func(cmd *cobra.Command, args []string) {
		if operation != "" {
			if operation == "backup" || operation == "restore" {
				fmt.Println(utils.Notice)
				utils.Fatal("New config required, please check --help")
			}
		}
	},
}
var operation = ""
var s3Path = "/mysql-bkup"

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("storage", "s", "local", "Set storage. local or s3")
	rootCmd.PersistentFlags().StringP("path", "P", s3Path, "Set s3 path, without file name. for S3 storage only")
	rootCmd.PersistentFlags().StringP("dbname", "d", "", "Set database name")
	rootCmd.PersistentFlags().IntP("timeout", "t", 30, "Set timeout")
	rootCmd.PersistentFlags().IntP("port", "p", 3306, "Set database port")
	rootCmd.PersistentFlags().StringVarP(&operation, "operation", "o", "", "Set operation, for old version only")

	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(BackupCmd)
	rootCmd.AddCommand(RestoreCmd)
	rootCmd.AddCommand(S3MountCmd)
	rootCmd.AddCommand(HistoryCmd)
}
