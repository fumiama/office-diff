package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/develerik/office-diff/diff"
	"github.com/develerik/office-diff/zip"
)

const (
	pathSrc = "a"
	pathDst = "b"
)

func run(_ *cobra.Command, args []string) {
	source := args[0]
	destination := args[1]

	dir, err := ioutil.TempDir("", "office-diff_")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	if err = zip.Extract(source, path.Join(dir, pathSrc)); err != nil {
		panic(err)
	}
	if err = zip.Extract(destination, path.Join(dir, pathDst)); err != nil {
		panic(err)
	}

	addedFiles := make([]string, 0)
	existingFiles := make([]string, 0)
	removedFiles := make([]string, 0)

	err = filepath.Walk(path.Join(dir, pathSrc),
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, pathSrc, pathDst, 1)); errors.Is(err, os.ErrNotExist) {
				removedFiles = append(removedFiles, p)
				return nil
			}

			existingFiles = append(existingFiles, p)

			return nil
		})

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(path.Join(dir, pathDst),
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, pathDst, pathSrc, 1)); errors.Is(err, os.ErrNotExist) {
				addedFiles = append(addedFiles, p)
				return nil
			}

			return nil
		})

	if err != nil {
		panic(err)
	}

	combinedDiff := ""
	options := diff.FileDiffOptions{
		SrcBasePath: path.Join(dir, pathSrc),
		DstBasePath: path.Join(dir, pathDst),
		SrcPrefix:   viper.GetString("src-prefix"),
		DstPrefix:   viper.GetString("dst-prefix"),
		NoPrefix:    viper.GetBool("no-prefix"),
	}

	for _, p := range addedFiles {
		partialDiff, err := diff.Files("", p, options)

		if err != nil {
			continue
		}

		combinedDiff += partialDiff
	}
	for _, p := range existingFiles {
		p2 := strings.Replace(p, pathSrc, pathDst, 1)

		partialDiff, err := diff.Files(p, p2, options)

		if err != nil || partialDiff == "" {
			continue
		}

		combinedDiff += partialDiff
	}
	for _, p := range removedFiles {
		partialDiff, err := diff.Files(p, "", options)

		if err != nil {
			continue
		}

		combinedDiff += partialDiff
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
