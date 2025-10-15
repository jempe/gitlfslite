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

	"golang.org/x/term"
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
	config              config
	trackedFiles        map[string]trackedFile
	sortedTrackedFiles  []string
	duplicatedFiles     map[string][]string
	duplicatedTotalSize int64
}

func main() {

	var action string
	var force bool
	var quiet bool
	var filePath string

	verbose := true

	if len(os.Args) == 2 {
		if os.Args[1] == "check" || os.Args[1] == "update" || os.Args[1] == "encrypt" || os.Args[1] == "decrypt" {
			action = os.Args[1]
		} else {
			action = "help"
		}
	} else {
		flag.StringVar(&action, "action", "help", "Action to perform. Possible values: check, update, encrypt, decrypt, help.")
		flag.BoolVar(&force, "force", false, "Force the action to be performed, it checks the files completely to confirm if they are up to date.")
		flag.BoolVar(&quiet, "quiet", false, "Prints only the summary of the files.")
		flag.StringVar(&filePath, "file", "", "File to check or update. It can be a file or a folder.")

		flag.Parse()

		if quiet {
			verbose = false
		}
	}

	if action != "check" && action != "update" && action != "encrypt" && action != "decrypt" && action != "help" {
		printError("Invalid action. Possible values: check, update, encrypt, decrypt, help.")
	}

	if action == "help" {
		fmt.Println("GitLFSLite is a tool to help you manage your large files in your git repository. It adds a JSON file to your repository with the information of each file so that you can check if the files are up to date. It also  helps you to keep a remote copy of the files using rsync.")
		fmt.Println("GitLFSLite version " + version)
		fmt.Println("Usage: glflite [options]")
		fmt.Println("Options:")
		fmt.Println("  -action string")
		fmt.Println("    	Action to perform. Possible values: check, update, encrypt, decrypt, help. (default \"help\")")
		fmt.Println("    	Actions:")
		fmt.Println("  		check")
		fmt.Println("    		Checks if the files are up to date.")
		fmt.Println("  		update")
		fmt.Println("    		Creates the JSON file with the information of the new files and updates the information of the existing files.")
		fmt.Println("  		encrypt")
		fmt.Println("    		Encrypts all tracked files that don't have a .gpg encrypted file yet.")
		fmt.Println("  		decrypt")
		fmt.Println("    		Decrypts all .gpg files that don't have the original file yet.")
		fmt.Println("  -file string")
		fmt.Println("    	File to check or update. It can be a file or a folder.")
		fmt.Println("  -force")
		fmt.Println("    	Force the action to be performed, it checks the Sha256 sum of the files to confirm if they are up to date. Whitoout this flag, it only checks the last modified date.")
		fmt.Println("  -quiet")
		fmt.Println("    	Prints only the summary of the files.")
		fmt.Println("To sync the files, use the rsync command with the list of files in the rsync_list_glflite file.")
		fmt.Println("Example:")
		fmt.Println("   rsync -v -t --ignore-missing-args --files-from=rsync_list_glflite . [destination]")
		os.Exit(0)
	}

	// Read password from stdin if encrypt or decrypt action is used
	var password string
	if action == "encrypt" || action == "decrypt" {
		fmt.Print("Enter password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // Print newline after password input
		if err != nil {
			printError("Failed to read password: " + err.Error())
		}
		password = strings.TrimSpace(string(passwordBytes))
		if password == "" {
			printError("Password cannot be empty")
		}
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
		config:          cfg,
		trackedFiles:    make(map[string]trackedFile),
		duplicatedFiles: make(map[string][]string),
	}

	// TODO Add instance information to find out if a files is backed up on another instance easily

	// Find all files and folders in the root folder
	files, err := findAllFilesAndFolders(cfg.rootFolder)

	if err != nil {
		printError(err.Error())
	}

	// find all the present tracked files
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

	// find all the glflite files
	for _, file := range files {
		if isGLFLiteFile(file.path) {
			trackedFileName := getTrackedFilePath(file.path)

			if _, ok := app.trackedFiles[trackedFileName]; !ok {
				trackedFileData, err := app.readJSONFile(trackedFileName)

				if err != nil {
					printError(err.Error())
				}

				app.trackedFiles[trackedFileName] = trackedFile{
					file: fileInformation{
						path:         trackedFileName,
						isDirectory:  false,
						lastModified: trackedFileData.LastModified,
						size:         trackedFileData.Size,
					},
					isPresent:  false,
					isUpToDate: false,
				}
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
		ignoredLinks := 0

		for _, fileFullPath := range app.sortedTrackedFiles {
			file := app.trackedFiles[fileFullPath]

			fileData, err := app.readJSONFile(fileFullPath)

			if errors.Is(err, ErrGLFLiteFileNotFound) {
				if !isLink(fileFullPath) && verbose {
					fmt.Printf("File %s is missing the GLFLite file.\n", fileFullPath)
				}

			} else if err != nil {
				printError(err.Error())
			} else if err == nil {

				// Update the Sha256 sum of the file
				trackedFileData := app.trackedFiles[fileFullPath]
				trackedFileData.shasum = fileData.Sha256Sum
				app.trackedFiles[fileFullPath] = trackedFileData
			}

			if !file.isPresent {
				if verbose {
					fmt.Printf("%s: ", file.file.path)
					printRed("Missing")
				}
				filesMissing++
			} else {

				if isLink(fileFullPath) {
					if verbose {
						fmt.Printf("Ignoring link file %s\n", fileFullPath)
					}

					ignoredLinks++
				} else {
					if force {
						shaSum, err := app.getFileShasum(fileFullPath)

						if err != nil {
							printError(err.Error())
						}

						if shaSum == fileData.Sha256Sum {
							file.isUpToDate = true
						} else {
							file.isUpToDate = false
						}

						if verbose {
							fmt.Printf("File %s is up to date because the Sha256 sum is the same: %s\n", fileFullPath, shaSum)
						}
					} else {
						if fileData.LastModified.Unix() == file.file.lastModified.Unix() && fileData.Size == file.file.size {

							if verbose {
								fmt.Printf("File %s is up to date because the last modified date and the size are the same.\n", fileFullPath)
							}

							file.isUpToDate = true
						} else {
							if verbose {
								if fileData.LastModified.Unix() != file.file.lastModified.Unix() {
									fmt.Printf("File %s is not up to date because the last modified date is different. %s != %s\n", fileFullPath, fileData.LastModified, file.file.lastModified)
								}

								if fileData.Size != file.file.size {
									fmt.Printf("File %s is not up to date because the size is different. %d != %d\n", fileFullPath, fileData.Size, file.file.size)
								}
							}

							file.isUpToDate = false
						}
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
		}

		err = app.generateRsyncFileList(true)

		if err != nil {
			printError(err.Error())
		}

		err = app.generateRsyncFileList(false)

		if err != nil {
			printError(err.Error())
		}

		err = app.generateSha256FileList()

		if err != nil {
			printError(err.Error())
		}

		if verbose {
			fmt.Println()
		}

		if app.duplicatedFiles != nil && verbose {
			for shaSum, files := range app.duplicatedFiles {
				printRed("  " + shaSum + ":")
				for i, file := range files {
					fmt.Printf("     %s\n", file)

					if i > 0 {
						fmt.Printf("Command to remove duplicates: mv -i \"%s\" ~/duplicatedFiles && ln -s \"%s\" \"%s.glflitelink\"\n", file, files[0], file)
					}

				}
			}
			fmt.Println()
		}

		fmt.Printf("Files missing: ")
		printRed(strconv.Itoa(filesMissing))

		fmt.Printf("Files up to date: ")
		printGreen(strconv.Itoa(filesUpToDate))

		fmt.Printf("Files not up to date: ")
		printRed(strconv.Itoa(filesNotUpToDate))

		fmt.Printf("Ignored links: ")
		printGreen(strconv.Itoa(ignoredLinks))

		if app.duplicatedTotalSize > 0 {
			printRed("Files with duplicates:" + strconv.Itoa(len(app.duplicatedFiles)))

			var humanSize int64

			if app.duplicatedTotalSize > 1024*1024*1024 {
				humanSize = app.duplicatedTotalSize / (1024 * 1024 * 1024)
				printRed(fmt.Sprintf("Total size of duplicated files: %d GB\n", humanSize))
			} else if app.duplicatedTotalSize > 1024*1024 {
				humanSize = app.duplicatedTotalSize / (1024 * 1024)
				printRed(fmt.Sprintf("Total size of duplicated files: %d MB\n", humanSize))
			} else if app.duplicatedTotalSize > 1024 {
				humanSize = app.duplicatedTotalSize / 1024
				printRed(fmt.Sprintf("Total size of duplicated files: %d KB\n", humanSize))
			} else {
				printRed(fmt.Sprintf("Total size of duplicated files: %d B\n", app.duplicatedTotalSize))
			}
		}

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

					if isLink(fileFullPath) {
						if verbose {
							fmt.Println("Ignoring link file " + fileFullPath)
						}
					} else {
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

						newTrackedFile := trackedFile{
							file:       file.file,
							isPresent:  true,
							isUpToDate: true,
							shasum:     shasum,
						}

						app.trackedFiles[fileFullPath] = newTrackedFile

						err = app.writeJSONFile(fileFullPath, data)

						if err != nil {
							printError(err.Error())
						}
					}

				} else if err != nil {
					printError(err.Error())
				} else if err == nil {
					// Update the Sha256 sum of the file
					trackedFileData := app.trackedFiles[fileFullPath]
					trackedFileData.shasum = data.Sha256Sum
					app.trackedFiles[fileFullPath] = trackedFileData

					if data.LastModified.Unix() == file.file.lastModified.Unix() && data.Size == file.file.size {
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

		err = app.generateRsyncFileList(true)

		if err != nil {
			printError(err.Error())
		}

		err = app.generateRsyncFileList(false)

		if err != nil {
			printError(err.Error())
		}

		err = app.generateSha256FileList()

		if err != nil {
			printError(err.Error())
		}

	}

	if action == "encrypt" {
		if !isGPGInstalled() {
			printError("GPG is not installed. Please install GPG to use encryption.")
		}

		// Check if verification file exists, if so verify password, otherwise create it
		verifyFileGPG := cfg.rootFolder + "/glflite-verify.gpg"
		if fileExists(verifyFileGPG) {
			if verbose {
				fmt.Println("Verifying password...")
			}
			err := verifyPassword(cfg.rootFolder, password)
			if err != nil {
				printError("Password verification failed: " + err.Error())
			}
			if verbose {
				fmt.Println("Password verified successfully.")
			}
		} else {
			if verbose {
				fmt.Println("Creating password verification file...")
			}
			err := createPasswordVerificationFile(cfg.rootFolder, password)
			if err != nil {
				printError("Failed to create password verification file: " + err.Error())
			}
			if verbose {
				fmt.Println("Password verification file created successfully.")
			}
		}

		filesEncrypted := 0
		filesSkipped := 0
		filesError := 0

		for _, fileFullPath := range app.sortedTrackedFiles {
			file := app.trackedFiles[fileFullPath]

			if !file.isPresent {
				continue
			}

			if isLink(fileFullPath) {
				if verbose {
					fmt.Printf("Skipping link file %s\n", fileFullPath)
				}
				filesSkipped++
				continue
			}

			encryptedFile := fileFullPath + ".gpg"
			if fileExists(encryptedFile) {
				if verbose {
					fmt.Printf("Encrypted file already exists for %s\n", fileFullPath)
				}
				filesSkipped++
				continue
			}

			if verbose {
				fmt.Printf("Encrypting %s...\n", fileFullPath)
			}

			err := encryptFile(fileFullPath, password)
			if err != nil {
				if verbose {
					fmt.Printf("%sError encrypting %s: %s%s\n", colorRed, fileFullPath, err.Error(), colorReset)
				}
				filesError++
			} else {
				if verbose {
					fmt.Printf("%sSuccessfully encrypted %s%s\n", colorGreen, fileFullPath, colorReset)
				}
				filesEncrypted++
			}
		}

		fmt.Printf("\nEncryption Summary:\n")
		fmt.Printf("Files encrypted: ")
		printGreen(strconv.Itoa(filesEncrypted))
		fmt.Printf("Files skipped: ")
		printGreen(strconv.Itoa(filesSkipped))
		if filesError > 0 {
			fmt.Printf("Files with errors: ")
			printRed(strconv.Itoa(filesError))
		}
	}

	if action == "decrypt" {
		if !isGPGInstalled() {
			printError("GPG is not installed. Please install GPG to use decryption.")
		}

		// Verify password before attempting to decrypt files
		verifyFileGPG := cfg.rootFolder + "/glflite-verify.gpg"
		if fileExists(verifyFileGPG) {
			if verbose {
				fmt.Println("Verifying password...")
			}
			err := verifyPassword(cfg.rootFolder, password)
			if err != nil {
				printError("Password verification failed: " + err.Error())
			}
			if verbose {
				fmt.Println("Password verified successfully.")
			}
		} else {
			if verbose {
				fmt.Println("Warning: Password verification file not found. Proceeding without verification.")
			}
		}

		filesDecrypted := 0
		filesSkipped := 0
		filesError := 0

		// Find all .gpg files in the repository
		files, err := findAllFilesAndFolders(cfg.rootFolder)
		if err != nil {
			printError(err.Error())
		}

		for _, file := range files {
			if file.isDirectory {
				continue
			}

			// Check if it's a .gpg file
			if len(file.path) < 5 || file.path[len(file.path)-4:] != ".gpg" {
				continue
			}

			// Check if the original file already exists
			originalFile := file.path[:len(file.path)-4]
			if fileExists(originalFile) {
				if verbose {
					fmt.Printf("Original file already exists for %s\n", file.path)
				}
				filesSkipped++
				continue
			}

			if verbose {
				fmt.Printf("Decrypting %s...\n", file.path)
			}

			err := decryptFile(file.path, password)
			if err != nil {
				if verbose {
					fmt.Printf("%sError decrypting %s: %s%s\n", colorRed, file.path, err.Error(), colorReset)
					if filesDecrypted == 0 && filesError == 0 {
						fmt.Printf("%sHint: Check if your password is correct%s\n", colorRed, colorReset)
					}
				}
				filesError++
			} else {
				if verbose {
					fmt.Printf("%sSuccessfully decrypted %s%s\n", colorGreen, file.path, colorReset)
				}
				filesDecrypted++

				// Restore the original file timestamp from the .glflite file
				if trackedFile, ok := app.trackedFiles[originalFile]; ok {
					err := os.Chtimes(originalFile, trackedFile.file.lastModified, trackedFile.file.lastModified)
					if err != nil {
						if verbose {
							fmt.Printf("%sWarning: Could not restore timestamp for %s: %s%s\n", colorRed, originalFile, err.Error(), colorReset)
						}
					} else if verbose {
						fmt.Printf("Restored timestamp for %s to %s\n", originalFile, trackedFile.file.lastModified.Format("2006-01-02 15:04:05"))
					}
				}
			}
		}

		fmt.Printf("\nDecryption Summary:\n")
		fmt.Printf("Files decrypted: ")
		printGreen(strconv.Itoa(filesDecrypted))
		fmt.Printf("Files skipped: ")
		printGreen(strconv.Itoa(filesSkipped))
		if filesError > 0 {
			fmt.Printf("Files with errors: ")
			printRed(strconv.Itoa(filesError))
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
