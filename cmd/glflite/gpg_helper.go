package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// createPasswordVerificationFile creates a verification file to validate passwords
func createPasswordVerificationFile(rootFolder, password string) error {
	verifyFile := filepath.Join(rootFolder, "glflite-verify")
	verifyFileGPG := verifyFile + ".gpg"

	// Check if verification file already exists
	if fileExists(verifyFileGPG) {
		return nil // Already exists, no need to create
	}

	// Create a temporary file with a known token
	token := "glflite-password-verification-token"
	err := os.WriteFile(verifyFile, []byte(token), 0644)
	if err != nil {
		return fmt.Errorf("failed to create verification file: %s", err.Error())
	}

	// Encrypt the verification file
	err = encryptFile(verifyFile, password)
	if err != nil {
		os.Remove(verifyFile) // Clean up
		return fmt.Errorf("failed to encrypt verification file: %s", err.Error())
	}

	// Remove the unencrypted verification file
	err = os.Remove(verifyFile)
	if err != nil {
		return fmt.Errorf("failed to remove unencrypted verification file: %s", err.Error())
	}

	return nil
}

// verifyPassword checks if the provided password is correct by attempting to decrypt the verification file
func verifyPassword(rootFolder, password string) error {
	verifyFileGPG := filepath.Join(rootFolder, "glflite-verify.gpg")

	// Check if verification file exists
	if !fileExists(verifyFileGPG) {
		return fmt.Errorf("password verification file not found")
	}

	// Create a temporary output file path
	verifyFile := filepath.Join(rootFolder, "glflite-verify")

	// If the decrypted file exists, remove it first (decryptFile will fail if it exists)
	if fileExists(verifyFile) {
		err := os.Remove(verifyFile)
		if err != nil {
			return fmt.Errorf("failed to remove existing verification file: %s", err.Error())
		}
	}

	// Try to decrypt the verification file using the existing decryptFile function
	err := decryptFile(verifyFileGPG, password)
	if err != nil {
		return fmt.Errorf("incorrect password")
	}

	// Clean up the decrypted verification file
	os.Remove(verifyFile)

	return nil
}
