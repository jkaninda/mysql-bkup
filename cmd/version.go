package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var appVersion = os.Getenv("VERSION")

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		Version()
	},
}

// Version display application version
func Version() {
	fmt.Printf("Version: %s \n", appVersion)
	fmt.Print()
}
