package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) {
	source1 := args[0]
	source2 := args[1]

	reader1, err := zip.OpenReader(source1)
	if err != nil {
		panic(err)
	}
	defer reader1.Close()

	reader2, err := zip.OpenReader(source2)
	if err != nil {
		panic(err)
	}
	defer reader2.Close()

	dir, err := ioutil.TempDir("", "office-diff_")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	for _, f := range reader1.File {
		err = unzipFile(f, path.Join(dir, "source"))
		if err != nil {
			panic(err)
		}
	}

	for _, f := range reader2.File {
		err = unzipFile(f, path.Join(dir, "target"))
		if err != nil {
			panic(err)
		}
	}

	addedFiles := make([]string, 0)
	existingFiles := make([]string, 0)
	removedFiles := make([]string, 0)

	err = filepath.Walk(path.Join(dir, "source"),
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, "source", "target", 1)); errors.Is(err, os.ErrNotExist) {
				removedFiles = append(removedFiles, p)
				return nil
			}

			existingFiles = append(existingFiles, p)

			return nil
		})

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(path.Join(dir, "target"),
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, "target", "source", 1)); errors.Is(err, os.ErrNotExist) {
				addedFiles = append(addedFiles, p)
				return nil
			}

			return nil
		})

	if err != nil {
		panic(err)
	}

	combinedDiff := ""

	for _, p := range addedFiles {
		ext := filepath.Ext(p)

		contentsArr, err := ioutil.ReadFile(p)
		if err != nil {
			continue
		}

		contents := string(contentsArr)

		if ext == ".xml" {
			contents = xmlfmt.FormatXML(contents, "", "  ")
		}

		diffPath1 := strings.Replace(p, dir, "", 1)
		diffPath1 = strings.Replace(diffPath1, "target", "source", 1)
		diffPath2 := strings.Replace(p, dir, "", 1)

		edits := myers.ComputeEdits(span.URIFromPath("/dev/null"), "", contents)
		diff := fmt.Sprint(gotextdiff.ToUnified("/dev/null", diffPath2, "", edits))

		combinedDiff += fmt.Sprintf("diff %s %s\n", diffPath1, diffPath2)
		combinedDiff += diff
	}
	for _, p := range existingFiles {
		ext := filepath.Ext(p)

		p1 := p
		p2 := strings.Replace(p, "source", "target", 1)

		contentsArr1, err := ioutil.ReadFile(p1)
		if err != nil {
			continue
		}

		contentsArr2, err := ioutil.ReadFile(p2)
		if err != nil {
			continue
		}

		contents1 := string(contentsArr1)
		contents2 := string(contentsArr2)

		if ext == ".xml" {
			contents1 = xmlfmt.FormatXML(contents1, "", "  ")
		}
		if ext == ".xml" {
			contents2 = xmlfmt.FormatXML(contents2, "", "  ")
		}

		diffPath1 := strings.Replace(p1, dir, "", 1)
		diffPath2 := strings.Replace(p2, dir, "", 1)

		edits := myers.ComputeEdits(span.URIFromPath(diffPath1), contents1, contents2)
		diff := fmt.Sprint(gotextdiff.ToUnified(diffPath1, diffPath2, contents1, edits))

		if diff == "" {
			continue
		}

		combinedDiff += fmt.Sprintf("diff %s %s\n", diffPath1, diffPath2)
		combinedDiff += diff
	}
	for _, p := range removedFiles {
		ext := filepath.Ext(p)

		contentsArr, err := ioutil.ReadFile(p)
		if err != nil {
			continue
		}

		contents := string(contentsArr)

		if ext == ".xml" {
			contents = xmlfmt.FormatXML(contents, "", "  ")
		}

		diffPath1 := strings.Replace(p, dir, "", 1)
		diffPath2 := strings.Replace(diffPath1, "source", "target", 1)

		edits := myers.ComputeEdits(span.URIFromPath(diffPath1), contents, "")
		diff := fmt.Sprint(gotextdiff.ToUnified(diffPath1, "/dev/null", contents, edits))

		combinedDiff += fmt.Sprintf("diff %s %s\n", diffPath1, diffPath2)
		combinedDiff += diff
	}

	if combinedDiff == "" {
		fmt.Println("files are equal")
		return
	}

	err = ioutil.WriteFile("result.diff", []byte(combinedDiff), 0755)

	if err != nil {
		panic(err)
	}

	fmt.Printf("done")
}

func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

func Execute() {
	rootCmd := &cobra.Command{
		Use:               "office-diff <file> <file>",
		Short:             "Diff tool for OpenXML Office files",
		Run:               run,
		Args:              cobra.ExactArgs(2),
		DisableAutoGenTag: true,
		Version:           "0.0.1", // TODO: read version from build
	}

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
