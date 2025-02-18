package wax

import (
	"fmt"
	"html/template"
	"math"
	"math/big"
	"strconv"
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

func (w *waxWriter) WriteAttributes(v any) *waxWriter {
	if toWrite, ok := v.(map[string]any); !ok {
		panic("expected to get map")
	} else {
		for a, v := range toWrite {
			if a == "children" {
				continue
			}
			w.WriteAttribute(a, v)
			w.sb.WriteString(" ")
		}
	}
	return w
}

func (w *waxWriter) WriteAttribute(attributeName string, v any) *waxWriter {
	switch v := v.(type) {

	case nil:
		{
		}
	case bool:
		if isBoolEnumerableAttribute(attributeName) {
			w.sb.WriteString(attributeName)
			w.sb.WriteString("=\"")
			if v {
				w.sb.WriteString("true")
			} else {
				w.sb.WriteString("false")
			}
			w.sb.WriteString("\"")
		} else {
			if v {
				w.sb.WriteString(attributeName)
			}
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
				if _, isString := pv.(string); !isString {
					continue
				}
				if pv == "" || pv == nil {
					continue
				}
				cssKey := camelToKebabCase(pk)
				valueToWrite := pv.(string)
				//valueToWrite = strings.ReplaceAll(valueToWrite,"'", `\27`)
				valueToWrite = strings.ReplaceAll(valueToWrite, "\"", `\22`)
				valueToWrite = strings.ReplaceAll(valueToWrite, ";", `\3B`)

				style := fmt.Sprintf("%s: %v", cssKey, valueToWrite)
				serializedStyles = append(serializedStyles, style)
			}
			if len(serializedStyles) > 0 {
				w.sb.WriteString(fmt.Sprintf("%s=\"%s;\"", attributeName, strings.Join(serializedStyles, "; ")))
			}
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

	default:
		toWrite := getStringRepresentation(v)
		w.sb.WriteString(fmt.Sprintf("%s=\"%v\"", attributeName, toWrite))
	}

	return w
}

func (w *waxWriter) WriteValue(v ...any) *waxWriter {
	if len(v) == 0 {
		return w
	}
	if len(v) > 1 {
		panic("expected to get single value")
	}

	switch v := v[0].(type) {
	case templateResult:
		{
			w.sb.WriteString(string(v))
		}
	case bool:
		{
			//noop
		}
	case nil:
		{
			//noop
		}
	case *waxWriter:
		{
			//noop
		}
	case []interface{}:
		{
			for _, o := range v {
				w.WriteValue(o)
			}
		}
	default:
		{
			toWrite := getStringRepresentation(v)
			w.sb.WriteString(toWrite)
		}
	}
	return w
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

type safe_str = string

func getStringRepresentation(v any) safe_str {
	switch v := v.(type) {
	case time.Time:
		return v.UTC().Format("2006-01-02T15:04:05.000Z")
	case int:
		return (strconv.Itoa(v))
	case *int:
		return (strconv.Itoa(*v))
	case int8:
		return (strconv.FormatInt(int64(v), 10))
	case *int8:
		return (strconv.FormatInt(int64(*v), 10))
	case int16:
		return (strconv.FormatInt(int64(v), 10))
	case *int16:
		return (strconv.FormatInt(int64(*v), 10))
	case int32:
		return (strconv.FormatInt(int64(v), 10))
	case *int32:
		return (strconv.FormatInt(int64(*v), 10))
	case int64:
		return (strconv.FormatInt(v, 10))
	case *int64:
		return (strconv.FormatInt(*v, 10))
	case uint:
		return (strconv.FormatUint(uint64(v), 10))
	case *uint:
		return (strconv.FormatUint(uint64(*v), 10))
	case uint8:
		return (strconv.FormatUint(uint64(v), 10))
	case *uint8:
		return (strconv.FormatUint(uint64(*v), 10))
	case uint16:
		return (strconv.FormatUint(uint64(v), 10))
	case *uint16:
		return (strconv.FormatUint(uint64(*v), 10))
	case uint32:
		return (strconv.FormatUint(uint64(v), 10))
	case *uint32:
		return (strconv.FormatUint(uint64(*v), 10))
	case uint64:
		return (strconv.FormatUint(v, 10))
	case *uint64:
		return (strconv.FormatUint(*v, 10))
	case float32:
		return (strconv.FormatFloat(float64(v), 'f', -1, 32))
	case *float32:
		return (strconv.FormatFloat(float64(*v), 'f', -1, 32))
	case float64:
		if math.IsInf(v, 1) {
			return ("Infinity")
		} else if math.IsInf(v, -1) {
			return ("-Infinity")
		} else {
			return (strconv.FormatFloat(v, 'f', -1, 64))
		}
	case *float64:
		return (strconv.FormatFloat(*v, 'f', -1, 64))
	case big.Int:
		return (v.String())
	case *big.Int:
		return (v.String())
	case string, *string:
		{
			toWrite := escapeHTML(v)
			return (toWrite)
		}
	default:
		{
			toWrite := escapeHTML(v)
			return (toWrite)
		}
	}
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

func isBoolEnumerableAttribute(attributeName string) bool {
	switch true {
	case attributeName == "draggable":
		return true
	case attributeName == "spellcheck":
		return true
	case strings.HasPrefix(attributeName, "aria-"):
		return true
	}
	return false
}
