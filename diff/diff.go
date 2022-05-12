package diff

import (
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

type FileDiffOptions struct {
	SrcBasePath string
	DstBasePath string
	NoPrefix    bool
	SrcPrefix   string
	DstPrefix   string
}

const nullPath = "/dev/null"

func readFile(filename string) (string, error) {
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
		contents, err := readFile(dst)

		if err != nil {
			return "", err
		}

		edits := myers.ComputeEdits(span.URIFromPath(srcDisplay), "", contents)
		diff := fmt.Sprint(gotextdiff.ToUnified(srcDisplay, dstDisplay, "", edits))
		result += diff
	}

	if dst == "" { // removed file
		contents, err := readFile(src)
		if err != nil {
			return "", err
		}

		edits := myers.ComputeEdits(span.URIFromPath(srcDisplay), contents, "")
		diff := fmt.Sprint(gotextdiff.ToUnified(srcDisplay, dstDisplay, contents, edits))
		result += diff
	} else { // modified file
		contentsSrc, err := readFile(src)
		if err != nil {
			return "", err
		}

		contentsDst, err := readFile(dst)
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
