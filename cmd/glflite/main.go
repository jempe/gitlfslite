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
	fileRules  []fileRule
	instance   struct {
		hostname string
		path     string
		ID       string
	}
}

type fileRule struct {
	extension string
	filename  string
	path      string
}

func main() {
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

	fmt.Println(fileRules)
}

func printError(message string) {
	fmt.Printf("%s%s%s\n", colorRed, message, colorReset)
	os.Exit(1)
}
