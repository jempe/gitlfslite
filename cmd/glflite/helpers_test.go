package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestRepo creates a fake repository (a folder with a .git directory and a
// .gitignore) in a temporary directory outside the project.
func newTestRepo(t *testing.T, gitIgnore string) string {
	t.Helper()

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, root, ".gitignore", gitIgnore)

	return root
}

func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(root, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

// chdir changes the working directory for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.Chdir(oldDir)
	})
}

func newTestApp(root string) *application {
	return &application{
		config:          config{rootFolder: root},
		trackedFiles:    make(map[string]trackedFile),
		duplicatedFiles: make(map[string][]string),
	}
}

func addTrackedFile(app *application, path string, isPresent bool, shasum string, size int64) {
	app.trackedFiles[path] = trackedFile{
		file:      fileInformation{path: path, lastModified: time.Unix(1700000000, 0), size: size},
		isPresent: isPresent,
		shasum:    shasum,
	}
	app.sortedTrackedFiles = append(app.sortedTrackedFiles, path)
}
