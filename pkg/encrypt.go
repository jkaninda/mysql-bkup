package pkg

import (
	"github.com/jkaninda/mysql-bkup/utils"
	"os"
	"os/exec"
	"strings"
)

func Decrypt(inputFile string, passphrase string) error {
	utils.Info("Decrypting backup file: " + inputFile + " ...")
	//Create gpg home dir
	err := utils.MakeDir(gpgHome)
	if err != nil {
		return err
	}
	utils.SetEnv("GNUPGHOME", gpgHome)
	cmd := exec.Command("gpg", "--batch", "--passphrase", passphrase, "--output", RemoveLastExtension(inputFile), "--decrypt", inputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	utils.Info("Backup file decrypted successful!")
	return nil
}

func Encrypt(inputFile string, passphrase string) error {
	utils.Info("Encrypting backup...")
	//Create gpg home dir
	err := utils.MakeDir(gpgHome)
	if err != nil {
		return err
	}
	utils.SetEnv("GNUPGHOME", gpgHome)
	cmd := exec.Command("gpg", "--batch", "--passphrase", passphrase, "--symmetric", "--cipher-algo", algorithm, inputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	utils.Info("Backup file encrypted successful!")
	return nil
}

func RemoveLastExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[:idx]
	}
	return filename
}
