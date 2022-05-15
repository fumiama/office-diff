package format

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

// https://github.com/sibprogrammer/xq/blob/master/internal/utils/utils.go

func Xml(reader io.Reader, writer io.Writer, indent string) error {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		e, err := ianaindex.MIME.Encoding(charset)
		if err != nil {
			return nil, err
		}
		return transform.NewReader(input, e.NewDecoder()), nil
	}

	level := 0
	hasContent := false
	nsAliases := map[string]string{}
	lastTagName := ""
	startTagClosed := true

	for {
		token, err := decoder.Token()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch typedToken := token.(type) {
		case xml.ProcInst:
			_, _ = fmt.Fprintf(writer, "<?%s", typedToken.Target)

			pi := strings.TrimSpace(string(typedToken.Inst))
			attrs := strings.Split(pi, " ")
			for _, attr := range attrs {
				attrComponents := strings.SplitN(attr, "=", 2)
				_, _ = fmt.Fprintf(writer, " %s%s", attrComponents[0], "="+attrComponents[1])
			}

			_, _ = fmt.Fprint(writer, "?>\n")
		case xml.StartElement:
			if !startTagClosed {
				_, _ = fmt.Fprint(writer, ">")
				startTagClosed = true
			}
			if level > 0 {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
			}
			var attrs []string
			for _, attr := range typedToken.Attr {
				if attr.Name.Space == "xmlns" {
					nsAliases[attr.Value] = attr.Name.Local
				}
				if attr.Name.Local == "xmlns" {
					nsAliases[attr.Value] = ""
				}
				attrs = append(attrs, getTokenFullName(attr.Name, nsAliases)+"=\""+attr.Value+"\"")
			}
			attrsStr := strings.Join(attrs, " ")
			if attrsStr != "" {
				attrsStr = " " + attrsStr
			}
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			_, _ = fmt.Fprint(writer, "<"+currentTagName+attrsStr)
			lastTagName = currentTagName
			startTagClosed = false
			level++
		case xml.CharData:
			str := string(typedToken)
			str = strings.TrimSpace(str)
			hasContent = str != ""
			if hasContent && !startTagClosed {
				_, _ = fmt.Fprint(writer, ">")
				startTagClosed = true
			}
			_, _ = fmt.Fprint(writer, str)
		case xml.Comment:
			if !startTagClosed {
				_, _ = fmt.Fprint(writer, ">")
				startTagClosed = true
			}
			if !hasContent && level > 0 {
				_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level))
			}
			_, _ = fmt.Fprint(writer, "<!--"+string(typedToken)+"-->")
			if level == 0 {
				_, _ = fmt.Fprint(writer, "\n")
			}
		case xml.EndElement:
			level--
			currentTagName := getTokenFullName(typedToken.Name, nsAliases)
			if !hasContent {
				if lastTagName != currentTagName {
					if !startTagClosed {
						_, _ = fmt.Fprint(writer, ">")
						startTagClosed = true
					}
					_, _ = fmt.Fprint(writer, "\n", strings.Repeat(indent, level), "</"+currentTagName+">")
				} else {
					_, _ = fmt.Fprint(writer, "/>")
					startTagClosed = true
				}
			} else {
				_, _ = fmt.Fprint(writer, "</"+currentTagName+">")
			}
			hasContent = false
			lastTagName = currentTagName
		default:
		}
	}

	_, _ = fmt.Fprint(writer, "\n")

	return nil
}

func getTokenFullName(name xml.Name, nsAliases map[string]string) string {
	result := name.Local
	if name.Space != "" {
		space := name.Space
		if alias, ok := nsAliases[space]; ok {
			space = alias
		}
		if space != "" {
			result = space + ":" + name.Local
		}
	}
	return result
}
