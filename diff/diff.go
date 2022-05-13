package diff

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

type FileDiffOptions struct {
	SrcBasePath string
	DstBasePath string
	NoPrefix    bool
	SrcPrefix   string
	DstPrefix   string
}

const nullPath = "/dev/null"

var fileTypes = map[string][]string{
	"xml": {".xml", ".xml.rels", ".rels"},
}

func isFileType(filename string, typeExts []string) bool {
	for _, ext := range typeExts {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}

	return false
}

func readTextFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	if isFileType(filename, fileTypes["xml"]) {
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
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, src, dst, 1)); errors.Is(err, os.ErrNotExist) {
				result["removed"] = append(result["removed"], p)
				return nil
			} else if err != nil {
				return err
			}

			result["existing"] = append(result["existing"], p)
			return nil
		})

	if err != nil {
		return nil, err
	}

	err = filepath.Walk(dst,
		func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if _, err = os.Stat(strings.Replace(p, dst, src, 1)); errors.Is(err, os.ErrNotExist) {
				result["added"] = append(result["added"], p)
				return nil
			} else if err != nil {
				return err
			}

			return nil
		})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func binaryIdentical(src, dst string) (bool, error) {
	srcR, err := os.Open(src)

	if err != nil {
		return false, err
	}

	defer func() {
		_ = srcR.Close()
	}()

	dstR, err := os.Open(dst)

	if err != nil {
		return false, err
	}

	defer func() {
		_ = dstR.Close()
	}()

	if _, err = io.Copy(ioutil.Discard, NewCompareReader(srcR, dstR)); err != nil {
		return false, nil
	}

	return true, nil
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

	if !isFileType(dst, fileTypes["xml"]) {
		if identical, err := binaryIdentical(src, dst); err != nil {
			return "", err
		} else if !identical {
			result += fmt.Sprintf("Binary files %s and %s differ\n", srcDisplay, dstDisplay)
			return result, nil
		}
	}

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
