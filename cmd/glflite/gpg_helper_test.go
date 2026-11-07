package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// requireGPG skips the test when gpg is unavailable and isolates gpg in a
// temporary home so the user's keyring and agent cache are never touched.
func requireGPG(t *testing.T) {
	t.Helper()

	if !isGPGInstalled() {
		t.Skip("gpg is not installed")
	}

	home := t.TempDir()
	t.Setenv("GNUPGHOME", home)
	t.Cleanup(func() {
		exec.Command("gpgconf", "--kill", "gpg-agent").Run()
	})
}

func TestEncryptDecryptFile(t *testing.T) {
	requireGPG(t)

	root := t.TempDir()
	writeTestFile(t, root, "secret.bin", "hello")
	file := filepath.Join(root, "secret.bin")

	if err := encryptFile(file, "pass"); err != nil {
		t.Fatal(err)
	}

	if !fileExists(file + ".gpg") {
		t.Fatal("expected the .gpg file to exist")
	}

	if err := encryptFile(file, "pass"); err == nil {
		t.Error("expected an error when the .gpg file already exists")
	}

	if err := decryptFile(file+".gpg", "pass"); err == nil {
		t.Error("expected an error when the decrypted file already exists")
	}

	writeTestFile(t, root, "copy.bin.gpg", readTestFile(t, file+".gpg"))

	if err := decryptFile(filepath.Join(root, "copy.bin.gpg"), "pass"); err != nil {
		t.Fatal(err)
	}

	if got := readTestFile(t, filepath.Join(root, "copy.bin")); got != "hello" {
		t.Errorf("decrypted content = %q, want %q", got, "hello")
	}
}

func TestEncryptDecryptFileErrors(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "plain.bin", "hello")

	if err := encryptFile(filepath.Join(root, "missing.bin"), "pass"); err == nil {
		t.Error("expected an error for a missing file")
	}

	if err := encryptFile(filepath.Join(root, "plain.bin"), ""); err == nil {
		t.Error("expected an error for an empty password")
	}

	if err := decryptFile(filepath.Join(root, "plain.bin"), "pass"); err == nil {
		t.Error("expected an error for a non .gpg file")
	}
}

func TestIsDecryptableFile(t *testing.T) {
	tests := map[string]bool{
		"video.mp4.gpg":          true,
		"media/video.mp4.gpg":    true,
		"sub/glflite-verify.gpg": true,
		"glflite-verify.gpg":     false,
		"video.mp4":              false,
		"video.mp4.gpg.glflite":  false,
		".gpg":                   false,
	}

	for path, want := range tests {
		if got := isDecryptableFile(path); got != want {
			t.Errorf("isDecryptableFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestPasswordVerification(t *testing.T) {
	requireGPG(t)

	root := t.TempDir()

	if err := verifyPassword(root, "pass"); err == nil {
		t.Error("expected an error when the verification file is missing")
	}

	if err := createPasswordVerificationFile(root, "pass"); err != nil {
		t.Fatal(err)
	}

	if fileExists(filepath.Join(root, "glflite-verify")) {
		t.Error("the unencrypted verification file should be removed")
	}

	if err := verifyPassword(root, "pass"); err != nil {
		t.Errorf("correct password rejected: %v", err)
	}

	if err := verifyPassword(root, "wrong"); err == nil {
		t.Error("wrong password accepted")
	}

	if fileExists(filepath.Join(root, "glflite-verify")) {
		t.Error("the decrypted verification file should be cleaned up")
	}
}
