package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unzipFile(f *zip.File, destination string) error {
	// check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	prefix := filepath.Clean(destination) + string(os.PathSeparator)
	if !strings.HasPrefix(filePath, prefix) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// create directory tree
	if f.FileInfo().IsDir() {
		return os.MkdirAll(filePath, os.ModePerm)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer func() {
		_ = destinationFile.Close()
	}()

	// unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = zippedFile.Close()
	}()

	_, err = io.Copy(destinationFile, zippedFile)
	return err
}

func Extract(source, target string) error {
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()

	for _, f := range r.File {
		err = unzipFile(f, target)
		if err != nil {
			return err
		}
	}

	return nil
}
