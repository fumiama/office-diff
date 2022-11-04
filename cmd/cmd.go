package cmd

import (
	"fmt"
	"os"
	"path"
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

type Options struct {
	Version string
	Date    string
}

func run(_ *cobra.Command, args []string) {
	dir, err := os.MkdirTemp("", "office-diff_")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	combinedDiff := ""
	options := diff.FileDiffOptions{
		SrcBasePath: path.Join(dir, pathSrc),
		DstBasePath: path.Join(dir, pathDst),
		SrcPrefix:   viper.GetString("src-prefix"),
		DstPrefix:   viper.GetString("dst-prefix"),
		NoPrefix:    viper.GetBool("no-prefix"),
	}

	if err := zip.Extract(args[0], options.SrcBasePath); err != nil {
		fmt.Printf("error: Could not access '%s'\n", args[0])
		os.Exit(1)
	}
	if err := zip.Extract(args[1], options.DstBasePath); err != nil {
		fmt.Printf("error: Could not access '%s'\n", args[1])
		os.Exit(1)
	}

	files, err := diff.Directories(options.SrcBasePath, options.DstBasePath)

	if err != nil {
		panic(err)
	}

	for _, p := range files["added"] {
		partialDiff, err := diff.Files("", p, options)

		if err != nil {
			continue
		}

		combinedDiff += partialDiff
	}
	for _, p := range files["existing"] {
		p2 := strings.Replace(p, options.SrcBasePath, options.DstBasePath, 1)

		partialDiff, err := diff.Files(p, p2, options)

		if err != nil || partialDiff == "" {
			continue
		}

		combinedDiff += partialDiff
	}
	for _, p := range files["removed"] {
		partialDiff, err := diff.Files(p, "", options)

		if err != nil {
			continue
		}

		combinedDiff += partialDiff
	}

	outputFile := viper.GetString("output")

	if combinedDiff == "" {
		if outputFile != "" {
			if err = os.WriteFile(outputFile, []byte(""), 0755); err != nil {
				panic(err)
			}
		}
		os.Exit(0)
	}

	if outputFile == "" {
		fmt.Print(combinedDiff)
	} else {
		if err = os.WriteFile(outputFile, []byte(combinedDiff), 0755); err != nil {
			panic(err)
		}
	}

	if viper.GetBool("exit-code") {
		os.Exit(1)
	}
}

func Execute(opts *Options) {
	rootCmd := &cobra.Command{
		Use:               "office-diff <file> <file>",
		Short:             "Show changes between OpenXML office files",
		Run:               run,
		Args:              cobra.ExactArgs(2),
		DisableAutoGenTag: true,
		Version:           fmt.Sprintf("%s (%s)", opts.Version, opts.Date),
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
