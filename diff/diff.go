package diff

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

// TODO: don't add binary content to diff (see git example)

type FileDiffOptions struct {
	SrcBasePath string
	DstBasePath string
	NoPrefix    bool
	SrcPrefix   string
	DstPrefix   string
}

const nullPath = "/dev/null"

func readTextFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(filename)

	if ext == ".xml" {
		return xmlfmt.FormatXML(string(data), "", "  "), nil
	} else {
		return string(data), nil
	}
}

func Directories(src, dst string) (map[string][]string, error) {
	result := make(map[string][]string, 0)
	result["added"] = make([]string, 0)
	result["existing"] = make([]string, 0)
	result["removed"] = make([]string, 0)

	err := filepath.Walk(src,
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil // TODO: remove when blob handling is implemented
			}

			if _, err = os.Stat(strings.Replace(p, src, dst, 1)); errors.Is(err, os.ErrNotExist) {
				result["removed"] = append(result["removed"], p)
				return nil
			}

			result["existing"] = append(result["existing"], p)
			return nil
		})

	if err != nil {
		return nil, err
	}

	err = filepath.Walk(dst,
		func(p string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(p) != ".xml" {
				return nil // TODO: remove when blob handling is implemented
			}

			if _, err = os.Stat(strings.Replace(p, dst, src, 1)); errors.Is(err, os.ErrNotExist) {
				result["added"] = append(result["added"], p)
				return nil
			}

			return nil
		})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func Files(src, dst string, opts FileDiffOptions) (string, error) {
	srcDisplay := src
	dstDisplay := dst

	if src == "" {
		srcDisplay = nullPath
	} else {
		srcDisplay = strings.TrimPrefix(srcDisplay, opts.SrcBasePath+string(os.PathSeparator))

		if !opts.NoPrefix {
			srcDisplay = fmt.Sprintf("%s%s", opts.SrcPrefix, srcDisplay)
		}
	}

	if dst == "" {
		dstDisplay = nullPath
	} else {
		dstDisplay = strings.TrimPrefix(dstDisplay, opts.DstBasePath+string(os.PathSeparator))

		if !opts.NoPrefix {
			dstDisplay = fmt.Sprintf("%s%s", opts.DstPrefix, dstDisplay)
		}
	}

	result := fmt.Sprintf("diff %s %s\n", srcDisplay, dstDisplay)

	if src == "" { // added file
		contents, err := readTextFile(dst)

		if err != nil {
			return "", err
		}

		edits := myers.ComputeEdits(span.URIFromPath(srcDisplay), "", contents)
		diff := fmt.Sprint(gotextdiff.ToUnified(srcDisplay, dstDisplay, "", edits))
		result += diff
	}

	if dst == "" { // removed file
		contents, err := readTextFile(src)
		if err != nil {
			return "", err
		}

		edits := myers.ComputeEdits(span.URIFromPath(srcDisplay), contents, "")
		diff := fmt.Sprint(gotextdiff.ToUnified(srcDisplay, dstDisplay, contents, edits))
		result += diff
	} else { // existing file
		contentsSrc, err := readTextFile(src)
		if err != nil {
			return "", err
		}

		contentsDst, err := readTextFile(dst)
		if err != nil {
			return "", err
		}

		edits := myers.ComputeEdits(span.URIFromPath(srcDisplay), contentsSrc, contentsDst)
		diff := fmt.Sprint(gotextdiff.ToUnified(srcDisplay, dstDisplay, contentsSrc, edits))

		if diff == "" {
			return "", nil
		}

		result += diff
	}

	return result, nil
}
