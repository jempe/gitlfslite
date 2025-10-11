package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func isGPGInstalled() bool {
	_, err := exec.LookPath("gpg")
	return err == nil
}

func encryptFile(filename, password string) error {
	if !fileExists(filename) {
		return fmt.Errorf("file %s does not exist", filename)
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if !isGPGInstalled() {
		return fmt.Errorf("gpg is not installed")
	}

	outputFile := filename + ".gpg"

	if fileExists(outputFile) {
		return fmt.Errorf("encrypted file %s already exists", outputFile)
	}

	cmd := exec.Command("gpg",
		"--batch",
		"--yes",
		"--passphrase-fd", "0",
		"--symmetric",
		"--cipher-algo", "AES256",
		"--output", outputFile,
		filename)

	cmd.Stdin = bytes.NewBufferString(password)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("gpg encryption failed: %s - %s", err.Error(), stderr.String())
	}

	return nil
}

func decryptFile(filename, password string) error {
	if !fileExists(filename) {
		return fmt.Errorf("file %s does not exist", filename)
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if !isGPGInstalled() {
		return fmt.Errorf("gpg is not installed")
	}

	// Remove .gpg extension to get output filename
	if len(filename) < 5 || filename[len(filename)-4:] != ".gpg" {
		return fmt.Errorf("file %s is not a .gpg file", filename)
	}

	outputFile := filename[:len(filename)-4]

	if fileExists(outputFile) {
		return fmt.Errorf("decrypted file %s already exists", outputFile)
	}

	cmd := exec.Command("gpg",
		"--batch",
		"--yes",
		"--passphrase-fd", "0",
		"--decrypt",
		"--output", outputFile,
		filename)

	cmd.Stdin = bytes.NewBufferString(password)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("gpg decryption failed: %s - %s", err.Error(), stderr.String())
	}

	return nil
}
