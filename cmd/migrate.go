// Package cmd /
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package cmd

import (
	"github.com/jkaninda/mysql-bkup/pkg"
	"github.com/jkaninda/mysql-bkup/utils"
	"github.com/spf13/cobra"
)

var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate database from a source database to a target database",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			pkg.StartMigration(cmd)
		} else {
			utils.Fatal(`"migrate" accepts no argument %q`, args)

		}

	},
}
