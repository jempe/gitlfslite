package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileToSort struct {
	Shasum string
	Path   string
}

func sortFiles(files []fileToSort) []fileToSort {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Shasum < files[j].Shasum
	})

	return files
}

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

	_, err = file.WriteString("\nrsync_list_glflite_local\n#GitLFSLite\n")

	if err != nil {
		return err
	}

	return nil
}

func getGitIgnoreContent(folder string) (fileRules []string, err error) {
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
			if line != "" {
				fileRules = append(fileRules, line)
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

func findAllFilesAndFolders(folder string) (files []fileInformation, err error) {

	err = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath := strings.TrimPrefix(path, folder)

		//exclude setup file

		if relativePath == "/"+setupFile {
			return nil
		}

		// exclude the root folder
		if relativePath == "" {
			return nil
		}

		// exclude the .git folder
		if relativePath == "/.git" {
			return nil
		}

		// exclude .git folder and its content
		if strings.HasPrefix(relativePath, "/.git/") {
			return nil
		}

		// exclude .gitignore file
		if strings.HasPrefix(relativePath, "/.gitignore") {
			return nil
		}

		filePath := strings.TrimPrefix(relativePath, "/")

		files = append(files, fileInformation{
			path:         filePath,
			isDirectory:  info.IsDir(),
			lastModified: info.ModTime(),
			size:         info.Size(),
		})

		return nil
	})

	if err != nil {
		return files, err
	}

	return files, nil
}

func isGLFLiteFile(file string) bool {

	return strings.HasSuffix(file, "."+fileExtension)
}

func getGLFLiteFilePath(file string) string {

	return file + "." + fileExtension
}

func getTrackedFilePath(file string) string {

	return strings.TrimSuffix(file, "."+fileExtension)
}

func isFileExcluded(gitIgnoreFiles []string, path string, isDir bool) bool {

	// This will store whether the file is currently excluded
	isExcluded := false

	// Iterate through the lines of the .gitignore
	for _, line := range gitIgnoreFiles {
		// Trim whitespace and ignore empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle negation ("!" prefix means inclusion)
		negate := false
		if strings.HasPrefix(line, "!") {
			negate = true
			line = line[1:] // Remove the "!" prefix
		}

		// Normalize pattern for cross-platform compatibility
		line = filepath.ToSlash(line)

		// Check if the line is a directory pattern (ends with "/")
		isPatternDir := strings.HasSuffix(line, "/")
		if isPatternDir {
			// Remove the trailing "/" for directory patterns
			line = strings.TrimSuffix(line, "/")
		}

		// Match pattern using filepath.Match (but handling special cases)
		matched := false
		if strings.HasPrefix(line, "*") {
			excludedFileSuffix := strings.ToLower(strings.TrimPrefix(line, "*"))

			if strings.HasSuffix(strings.ToLower(path), excludedFileSuffix) {
				matched = true
			}
		} else {
			// Use normal matching for other patterns
			if path == line {
				matched = true
			}
		}

		// If it's a directory pattern, ensure we're matching a directory
		if isPatternDir && !isDir {
			matched = false
		}

		// Apply the rule based on whether the pattern matched and whether it's a negation
		if matched {
			if negate {
				isExcluded = false // Negate means re-inclusion
			} else {
				isExcluded = true // Otherwise, exclude the file
			}
		}
	}

	return isExcluded
}

func (app *application) getFileShasum(fileName string) (string, error) {
	bufferSize := 32 * 1024 // 32KB buffer

	filePath := fileName
	if !fileExists(filePath) {
		return "", errors.New(fmt.Sprintf("file %s does not exist", fileName))
	}

	file, err := os.Open(app.getFullPath(filePath))
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	buf := make([]byte, bufferSize)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, err := hash.Write(buf[:n])
			if err != nil {
				return "", err
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (app *application) generateRsyncFileList(local bool) error {
	fileName := "rsync_list_" + fileExtension

	if local {
		fileName += "_local"
	}

	file, err := os.Create(app.getFullPath(fileName))

	if err != nil {
		return err
	}

	defer file.Close()

	for _, fileFullPath := range app.sortedTrackedFiles {
		trackedFile := app.trackedFiles[fileFullPath]

		if !local || (trackedFile.isPresent && local) {

			_, err = file.WriteString(fmt.Sprintf("./%s\n", fileFullPath))

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (app *application) generateSha256FileList() error {
	var lines []string

	sortedByShasum := []fileToSort{}

	for _, fileFullPath := range app.sortedTrackedFiles {
		trackedFile := app.trackedFiles[fileFullPath]

		if trackedFile.shasum != "" {
			lines = append(lines, fmt.Sprintf("%s  ./%s", trackedFile.shasum, fileFullPath))

			sortedByShasum = append(sortedByShasum, fileToSort{Shasum: trackedFile.shasum, Path: fileFullPath})
		}
	}

	sort.Strings(lines)

	lastShasum := ""
	lastFilePath := ""

	for _, sortedFile := range sortFiles(sortedByShasum) {
		if lastShasum == sortedFile.Shasum {
			if _, ok := app.duplicatedFiles[sortedFile.Shasum]; !ok {
				app.duplicatedFiles[sortedFile.Shasum] = []string{lastFilePath}

				fmt.Printf("Original file: %s\n", lastFilePath)
			}

			fmt.Printf("Duplicated file: %s\n", sortedFile.Path)

			app.duplicatedFiles[sortedFile.Shasum] = append(app.duplicatedFiles[sortedFile.Shasum], sortedFile.Path)
		}

		lastShasum = sortedFile.Shasum
		lastFilePath = sortedFile.Path
	}

	err := ioutil.WriteFile(app.getFullPath("sha256_list_"+fileExtension), []byte(strings.Join(lines, "\n")), 0644)

	if err != nil {
		return err
	}

	return nil
}
