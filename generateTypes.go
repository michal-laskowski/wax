package wax

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

func GenerateDefinitionFile(out string, namespace string, o ...any) error {
	file, err := os.Create(out)
	if err != nil {
		return err
	}
	defer file.Close()
	generator := definitionGenerator{
		out:       file,
		namespace: namespace,
		pkg:       reflect.TypeOf(o[0]).PkgPath(),
	}
	return generator.Generate(o...)
}

type definitionGenerator struct {
	out       io.StringWriter
	indent    int
	namespace string
	pkg       string
}

func (this *definitionGenerator) Generate(o ...any) error {
	if this.namespace != "" {
		this.outLine("declare namespace " + this.namespace + " {")
		this.doIndent()
	}
	if err := this.writeDefinition(o...); err != nil {
		return err
	}

	if this.namespace != "" {
		this.outLine("}")
		this.doDeIndent()
	}
	return nil
}

func (this *definitionGenerator) outLine(v string) {
	this.out.WriteString(strings.Repeat("  ", this.indent))
	this.out.WriteString(v)
	this.out.WriteString("\n")
}

func (this *definitionGenerator) doIndent() {
	this.indent++
}

func (this *definitionGenerator) doDeIndent() {
	this.indent--
}
func (this *definitionGenerator) outNext(v string) {
	this.out.WriteString(v)
}

func (this *definitionGenerator) outEndLine() {
	this.out.WriteString("\n")
}

func (this *definitionGenerator) writeDefinition(o ...any) error {

	otherTypes := []reflect.Type{}
	for _, obj := range o {
		t := reflect.TypeOf(obj)
		otherTypes = append(otherTypes, t)
		useTypes := this.writeType(t)

		idx := 0
		for idx < len(useTypes) {
			ot := useTypes[idx]
			idx++

			if ot.Kind() != reflect.Struct && ot.Name() == ot.Kind().String() {
				continue
			}
			if ot.Kind() == reflect.Interface {
				continue
			}
			if slices.Contains(otherTypes, ot) {
				continue
			}
			if strings.HasPrefix(ot.PkgPath(), this.pkg) == false {
				if ot.Kind() == reflect.Struct {
					continue
				}
			}

			otherTypes = append(otherTypes, ot)
			useTypes = append(useTypes, this.writeType(ot)...)
		}
	}

	return nil
}

func examiner(t reflect.Type) {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("type %s = {\n", t.Name()))

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		sb.WriteString(fmt.Sprintf("%s: %s\n", f.Name, f.Type.Name()))
	}
	sb.WriteString(fmt.Sprintf("}\n"))
	fmt.Println(sb.String())
}

type genericInfo struct {
	IsGeneric bool
	BaseType  string
	Param     string
}

var r = regexp.MustCompile("(([A-Za-z0-9_]*\\.)+([A-Za-z0-9_]+))(?:\\[(.*\\.(.*))\\])?")

func checkIfGeneric(t reflect.Type) genericInfo {
	tname := t.String()

	pkg := t.PkgPath()
	match := r.FindStringSubmatch(tname)

	if len(match) < 4 {
		panic(fmt.Sprintf("co to %s - %s", pkg, tname))
	}
	if match[4] == "" {
		return genericInfo{
			IsGeneric: false,
			BaseType:  match[3],
		}
	}
	return genericInfo{
		IsGeneric: true,
		BaseType:  match[3],
		Param:     match[5],
	}
}

