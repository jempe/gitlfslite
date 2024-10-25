package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	version            = "2.0.0"
	colorRed           = "\033[31m"
	colorReset         = "\033[0m"
	setupFile          = ".glflite"
	fileExtension      = "glflite"
	gitIgnoreSeparator = "#GitLFSLite"
)

type config struct {
	rootFolder string
	fileRules  []string
	instance   struct {
		hostname string
		path     string
		ID       string
	}
}

type fileInformation struct {
	path            string
	isDirectory     bool
	lastModified    int64
	sha256sum       string
	glfliteFilePath string
}

var fileList map[string]fileInformation

func main() {
	var cfg config

	gitFolder, err := findGitFolder()

	if err != nil {
		printError(err.Error())
	}

	if !hasGitIgnoreFile(gitFolder) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("The folder doesn't have a .gitignore file. Do you want to create a .gitignore file? (yes/no): ")
		response, _ := reader.ReadString('\n')

		response = strings.TrimSpace(strings.ToLower(response))

		if response == "yes" || response == "y" {
			createGitIgnoreFile(gitFolder)
		}
	}

	fileRules, err := getGitIgnoreContent(gitFolder)

	if err != nil {
		printError(err.Error())
	}

	cfg.rootFolder = gitFolder
	cfg.fileRules = fileRules

	// TODO Add instance information to find out if a files is backed up on another instance easily

	files, err := findAllFilesAndFolders(cfg.rootFolder)

	if err != nil {
		printError(err.Error())
	}

	for _, file := range files {
		if isFileExcluded(cfg.fileRules, strings.TrimPrefix(file.Path, "./"), file.IsDirectory) {
			file.GlfliteFilePath = getGLFLiteFilePath(file.Path)
			fmt.Printf("Exclude File: %s \nGLFLite File: %s\n", file.Path, file.GlfliteFilePath)
		}

		if isFileExcluded(cfg.fileRules, strings.TrimPrefix(file.Path, "./"), file.IsDirectory) {
			file.

	}
}

func printError(message string) {
	fmt.Printf("%s%s%s\n", colorRed, message, colorReset)
	os.Exit(1)
}
