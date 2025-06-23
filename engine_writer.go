package wax

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type templateResult string

type waxWriter struct {
	jsObj goja.Value
	vm    *goja.Runtime
	out   io.Writer
}

func newWriter(out io.Writer, vm *goja.Runtime) *waxWriter {
	o := vm.NewObject()
	result := &waxWriter{
		out:   out,
		vm:    vm,
		jsObj: o,
	}
	o.DefineDataProperty("WriteHTML", vm.ToValue(result.writeHTML), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
	o.DefineDataProperty("WriteValue", vm.ToValue(result.writeValue), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
	o.DefineDataProperty("WriteAttribute", vm.ToValue(result.writeAttribute), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
	o.DefineDataProperty("WriteAttributes", vm.ToValue(result.writeAttributes), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
	return result
}

var (
	reflectTypeString         = reflect.TypeOf("")
	reflectTypeTemplateResult = reflect.TypeOf((*templateResult)(nil)).Elem()
	reflectTypeSlice          = reflect.TypeOf((*[]interface{})(nil)).Elem()
)

func (w *waxWriter) process(arg goja.Value, vm *goja.Runtime) {
	switch arg.ExportType() {
	case reflectTypeString:
		toWrite := escapeHTML(arg.String())
		w.WriteRaw(toWrite)
	case reflectTypeTemplateResult:
		w.WriteRaw(arg.String())
	case reflectTypeSlice:
		vm.ForOf(arg, func(v goja.Value) bool {
			w.process(v, vm)
			return true
		})
	default:
		w.callSub(arg)
	}
}

func (w *waxWriter) callSub(arg goja.Value) {
	if subCall, ok := goja.AssertFunction(arg); !ok {
		w.WriteValue(arg.Export())
	} else {
		_, err := subCall(nil, w.jsObj, w.jsObj)
		if err != nil {
			panic(err)
		}
	}
}

func (w *waxWriter) WriteRaw(v string) {
	io.WriteString(w.out, v)
}

func (w *waxWriter) writeHTML(fc goja.FunctionCall) goja.Value {
	arg := fc.Arguments[0]
	switch arg.ExportType() {
	case reflectTypeString:
		w.WriteHTML(arg.String())
	default:
		panic("expected to ge string")
	}
	return fc.This
}

func (w *waxWriter) writeValue(fc goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(fc.Arguments) == 0 {
		return fc.This
	}
	arg := fc.Arguments[0]
	w.process(arg, vm)
	return fc.This
}

func (w *waxWriter) writeAttribute(fc goja.FunctionCall) goja.Value {
	name := fc.Arguments[0]
	v := fc.Arguments[1]
	err := w.WriteAttribute(name.String(), v.Export())
	if err != nil {
		w.vm.Interrupt(err)
	}
	return fc.This
}

func (w *waxWriter) writeAttributes(fc goja.FunctionCall) goja.Value {
	v := fc.Arguments[0]
	err := w.WriteAttributes(v.Export())
	if err != nil {
		w.vm.Interrupt(err)
	}
	return fc.This
}

func (w *waxWriter) WriteHTML(v string) {
	w.WriteRaw(v)
}

func (w *waxWriter) WriteAttributes(v any) error {
	if toWrite, ok := v.(map[string]any); !ok {
		panic("expected to get map")
	} else {
		keys := make([]string, 0, len(toWrite))
		for k := range toWrite {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, pk := range keys {
			pv := toWrite[pk]
			if pk == "children" {
				continue
			}
			err := w.WriteAttribute(pk, pv)
			if err != nil {
				return err
			}
			w.WriteRaw(" ")
		}
	}
	return nil
}

func (w *waxWriter) WriteAttribute(attributeName string, v any) error {
	switch v := v.(type) {
	case nil:
		{
		}
	case bool:
		if isBoolEnumerableAttribute(attributeName) {
			w.WriteRaw(attributeName)
			w.WriteRaw("=\"")
			if v {
				w.WriteRaw("true")
			} else {
				w.WriteRaw("false")
			}
			w.WriteRaw("\"")
		} else {
			if v {
				w.WriteRaw(attributeName)
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if attributeName == "wax-attrs" {
			for _, pk := range keys {

				w.WriteAttribute(pk, v[pk])
				w.WriteRaw(" ")
			}
		} else if attributeName == "style" {
			var styles strings.Builder

			for i, pk := range keys {
				pv := v[pk]
				if _, isString := pv.(string); !isString {
					continue
				}
				if pv == "" || pv == nil {
					continue
				}
				cssKey := camelToKebabCase(pk)
				valueToWrite := pv.(string)
				// valueToWrite = strings.ReplaceAll(valueToWrite,"'", `\27`)
				valueToWrite = strings.ReplaceAll(valueToWrite, "\"", `\22`)
				valueToWrite = strings.ReplaceAll(valueToWrite, ";", `\3B`)

				styles.WriteString(cssKey)
				styles.WriteString(": ")
				styles.WriteString(valueToWrite)

				if i < len(keys)-1 {
					styles.WriteString(";")
				}
			}
			if styles.Len() > 0 {
				w.WriteRaw(attributeName)
				w.WriteRaw("=\"")
				w.WriteRaw(styles.String())
				w.WriteRaw("\"")
			}

		} else {
			w.WriteRaw(fmt.Sprintf("%s=\"[object Object]\"", attributeName))
		}
	case struct{}:
		w.WriteRaw(fmt.Sprintf("%s=\"[object Object]\"", attributeName))
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
			w.WriteRaw(fmt.Sprintf("%s=\"%v\"", attributeName, values))
		}
	default:
		toWrite, err := getStringRepresentation(v)
		if err != nil {
			return err
		}
		w.WriteRaw(attributeName)
		w.WriteRaw("=")
		w.WriteRaw("\"")
		w.WriteRaw(toWrite)
		w.WriteRaw("\"")
	}
	return nil
}

func (w *waxWriter) WriteValue(v any) error {
	switch v := v.(type) {
	case templateResult:
		{
			w.WriteRaw(string(v))
		}
	case string:
		{
			toWrite := escapeHTML(v)
			w.WriteRaw(toWrite)
		}
	case bool, nil:
		{
			// noop
		}
	case []interface{}:
		{
			for _, o := range v {
				w.WriteValue(o)
			}
		}
	default:
		{
			toWrite, err := getStringRepresentation(v)
			if err != nil {
				return err
			}
			w.WriteRaw(toWrite)
		}
	}
	return nil
}

func escapeHTML(args ...any) string {
	result := template.HTMLEscaper(args...)
	result = strings.ReplaceAll(result, " ", "&nbsp;")
	return result
}

type safeString = string

func getStringRepresentation(v any) (safeString, error) {
	switch v := v.(type) {
	case time.Time:
		return v.UTC().Format("2006-01-02T15:04:05.000Z"), nil
	case int:
		return (strconv.Itoa(v)), nil
	case *int:
		return (strconv.Itoa(*v)), nil
	case int8:
		return (strconv.FormatInt(int64(v), 10)), nil
	case *int8:
		return (strconv.FormatInt(int64(*v), 10)), nil
	case int16:
		return (strconv.FormatInt(int64(v), 10)), nil
	case *int16:
		return (strconv.FormatInt(int64(*v), 10)), nil
	case int32:
		return (strconv.FormatInt(int64(v), 10)), nil
	case *int32:
		return (strconv.FormatInt(int64(*v), 10)), nil
	case int64:
		return (strconv.FormatInt(v, 10)), nil
	case *int64:
		return (strconv.FormatInt(*v, 10)), nil
	case uint:
		return (strconv.FormatUint(uint64(v), 10)), nil
	case *uint:
		return (strconv.FormatUint(uint64(*v), 10)), nil
	case uint8:
		return (strconv.FormatUint(uint64(v), 10)), nil
	case *uint8:
		return (strconv.FormatUint(uint64(*v), 10)), nil
	case uint16:
		return (strconv.FormatUint(uint64(v), 10)), nil
	case *uint16:
		return (strconv.FormatUint(uint64(*v), 10)), nil
	case uint32:
		return (strconv.FormatUint(uint64(v), 10)), nil
	case *uint32:
		return (strconv.FormatUint(uint64(*v), 10)), nil
	case uint64:
		return (strconv.FormatUint(v, 10)), nil
	case *uint64:
		return (strconv.FormatUint(*v, 10)), nil
	case float32:
		return (strconv.FormatFloat(float64(v), 'f', -1, 32)), nil
	case *float32:
		return (strconv.FormatFloat(float64(*v), 'f', -1, 32)), nil
	case float64:
		if math.IsInf(v, 1) {
			return ("Infinity"), nil
		} else if math.IsInf(v, -1) {
			return ("-Infinity"), nil
		} else {
			return (strconv.FormatFloat(v, 'f', -1, 64)), nil
		}
	case *float64:
		return (strconv.FormatFloat(*v, 'f', -1, 64)), nil
	case big.Int:
		return (v.String()), nil
	case *big.Int:
		return (v.String()), nil
	case string, *string:
		{
			toWrite := escapeHTML(v)
			return toWrite, nil
		}
	default:
		{
			switch reflect.TypeOf(v).Kind() {
			case reflect.Func:
				{
					return "#invalid_func", errors.New("function is not allowed here")
				}
			}
			toWrite := escapeHTML(v)
			return toWrite, nil
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
	case attributeName[0] == 'a' &&
		attributeName[1] == 'r' &&
		attributeName[2] == 'i' &&
		attributeName[3] == 'a' &&
		attributeName[4] == '-':
		return true
	}
	return false
}