func (this *definitionGenerator) writeType(t reflect.Type) []reflect.Type {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	tInfo := checkIfGeneric(t)

	typeName := tInfo.BaseType

	if tInfo.IsGeneric {
		this.outLine(fmt.Sprintf("type %s<T> = {", typeName))
	} else if isBaseType(t) {
		this.outLine(fmt.Sprintf("type %s =  object & { /* BASE_TYPE */", typeName))
	} else {
		this.outLine(fmt.Sprintf("type %s = {", typeName))
	}
	this.doIndent()
	usedTypes := []reflect.Type{}
	andAlso := []reflect.Type{}
	{
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				fieldInfo := t.Field(i)
				if fieldInfo.IsExported() == false {
					continue
				}
				itemTypeStr := "T"

				if !tInfo.IsGeneric && slices.Index([]reflect.Kind{reflect.Array, reflect.Chan, reflect.Map, reflect.Pointer, reflect.Slice}, fieldInfo.Type.Kind()) != -1 {
					itemTypeStr = fieldInfo.Type.Elem().Name()
				}

				switch fieldInfo.Type.Kind() {
				case reflect.Slice:
					{
						this.outLine(fmt.Sprintf("%s: %s[]", fieldInfo.Name, itemTypeStr))
						usedTypes = append(usedTypes, fieldInfo.Type.Elem())
					}
				case reflect.Map:
					{
						this.outLine(fmt.Sprintf("%s: Record<string, any>", fieldInfo.Name))
					}
				case reflect.Pointer:
					{
						this.outLine(fmt.Sprintf("%s?: %s | null | undefined", fieldInfo.Name, itemTypeStr))
						usedTypes = append(usedTypes, fieldInfo.Type.Elem())
					}
				case reflect.Struct:
					if fieldInfo.Anonymous {
						andAlso = append(andAlso, fieldInfo.Type)
					}

					this.outLine(fmt.Sprintf("%s: %s", fieldInfo.Name, this.getTypeNameForScript(fieldInfo.Type)))
					usedTypes = append(usedTypes, fieldInfo.Type)
				default:
					if fieldInfo.Anonymous {
						andAlso = append(andAlso, fieldInfo.Type)
						usedTypes = append(usedTypes, fieldInfo.Type)
					} else {
						this.outLine(fmt.Sprintf("%s: %s", fieldInfo.Name, this.getTypeNameForScript(fieldInfo.Type)))
						usedTypes = append(usedTypes, fieldInfo.Type)
					}
				}
			}
		}

		ptrType := reflect.PointerTo(t)
		for i := 0; i < ptrType.NumMethod(); i++ {
			methodInfo := ptrType.Method(i)

			numParams := methodInfo.Type.NumIn()
			numResults := methodInfo.Type.NumOut()
			if numResults > 1 {
				this.outLine(fmt.Sprintf("// multiple results %s", methodInfo.Name))
				continue
			}

			prmsStr := []string{}

			if numParams > 1 {

				for pI := 0; pI < numParams; pI++ {
					prmType := methodInfo.Type.In(pI)
					if pI == 0 && prmType.Kind() == reflect.Ptr && prmType.Elem() == t {
						//pointer to 'this'
					} else {
						usedTypes = append(usedTypes, prmType)

						prmsStr = append(prmsStr, fmt.Sprintf("p%d : %s", pI, this.getTypeNameForScript(prmType)))
						usedTypes = append(usedTypes, prmType)
					}
				}
			}

			if numResults == 0 {
				this.outLine(fmt.Sprintf("%s(%s) : void", methodInfo.Name, strings.Join(prmsStr, ", ")))

			} else {

				resultType := methodInfo.Type.Out(0)
				resultStr := []string{}
				switch resultType.Kind() {
				case reflect.Pointer:
					{
						resultStr = append(resultStr, this.getTypeNameForScript(resultType))
						usedTypes = append(usedTypes, resultType.Elem())
					}
				case reflect.Struct:
					{
						resultStr = append(resultStr, this.getTypeNameForScript(resultType))
						usedTypes = append(usedTypes, resultType)
					}
				case reflect.Slice:
					{
						resultStr = append(resultStr, this.getTypeNameForScript(resultType))
						usedTypes = append(usedTypes, resultType.Elem())
					}
				case reflect.Map:
					{
						resultStr = append(resultStr, this.getTypeNameForScript(resultType))
					}
				default:
					resultStr = append(resultStr, this.getTypeNameForScript(resultType))
					usedTypes = append(usedTypes, resultType)
				}

				if len(resultStr) > 1 {
					panic("not supported")
				}
				this.outLine(fmt.Sprintf("%s(%s) : %s", methodInfo.Name, strings.Join(prmsStr, ", "), strings.Join(resultStr, "  ")))
			}
		}
	}
	this.doDeIndent()

	this.outLine("}")

	for _, ao := range andAlso {
		oTI := checkIfGeneric(ao)
		if oTI.IsGeneric {
			this.outNext(" & " + oTI.BaseType + "<" + oTI.Param + ">")
		} else {
			this.outNext(" & " + ao.Name())
		}
	}
	this.outEndLine()
	return usedTypes
}

func (this *definitionGenerator) getTypeNameForScript(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Pointer:
		return t.Elem().Name()
	case reflect.Slice:
		return t.Elem().Name() + "[]"
	case reflect.Interface:
		return "unknown"
	case reflect.Map:
		return "Record<string,any>"
	case reflect.Struct:
		isSubGeneric := checkIfGeneric(t)
		if isSubGeneric.IsGeneric {
			return fmt.Sprintf("%s<%s>", isSubGeneric.BaseType, isSubGeneric.Param)
		} else {
			if strings.HasPrefix(t.PkgPath(), this.pkg) == false {
				return "unknown"
			}

			return fmt.Sprintf("%s", t.Name())
		}
	default:
		return t.Name()
	}
}

func isBaseType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return true
	}
	return false
}
