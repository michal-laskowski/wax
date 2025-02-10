package wax

import (
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"
)

type templateResult string

type waxWriter struct {
	sb strings.Builder
}

func newWriter() *waxWriter {
	result := &waxWriter{
		sb: strings.Builder{},
	}

	return result
}

func (w *waxWriter) WriteRaw(v string) *waxWriter {
	w.sb.WriteString(v)
	return w
}

func (w *waxWriter) WriteHTML(v string) *waxWriter {
	w.sb.WriteString(v)
	return w
}

func (w *waxWriter) WriteAttribute(attributeName string, v any) *waxWriter {
	switch v := v.(type) {
	case bool:
		if v {
			w.sb.WriteString(attributeName)
		}

	case map[string]any:
		if attributeName == "wax-attrs" {

			for a, v := range v {
				w.WriteAttribute(a, v)
				w.sb.WriteString(" ")
			}
		} else if attributeName == "style" {
			var serializedStyles []string
			for pk, pv := range v {
				cssKey := camelToKebabCase(pk)
				style := fmt.Sprintf("%s: %v;", cssKey, pv)
				serializedStyles = append(serializedStyles, style)
			}
			w.sb.WriteString(fmt.Sprintf("%s=\"%s\"", attributeName, strings.Join(serializedStyles, " ")))
		} else {
			w.sb.WriteString(fmt.Sprintf("%s=\"[object Object]\"", attributeName))
		}

	case struct{}:
		w.sb.WriteString(fmt.Sprintf("%s=\"[object Object]\"", attributeName))

	case []interface{}:
		{
			values := ""
			separator := ","
			if attributeName == "class" {
				separator = " "
			}
			for i, iv := range v {
				if iv != false && iv != nil && iv != int64(0) {
					if i > 0 {
						values += separator
					}
					values += fmt.Sprintf("%v", iv)

				}
			}
			w.sb.WriteString(fmt.Sprintf("%s=\"%v\"", attributeName, values))

		}

	case time.Time:
		{
			toWrite := v.UTC().Format("2006-01-02T15:04:05.000Z")
			w.sb.WriteString(fmt.Sprintf("%s=\"%v\"", attributeName, toWrite))
		}

	case nil:
		{
		}

	default:
		toWrite := escapeHTML(v)
		w.sb.WriteString(fmt.Sprintf("%s=\"%v\"", attributeName, toWrite))
	}

	return w
}

func (w *waxWriter) WriteValue(v ...any) *waxWriter {
	if len(v) == 0 {
		return w
	}

	switch v := v[0].(type) {
	case templateResult:
		{
			w.sb.WriteString(string(v))
			return w
		}
	case bool:
		return w
	case nil:
		return w

	case float64:
		var toWrite string
		if math.IsInf(v, 1) {
			toWrite = ("Infinity")
		} else if math.IsInf(v, -1) {
			toWrite = ("-Infinity")
		} else {
			toWrite = escapeHTML(v)
		}
		w.sb.WriteString(toWrite)
		return w

	case []interface{}:
		{
			for _, o := range v {
				w.WriteValue(o)
			}
			return w
		}

	case string:
		{
			toWrite := escapeHTML(v)
			w.sb.WriteString(toWrite)
			return w
		}

	case *waxWriter:
		{
			return w
		}

	default:
		{
			toWrite := escapeHTML(v)
			w.sb.WriteString(toWrite)
			return w
		}
	}
}

func (w *waxWriter) ToString(v any) string {
	return w.sb.String()
}

func (w *waxWriter) Result(v any) templateResult {
	return templateResult(w.sb.String())
}

func escapeHTML(args ...any) string {
	result := template.HTMLEscaper(args...)
	result = strings.ReplaceAll(result, " ", "&nbsp;")
	return result
}

func camelToKebabCase(input string) string {
	var output strings.Builder
	for i, r := range input {
		if i > 0 && r >= 'A' && r <= 'Z' {
			output.WriteRune('-')
		}
		output.WriteRune(r)
	}
	return strings.ToLower(output.String())
}
