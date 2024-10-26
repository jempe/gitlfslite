package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"
)

var ErrGLFLiteFileNotFound = errors.New("GLFLite file not found")

type fileData struct {
	FilePath     string    `json:"file_path"`
	TrackedSince time.Time `json:"tracked_since"`
	LastModified time.Time `json:"last_modified"`
	Size         int64     `json:"size"`
	Sha256Sum    string    `json:"sha256sum"`
}

func (app *application) getFullPath(filePath string) string {
	return app.config.rootFolder + "/" + filePath
}

func (app *application) readJSONFile(filePath string) (fileData, error) {
	var data fileData

	glfFile := getGLFLiteFilePath(filePath)

	if !fileExists(app.getFullPath(glfFile)) {
		return data, ErrGLFLiteFileNotFound
	}

	jsonData, err := ioutil.ReadFile(app.getFullPath(glfFile))

	if err != nil {
		return data, err
	}

	err = json.Unmarshal(jsonData, &data)

	if err != nil {
		return data, err
	}

	return data, nil
}

func (app *application) writeJSONFile(filePath string, data fileData) error {
	glfFile := getGLFLiteFilePath(filePath)

	if fileExists(app.getFullPath(glfFile)) {
		_, err := app.readJSONFile(filePath)

		if err != nil {
			return err
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "\t")

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(app.getFullPath(glfFile), jsonData, 0644)

	if err != nil {
		return err
	}

	return nil
}
