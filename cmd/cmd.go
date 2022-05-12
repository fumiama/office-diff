package cmd

import (
	"errors"
	"fmt"
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
	"github.com/spf13/viper"

	"github.com/develerik/office-diff/zip"
)

const (
	pathSource = "a"
	pathTarget = "b"
	pathNull   = "/dev/null"
)

func run(_ *cobra.Command, args []string) {
	source1 := args[0]
	source2 := args[1]

	dir, err := ioutil.TempDir("", "office-diff_")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	if err = zip.Extract(source1, path.Join(dir, pathSource)); err != nil {
		panic(err)
	}
	if err = zip.Extract(source2, path.Join(dir, pathTarget)); err != nil {
		panic(err)
	}

	addedFiles := make([]string, 0)
	existingFiles := make([]string, 0)
	removedFiles := make([]string, 0)

	err = filepath.Walk(path.Join(dir, pathSource),
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, pathSource, pathTarget, 1)); errors.Is(err, os.ErrNotExist) {
				removedFiles = append(removedFiles, p)
				return nil
			}

			existingFiles = append(existingFiles, p)

			return nil
		})

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(path.Join(dir, pathTarget),
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, pathTarget, pathSource, 1)); errors.Is(err, os.ErrNotExist) {
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

		diffPath1 := strings.Replace(p, dir+string(os.PathSeparator), "", 1)
		diffPath1 = strings.Replace(diffPath1, pathTarget, pathSource, 1)
		diffPath2 := strings.Replace(p, dir+string(os.PathSeparator), "", 1)

		edits := myers.ComputeEdits(span.URIFromPath(pathNull), "", contents)
		diff := fmt.Sprint(gotextdiff.ToUnified(pathNull, diffPath2, "", edits))

		combinedDiff += fmt.Sprintf("diff %s %s\n", diffPath1, diffPath2)
		combinedDiff += diff
	}
	for _, p := range existingFiles {
		ext := filepath.Ext(p)

		p1 := p
		p2 := strings.Replace(p, pathSource, pathTarget, 1)

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

		diffPath1 := strings.Replace(p1, dir+string(os.PathSeparator), "", 1)
		diffPath2 := strings.Replace(p2, dir+string(os.PathSeparator), "", 1)

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

		diffPath1 := strings.Replace(p, dir+string(os.PathSeparator), "", 1)
		diffPath2 := strings.Replace(diffPath1, pathSource, pathTarget, 1)

		edits := myers.ComputeEdits(span.URIFromPath(diffPath1), contents, "")
		diff := fmt.Sprint(gotextdiff.ToUnified(diffPath1, pathNull, contents, edits))

		combinedDiff += fmt.Sprintf("diff %s %s\n", diffPath1, diffPath2)
		combinedDiff += diff
	}

	outputFile := viper.GetString("output")

	if combinedDiff == "" {
		if outputFile != "" {
			if err = ioutil.WriteFile(outputFile, []byte(""), 0755); err != nil {
				panic(err)
			}
		}
		os.Exit(0)
	}

	if outputFile == "" {
		fmt.Print(combinedDiff)
	} else {
		if err = ioutil.WriteFile(outputFile, []byte(combinedDiff), 0755); err != nil {
			panic(err)
		}
	}

	if viper.GetBool("exit-code") {
		os.Exit(1)
	}
}

func Execute() {
	rootCmd := &cobra.Command{
		Use:               "office-diff <file> <file>",
		Short:             "Show changes between OpenXML office files",
		Long:              "",
		Run:               run,
		Args:              cobra.ExactArgs(2),
		DisableAutoGenTag: true,
		Version:           "0.0.1", // TODO: read version from build
	}

	rootCmd.Flags().String("output", "", "Output to a specific file instead of stdout.")
	rootCmd.Flags().String("src-prefix", "a/", "[WIP] Show the given source prefix instead of 'a/'.")
	rootCmd.Flags().String("dst-prefix", "b/", "[WIP] Show the given destination prefix instead of 'b/'.")
	rootCmd.Flags().Bool("no-prefix", false, "[WIP] Do not show any source or destination prefix.")
	rootCmd.Flags().Bool("exit-code", false, `Make the program exit with codes similar to diff(1). That is, it exits with 1 if
there were differences and 0 means no differences.`)

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
