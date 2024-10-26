package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	version            = "2.0.0"
	colorRed           = "\033[31m"
	colorGreen         = "\033[32m"
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
	path         string
	isDirectory  bool
	lastModified time.Time
	size         int64
}

type trackedFile struct {
	file       fileInformation
	isPresent  bool
	isUpToDate bool
	shasum     string
}

type application struct {
	config             config
	trackedFiles       map[string]trackedFile
	sortedTrackedFiles []string
}

func main() {

	var action string
	var force bool
	var quiet bool
	var filePath string

	verbose := true

	if len(os.Args) == 2 {
		if os.Args[1] == "check" || os.Args[1] == "update" {
			action = os.Args[1]
		} else {
			action = "help"
		}
	} else {
		flag.StringVar(&action, "action", "help", "Action to perform. Possible values: check, update, help.")
		flag.BoolVar(&force, "force", false, "Force the action to be performed, it checks the files completely to confirm if they are up to date.")
		flag.BoolVar(&quiet, "quiet", false, "Prints only the summary of the files.")
		flag.StringVar(&filePath, "file", "", "File to check or update. It can be a file or a folder.")

		flag.Parse()

		if quiet {
			verbose = false
		}
	}

	if action != "check" && action != "update" && action != "help" {
		printError("Invalid action. Possible values: check, update, help.")
	}

	if action == "help" {
		fmt.Println("GitLFSLite is a tool to help you manage your large files in your git repository. It adds a JSON file to your repository with the information of each file so that you can check if the files are up to date. It also  helps you to keep a remote copy of the files using rsync.")
		fmt.Println("GitLFSLite version " + version)
		fmt.Println("Usage: glflite [options]")
		fmt.Println("Options:")
		fmt.Println("  -action string")
		fmt.Println("    	Action to perform. Possible values: check, update, help. (default \"help\")")
		fmt.Println("    	Actions:")
		fmt.Println("  		check")
		fmt.Println("    		Checks if the files are up to date.")
		fmt.Println("  		update")
		fmt.Println("    		Creates the JSON file with the information of the new files and updates the information of the existing files.")
		fmt.Println("  -file string")
		fmt.Println("    	File to check or update. It can be a file or a folder.")
		fmt.Println("  -force")
		fmt.Println("    	Force the action to be performed, it checks the Sha256 sum of the files to confirm if they are up to date. Whitoout this flag, it only checks the last modified date.")
		fmt.Println("  -verbose")
		fmt.Println("    	Prints more information about the files.")
		os.Exit(0)
	}

	var cfg config

	// Check if folder belongs to a git repository
	gitFolder, err := findGitFolder()

	if err != nil {
		printError(err.Error())
	}

	// check if the folder has a .gitignore file, ask the user if they want to create one if it doesn't
	if !hasGitIgnoreFile(gitFolder) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("The folder doesn't have a .gitignore file. Do you want to create a .gitignore file? (yes/no): ")
		response, _ := reader.ReadString('\n')

		response = strings.TrimSpace(strings.ToLower(response))

		if response == "yes" || response == "y" {
			createGitIgnoreFile(gitFolder)
		}
	}

	// Get the content of the .gitignore file
	fileRules, err := getGitIgnoreContent(gitFolder)

	if err != nil {
		printError(err.Error())
	}

	cfg.rootFolder = gitFolder
	cfg.fileRules = fileRules

	app := &application{
		config:       cfg,
		trackedFiles: make(map[string]trackedFile),
	}

	// TODO Add instance information to find out if a files is backed up on another instance easily

	// Find all files and folders in the root folder
	files, err := findAllFilesAndFolders(cfg.rootFolder)

	if err != nil {
		printError(err.Error())
	}

	for _, file := range files {
		// Check if the file is excluded by the .gitignore file after the #GitLFSLite separator
		if isFileExcluded(cfg.fileRules, file.path, file.isDirectory) {
			app.trackedFiles[file.path] = trackedFile{
				file:       file,
				isPresent:  true,
				isUpToDate: false,
			}
		}
	}

	app.sortedTrackedFiles = make([]string, 0, len(app.trackedFiles))

	for file := range app.trackedFiles {
		app.sortedTrackedFiles = append(app.sortedTrackedFiles, file)
	}

	sort.Strings(app.sortedTrackedFiles)

	if action == "check" {
		filesMissing := 0
		filesUpToDate := 0
		filesNotUpToDate := 0

		for _, fileFullPath := range app.sortedTrackedFiles {
			file := app.trackedFiles[fileFullPath]

			if !file.isPresent {
				if verbose {
					fmt.Printf("%s: ", file.file.path)
					printRed("Missing")
				}
				filesMissing++
			} else {
				fileData, err := app.readJSONFile(fileFullPath)

				if errors.Is(err, ErrGLFLiteFileNotFound) {
					if verbose {
						fmt.Printf("File %s is missing the GLFLite file.\n", fileFullPath)
					}

				} else if err != nil {
					printError(err.Error())
				} else if err == nil {
					if fileData.LastModified == file.file.lastModified && fileData.Size == file.file.size {

						if force {
							shaSum, err := app.getFileShasum(fileFullPath)

							if err != nil {
								printError(err.Error())
							}

							if shaSum != fileData.Sha256Sum {
								file.isUpToDate = false
							}

							if verbose {
								fmt.Printf("File %s is up to date because the Sha256 sum is the same: %s\n", fileFullPath, shaSum)
							}
						} else {
							if verbose {
								fmt.Printf("File %s is up to date because the last modified date and the size are the same.\n", fileFullPath)
							}
						}

						file.isUpToDate = true
					}
				} else {
					printError("Unknown error.")
				}

				if file.isUpToDate {
					if verbose {
						fmt.Printf("%s: ", file.file.path)
						printGreen("Up to date")
					}
					filesUpToDate++
				} else {
					if verbose {
						fmt.Printf("%s: ", file.file.path)
						printRed("Not up to date")
					}
					filesNotUpToDate++
				}
			}
		}

		if verbose {
			fmt.Println()
		}

		fmt.Printf("Files missing: ")
		printRed(strconv.Itoa(filesMissing))

		fmt.Printf("Files up to date: ")
		printGreen(strconv.Itoa(filesUpToDate))

		fmt.Printf("Files not up to date: ")
		printRed(strconv.Itoa(filesNotUpToDate))

		if !force {
			fmt.Println("The files are checked using the last modified date and the size.")
			fmt.Println("To check the files using the Sha256 sum, use the -force flag.")
		}

	}

	if action == "update" {
		for _, fileFullPath := range app.sortedTrackedFiles {
			file := app.trackedFiles[fileFullPath]

			if fileFullPath != file.file.path {
				printError("The file path is different from the file name.")
			}

			if file.isPresent {
				data, err := app.readJSONFile(fileFullPath)

				if errors.Is(err, ErrGLFLiteFileNotFound) {

					if verbose {
						fmt.Println("Creating GLFLite file for " + fileFullPath)
					}

					shasum, err := app.getFileShasum(fileFullPath)

					if err != nil {
						printError(err.Error())
					}

					data = fileData{
						FilePath:     fileFullPath,
						TrackedSince: time.Now(),
						LastModified: file.file.lastModified,
						Size:         file.file.size,
						Sha256Sum:    shasum,
					}

					err = app.writeJSONFile(fileFullPath, data)

					if err != nil {
						printError(err.Error())
					}

				} else if err != nil {
					printError(err.Error())
				} else if err == nil {
					if data.LastModified == file.file.lastModified && data.Size == file.file.size {
						if verbose {
							fmt.Println("File " + fileFullPath + " is up to date.")
						}
					} else {
						if verbose {
							fmt.Println("Updating GLFLite file for " + fileFullPath)
						}

						data.LastModified = file.file.lastModified
						data.Size = file.file.size

						shaSum, err := app.getFileShasum(fileFullPath)

						if err != nil {
							printError(err.Error())
						}

						data.Sha256Sum = shaSum

						err = app.writeJSONFile(fileFullPath, data)

						if err != nil {
							printError(err.Error())
						}
					}
				} else {
					printError("Unknown error.")
				}
			}
		}
	}
}

func printError(message string) {
	printRed(message)
	os.Exit(1)
}

func printRed(message string) {
	fmt.Printf("%s%s%s\n", colorRed, message, colorReset)
}

func printGreen(message string) {
	fmt.Printf("%s%s%s\n", colorGreen, message, colorReset)
}
