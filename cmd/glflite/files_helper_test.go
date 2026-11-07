package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// sha256 of "hello"
const helloShasum = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

func TestIsFileExcluded(t *testing.T) {
	tests := []struct {
		name  string
		rules []string
		path  string
		isDir bool
		want  bool
	}{
		{"no rules", nil, "video.mp4", false, false},
		{"wildcard suffix", []string{"*.mp4"}, "video.mp4", false, true},
		{"wildcard suffix in subfolder", []string{"*.mp4"}, "media/video.mp4", false, true},
		{"wildcard is case insensitive", []string{"*.mp4"}, "VIDEO.MP4", false, true},
		{"wildcard pattern uppercase", []string{"*.MP4"}, "video.mp4", false, true},
		{"wildcard no match", []string{"*.mp4"}, "video.mov", false, false},
		{"exact path", []string{"media/big.bin"}, "media/big.bin", false, true},
		{"exact path no partial match", []string{"big.bin"}, "media/big.bin", false, false},
		{"directory pattern matches dir", []string{"media/"}, "media", true, true},
		{"directory pattern ignores file", []string{"media/"}, "media", false, false},
		{"negation re-includes", []string{"*.mp4", "!keep.mp4"}, "keep.mp4", false, false},
		{"negation only affects match", []string{"*.mp4", "!keep.mp4"}, "other.mp4", false, true},
		{"last rule wins", []string{"!keep.mp4", "*.mp4"}, "keep.mp4", false, true},
		{"comments and blanks ignored", []string{"# *.mp4", "   ", ""}, "video.mp4", false, false},
		{"rule whitespace trimmed", []string{"  *.mp4  "}, "video.mp4", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFileExcluded(tt.rules, tt.path, tt.isDir); got != tt.want {
				t.Errorf("isFileExcluded(%q, %q, %v) = %v, want %v", tt.rules, tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}

func TestGetGitIgnoreContent(t *testing.T) {
	t.Run("returns only rules after the separator", func(t *testing.T) {
		root := newTestRepo(t, ".DS_Store\n*.swp\n#GitLFSLite\n*.mp4\n\nmedia/\n# comment\n")

		got, err := getGitIgnoreContent(root)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{"*.mp4", "media/", "# comment"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("no separator returns no rules", func(t *testing.T) {
		root := newTestRepo(t, "*.mp4\n")

		got, err := getGitIgnoreContent(root)
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 0 {
			t.Errorf("got %q, want no rules", got)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		if _, err := getGitIgnoreContent(t.TempDir()); err == nil {
			t.Error("expected an error for a missing .gitignore")
		}
	})
}

func TestCreateGitIgnoreFile(t *testing.T) {
	root := t.TempDir()

	if err := createGitIgnoreFile(root); err != nil {
		t.Fatal(err)
	}

	rules, err := getGitIgnoreContent(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(rules) != 0 {
		t.Errorf("new .gitignore should have no glflite rules, got %q", rules)
	}

	if !hasGitIgnoreFile(root) {
		t.Error("hasGitIgnoreFile should be true after creating it")
	}
}

func TestGLFLitePathHelpers(t *testing.T) {
	if got := getGLFLiteFilePath("media/video.mp4"); got != "media/video.mp4.glflite" {
		t.Errorf("getGLFLiteFilePath = %q", got)
	}

	if got := getTrackedFilePath("media/video.mp4.glflite"); got != "media/video.mp4" {
		t.Errorf("getTrackedFilePath = %q", got)
	}

	if !isGLFLiteFile("media/video.mp4.glflite") {
		t.Error("isGLFLiteFile should be true for .glflite files")
	}

	if isGLFLiteFile("media/video.mp4") {
		t.Error("isGLFLiteFile should be false for other files")
	}
}

func TestFindGitFolder(t *testing.T) {
	t.Run("from repo root", func(t *testing.T) {
		root := newTestRepo(t, "")
		chdir(t, root)

		got, err := findGitFolder()
		if err != nil {
			t.Fatal(err)
		}

		if got != root {
			t.Errorf("got %q, want %q", got, root)
		}
	})

	t.Run("from nested subfolder", func(t *testing.T) {
		root := newTestRepo(t, "")
		sub := filepath.Join(root, "a", "b")
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatal(err)
		}
		chdir(t, sub)

		got, err := findGitFolder()
		if err != nil {
			t.Fatal(err)
		}

		if got != root {
			t.Errorf("got %q, want %q", got, root)
		}
	})

	t.Run("not in a repo", func(t *testing.T) {
		chdir(t, t.TempDir())

		if _, err := findGitFolder(); err == nil {
			t.Error("expected an error outside a git repository")
		}
	})
}

func TestFindAllFilesAndFolders(t *testing.T) {
	root := newTestRepo(t, "#GitLFSLite\n")
	writeTestFile(t, root, ".git/config", "")
	writeTestFile(t, root, setupFile, "")
	writeTestFile(t, root, "video.mp4", "data")
	writeTestFile(t, root, "media/song.mp3", "data")

	files, err := findAllFilesAndFolders(root)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, f := range files {
		got = append(got, f.path)
	}
	sort.Strings(got)

	want := []string{"media", "media/song.mp3", "video.mp4"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")

	writeTestFile(t, root, "target", "data")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if !isLink(link) {
		t.Error("isLink should be true for a symlink")
	}

	if isLink(target) {
		t.Error("isLink should be false for a regular file")
	}
}

func TestGetFileShasum(t *testing.T) {
	root := newTestRepo(t, "")
	writeTestFile(t, root, "media/hello.bin", "hello")
	app := newTestApp(root)

	got, err := app.getFileShasum("media/hello.bin")
	if err != nil {
		t.Fatal(err)
	}

	if got != helloShasum {
		t.Errorf("got %s, want %s", got, helloShasum)
	}

	if _, err := app.getFileShasum("missing.bin"); err == nil {
		t.Error("expected an error for a missing file")
	}
}

// Regression test: paths are relative to the repo root, so they must resolve
// correctly when the working directory is a subfolder of the repository.
func TestGetFileShasumFromSubfolder(t *testing.T) {
	root := newTestRepo(t, "")
	writeTestFile(t, root, "hello.bin", "hello")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	chdir(t, filepath.Join(root, "sub"))

	got, err := newTestApp(root).getFileShasum("hello.bin")
	if err != nil {
		t.Fatal(err)
	}

	if got != helloShasum {
		t.Errorf("got %s, want %s", got, helloShasum)
	}
}

func TestGenerateRsyncFileList(t *testing.T) {
	root := newTestRepo(t, "")
	chdir(t, t.TempDir())

	app := newTestApp(root)
	addTrackedFile(app, "a.mp4", true, "", 1)
	addTrackedFile(app, "b.mp4", false, "", 1)

	if err := app.generateRsyncFileList(false); err != nil {
		t.Fatal(err)
	}

	if err := app.generateRsyncFileList(true); err != nil {
		t.Fatal(err)
	}

	if got := readTestFile(t, filepath.Join(root, "rsync_list_glflite")); got != "./a.mp4\n./b.mp4\n" {
		t.Errorf("rsync_list_glflite = %q", got)
	}

	if got := readTestFile(t, filepath.Join(root, "rsync_list_glflite_local")); got != "./a.mp4\n" {
		t.Errorf("rsync_list_glflite_local = %q", got)
	}
}

func TestGenerateSha256FileList(t *testing.T) {
	root := newTestRepo(t, "")
	chdir(t, t.TempDir())

	app := newTestApp(root)
	addTrackedFile(app, "a.mp4", true, "bbb", 10)
	addTrackedFile(app, "b.mp4", true, "aaa", 20)
	addTrackedFile(app, "c.mp4", true, "bbb", 30)
	addTrackedFile(app, "untracked.mp4", true, "", 40)

	if err := app.generateSha256FileList(); err != nil {
		t.Fatal(err)
	}

	want := "aaa  ./b.mp4\nbbb  ./a.mp4\nbbb  ./c.mp4"
	if got := readTestFile(t, filepath.Join(root, "sha256_list_glflite")); got != want {
		t.Errorf("sha256_list_glflite = %q, want %q", got, want)
	}

	if got := app.duplicatedFiles["bbb"]; !reflect.DeepEqual(got, []string{"a.mp4", "c.mp4"}) {
		t.Errorf("duplicatedFiles[bbb] = %q", got)
	}

	if len(app.duplicatedFiles) != 1 {
		t.Errorf("expected 1 duplicated group, got %d", len(app.duplicatedFiles))
	}

	if app.duplicatedTotalSize != 30 {
		t.Errorf("duplicatedTotalSize = %d, want 30", app.duplicatedTotalSize)
	}
}
