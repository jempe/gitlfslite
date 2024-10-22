package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func findGitFolder() (string, error) {
	var err error

	currentFolder := getCurrentFolder()

	folder := currentFolder

	for {
		if hasGitFolder(folder) {
			return folder, nil
		}

		folder, err = getAbsolutePath(folder + "/..")

		if err != nil {
			return "", err
		}

		if folder == "/" {
			break
		}
	}

	return "", errors.New(fmt.Sprintf("Git folder not found in %s or any of its parent folders", currentFolder))
}

func getAbsolutePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

func hasGitFolder(folder string) bool {
	gitFolder := folder + "/.git"
	if fileExists(gitFolder) && isDirectory(gitFolder) {
		return true
	}

	return false
}

func hasGitIgnoreFile(folder string) bool {
	if fileExists(folder + "/.gitignore") {
		return true
	}

	return false
}

func createGitIgnoreFile(folder string) error {
	file, err := os.Create(folder + "/.gitignore")

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.WriteString("#GitLFSLite\n")

	if err != nil {
		return err
	}

	return nil
}

func getGitIgnoreContent(folder string) (fileRules []fileRule, err error) {
	gitIgnoreFile := folder + "/.gitignore"

	if !fileExists(gitIgnoreFile) {
		return fileRules, errors.New(fmt.Sprintf("The file %s doesn't exist", gitIgnoreFile))
	}

	if isDirectory(gitIgnoreFile) {
		return fileRules, errors.New(fmt.Sprintf("The file %s is a directory", gitIgnoreFile))
	}

	content, err := ioutil.ReadFile(gitIgnoreFile)

	foundGitLFSLite := false

	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, gitIgnoreSeparator) {
			foundGitLFSLite = true
			continue
		}

		if foundGitLFSLite {
			line = strings.TrimSpace(line)

			if line != "" {

				if strings.HasPrefix(line, "*") {
					fileRules = append(fileRules, fileRule{extension: strings.TrimPrefix(line, "*"), filename: "*", path: ""})
				} else {
					fileRules = append(fileRules, fileRule{extension: filepath.Ext(line), filename: filepath.Base(line), path: filepath.Dir(line)})
				}

			}
		}
	}

	if err != nil {
		return fileRules, err
	}

	return fileRules, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)

	if err != nil {
		return false
	}

	return true
}

func isDirectory(filename string) bool {
	file, err := os.Stat(filename)

	if err != nil {
		return false
	}

	return file.IsDir()
}

func getCurrentFolder() string {
	dir, err := os.Getwd()

	if err != nil {
		return ""
	}

	return dir
}
