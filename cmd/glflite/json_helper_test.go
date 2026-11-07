package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestReadWriteJSONFile(t *testing.T) {
	root := newTestRepo(t, "")
	chdir(t, t.TempDir())
	app := newTestApp(root)

	if _, err := app.readJSONFile("media/video.mp4"); !errors.Is(err, ErrGLFLiteFileNotFound) {
		t.Fatalf("expected ErrGLFLiteFileNotFound, got %v", err)
	}

	data := fileData{
		FilePath:     "media/video.mp4",
		TrackedSince: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		LastModified: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		Size:         5,
		Sha256Sum:    helloShasum,
	}

	writeTestFile(t, root, "media/.keep", "")
	if err := app.writeJSONFile("media/video.mp4", data); err != nil {
		t.Fatal(err)
	}

	if !fileExists(filepath.Join(root, "media/video.mp4.glflite")) {
		t.Fatal("expected the .glflite file to be written in the repo root")
	}

	got, err := app.readJSONFile("media/video.mp4")
	if err != nil {
		t.Fatal(err)
	}

	if got != data {
		t.Errorf("got %+v, want %+v", got, data)
	}
}

func TestReadJSONFileInvalid(t *testing.T) {
	root := newTestRepo(t, "")
	writeTestFile(t, root, "video.mp4.glflite", "not json")

	if _, err := newTestApp(root).readJSONFile("video.mp4"); err == nil || errors.Is(err, ErrGLFLiteFileNotFound) {
		t.Errorf("expected a JSON parse error, got %v", err)
	}
}
