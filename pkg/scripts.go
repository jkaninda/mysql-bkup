package pkg

// Package pkg /*
/*
Copyright © 2024 Jonas Kaninda  <jonaskaninda.gmail.com>
*/
import (
	"fmt"
	"github.com/jkaninda/mysql-bkup/utils"
	"os"
	"os/exec"
)

func init() {

}
func CreateCrontabScript(disableCompression bool, storage string) {
	task := "/usr/local/bin/backup_cron.sh"
	touchCmd := exec.Command("touch", task)
	if err := touchCmd.Run(); err != nil {
		utils.Fatalf("Error creating file %s: %v\n", task, err)
	}
	var disableC = ""
	if disableCompression {
		disableC = "--disable-compression"
	}

	var scriptContent string

	if storage == "s3" {
		scriptContent = fmt.Sprintf(`#!/usr/bin/env bash
set -e
bkup --operation backup --dbname %s --port %s --storage s3 --path %s %v
`, os.Getenv("DB_NAME"), os.Getenv("DB_PORT"), os.Getenv("S3_PATH"), disableC)
	} else {
		scriptContent = fmt.Sprintf(`#!/usr/bin/env bash
set -e
bkup --operation backup --dbname %s --port %s %v
`, os.Getenv("DB_NAME"), os.Getenv("DB_PORT"), disableC)
	}

	if err := utils.WriteToFile(task, scriptContent); err != nil {
		utils.Fatalf("Error writing to %s: %v\n", task, err)
	}

	chmodCmd := exec.Command("chmod", "+x", "/usr/local/bin/backup_cron.sh")
	if err := chmodCmd.Run(); err != nil {
		utils.Fatalf("Error changing permissions of %s: %v\n", task, err)
	}

	lnCmd := exec.Command("ln", "-s", "/usr/local/bin/backup_cron.sh", "/usr/local/bin/backup_cron")
	if err := lnCmd.Run(); err != nil {
		utils.Fatalf("Error creating symbolic link: %v\n", err)

	}

	cronJob := "/etc/cron.d/backup_cron"
	touchCronCmd := exec.Command("touch", cronJob)
	if err := touchCronCmd.Run(); err != nil {
		utils.Fatalf("Error creating file %s: %v\n", cronJob, err)
	}

	cronContent := fmt.Sprintf(`%s root exec /bin/bash -c ". /run/supervisord.env; /usr/local/bin/backup_cron.sh >> /var/log/mysql-bkup.log"
`, os.Getenv("SCHEDULE_PERIOD"))

	if err := utils.WriteToFile(cronJob, cronContent); err != nil {
		utils.Fatalf("Error writing to %s: %v\n", cronJob, err)
	}
	utils.ChangePermission("/etc/cron.d/backup_cron", 0644)

	crontabCmd := exec.Command("crontab", "/etc/cron.d/backup_cron")
	if err := crontabCmd.Run(); err != nil {
		utils.Fatal("Error updating crontab: ", err)
	}
	utils.Info("Starting backup in scheduled mode")
}
