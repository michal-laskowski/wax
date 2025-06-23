package gots

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
)

// Generates typescript typings (.d.ts) for given types.
//
// Can wrap types in given namespace (if empty will omit namespace).
//
// If pkg is specified it will generate types in packages containing given prefix.
func GenerateTypeDefinition(out io.StringWriter, namespace string, pkg string, typesToGenerate ...any) error {
	generator := definitionGenerator{
		out:       out,
		namespace: namespace,
		pkg:       pkg,
	}
	return generator.Generate(typesToGenerate...)
}

type definitionGenerator struct {
	out       io.StringWriter
	indent    int
	namespace string
	pkg       string
}

func (g *definitionGenerator) Generate(o ...any) error {
	if g.namespace != "" {
		g.outLine("declare namespace " + g.namespace + " {")
		g.doIndent()
	}
	if err := g.writeDefinition(o...); err != nil {
		return err
	}

	if g.namespace != "" {
		g.outLine("}")
		g.doDeIndent()
	}
	return nil
}

func (g *definitionGenerator) outLine(v string) {
	g.out.WriteString(strings.Repeat("  ", g.indent))
	g.out.WriteString(v)
	g.out.WriteString("\n")
}

func (g *definitionGenerator) doIndent() {
	g.indent++
}

func (g *definitionGenerator) doDeIndent() {
	g.indent--
}

func (g *definitionGenerator) outNext(v string) {
	g.out.WriteString(v)
}

func (g *definitionGenerator) outEndLine() {
	g.out.WriteString("\n")
}

func (g *definitionGenerator) shouldWriteType(t reflect.Type, i exTypeInfo) bool {
	if i.IsBasicType {
		return false
	}
	if !strings.HasPrefix(t.PkgPath(), g.pkg) {
		return false
	}

	if t.Name() == "" {
		return false
	}

	return true
}

func (g *definitionGenerator) writeDefinition(o ...any) error {
	typesToProcess := []reflect.Type{}
	processedTypes := map[string]exTypeInfo{}
	for _, obj := range o {
		t := reflect.TypeOf(obj)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		typeInfo := getTypeInfo(t)

		if _, ok := processedTypes[typeInfo.FullBaseTypeName]; ok {
			continue
		}

		processedTypes[typeInfo.FullBaseTypeName] = typeInfo
		useTypes := g.writeType(t, typeInfo)
		g.outEndLine()
		typesToProcess = append(typesToProcess, useTypes...)
	}

	for len(typesToProcess) > 0 {
		t := typesToProcess[0]
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		typeInfo := getTypeInfo(t)

		typesToProcess = typesToProcess[1:]
		if _, ok := processedTypes[typeInfo.FullBaseTypeName]; ok {
			continue
		}

		processedTypes[typeInfo.FullBaseTypeName] = typeInfo
		if !g.shouldWriteType(t, typeInfo) {
			continue
		}
		useTypes := g.writeType(t, typeInfo)
		g.outEndLine()
		typesToProcess = append(typesToProcess, useTypes...)
	}
	return nil
}

type exTypeInfo struct {
	IsGenericType        bool
	IsBasicType          bool
	IsAlias              bool
	IsInterface          bool
	UnderlyingSystemType reflect.Kind
	BaseType             string
	FullBaseTypeName     string
	TypeParam            string
}

var r = regexp.MustCompile(`(([A-Za-z0-9_]*\.)?([A-Za-z0-9_]+))(?:\[((.*\.)?(.*))\])?`)

func getTypeInfo(t reflect.Type) exTypeInfo {
	tname := t.String()
	match := r.FindStringSubmatch(tname)

	if len(match) < 4 || match[4] == "" {
		if getUnderlyingType(t).Kind() == reflect.Interface {
			i := getUnderlyingType(t)
			return exTypeInfo{
				IsGenericType: false,
				IsBasicType:   isBaseType(t) && !isAliasToBaseType(t),
				IsAlias:       isAliasToBaseType(t),
				IsInterface:   true,

				UnderlyingSystemType: i.Kind(),
				BaseType:             i.Name(),
				FullBaseTypeName:     i.String(),
			}
		} else {
			return exTypeInfo{
				IsGenericType: false,
				IsBasicType:   isBaseType(t) && !isAliasToBaseType(t),
				IsAlias:       isAliasToBaseType(t),

				UnderlyingSystemType: t.Kind(),
				BaseType:             match[3],
				FullBaseTypeName:     tname,
			}
		}
	}

	return exTypeInfo{
		IsGenericType:        true,
		UnderlyingSystemType: t.Kind(),
		BaseType:             match[3],
		TypeParam:            match[6],
		FullBaseTypeName:     match[1],
	}
}

func (g *definitionGenerator) writeType(t reflect.Type, tInfo exTypeInfo) []reflect.Type {
	typeName := tInfo.BaseType

	if tInfo.IsGenericType {
		g.outLine(fmt.Sprintf("type %s<T> = {", typeName))
	} else if isBaseType(t) {
		// type alias is object in goja
		g.outLine(fmt.Sprintf("type %s = %s & { ", typeName, "object"))
	} else {
		g.outLine(fmt.Sprintf("type %s = {", typeName))
	}
	g.doIndent()
	result := g.writeMembers(t, tInfo)

	g.doDeIndent()

	g.outNext("}")

	for _, ao := range result.andAlso {
		oTI := getTypeInfo(ao)
		if oTI.IsGenericType {
			g.outNext(" & " + oTI.BaseType + "<" + oTI.TypeParam + ">")
		} else {
			g.outNext(" & " + ao.Name())
		}
	}
	g.outEndLine()
	return result.usedTypes
}

func (g *definitionGenerator) writeMembers(t reflect.Type, tInfo exTypeInfo) struct {
	andAlso   []reflect.Type
	usedTypes []reflect.Type
} {
	usedTypes := []reflect.Type{}
	andAlso := []reflect.Type{}
	{
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				dumpMemberType := true
				fieldInfo := t.Field(i)
				if !fieldInfo.IsExported() {
					continue
				}

				ft := getUnderlyingType(fieldInfo.Type)
				if fieldInfo.Anonymous {
					andAlso = append(andAlso, ft)
				} else {
					if tInfo.IsGenericType {
						g.outLine(fmt.Sprintf("%s: %s", fieldInfo.Name, g.getTypingNameForGeneric(fieldInfo)))
					} else if ft.Kind() == reflect.Struct && ft.Name() == "" {

						if fieldInfo.Type.Kind() == reflect.Pointer {
							g.outLine(fmt.Sprintf("%s: null | {", fieldInfo.Name))
						} else {
							g.outLine(fmt.Sprintf("%s: {", fieldInfo.Name))
						}
						g.doIndent()
						memberInfo := getTypeInfo(ft)
						membersResult := g.writeMembers(ft, memberInfo)
						usedTypes = append(usedTypes, membersResult.usedTypes...)
						g.doDeIndent()
						g.outLine("}")
						dumpMemberType = false
					} else {
						g.outLine(fmt.Sprintf("%s: %s", fieldInfo.Name, g.getTypingName(fieldInfo.Type)))
					}
				}

				if dumpMemberType {
					switch ft.Kind() {
					case reflect.Slice:
						usedTypes = append(usedTypes, getUnderlyingType(ft.Elem()))
					case reflect.Map:
						usedTypes = append(usedTypes, getUnderlyingType(ft.Key()), getUnderlyingType(ft.Elem()))
					default:
						usedTypes = append(usedTypes, getUnderlyingType(ft))
					}
				}

			}
		}

		if t.Kind() == reflect.Interface {
			mr := g.writeMethods(t)
			andAlso = append(andAlso, mr.andAlso...)
			usedTypes = append(usedTypes, mr.usedTypes...)
		} else {
			ptrType := reflect.PointerTo(t)
			mr := g.writeMethods(ptrType)
			andAlso = append(andAlso, mr.andAlso...)
			usedTypes = append(usedTypes, mr.usedTypes...)
		}
	}

	return struct {
		andAlso   []reflect.Type
		usedTypes []reflect.Type
	}{
		andAlso,
		usedTypes,
	}
}

func (g *definitionGenerator) writeMethods(ptrType reflect.Type) struct {
	andAlso   []reflect.Type
	usedTypes []reflect.Type
} {
	usedTypes := []reflect.Type{}
	andAlso := []reflect.Type{}

	isInterface := ptrType.Kind() == reflect.Interface
	for i := 0; i < ptrType.NumMethod(); i++ {
		methodInfo := ptrType.Method(i)

		numParams := methodInfo.Type.NumIn()
		numResults := methodInfo.Type.NumOut()
		if numResults > 1 {
			// TODO configure to panic
			g.outLine(fmt.Sprintf("// multiple results %s", methodInfo.Name))
			continue
		}

		paramsStr := []string{}

		if isInterface {
			for pI := 0; pI < numParams; pI++ {
				prmType := methodInfo.Type.In(pI)
				usedTypes = append(usedTypes, prmType)
				paramsStr = append(paramsStr, fmt.Sprintf("p%d: %s", pI+1, g.getTypingName(prmType)))
				usedTypes = append(usedTypes, prmType)
			}
		} else {
			for pI := 0; pI < numParams; pI++ {
				prmType := methodInfo.Type.In(pI)
				if pI == 0 {
				} else {
					usedTypes = append(usedTypes, prmType)

					paramsStr = append(paramsStr, fmt.Sprintf("p%d: %s", pI, g.getTypingName(prmType)))
					usedTypes = append(usedTypes, prmType)
				}
			}
		}

		if numResults == 0 {
			g.outLine(fmt.Sprintf("%s(%s): void", methodInfo.Name, strings.Join(paramsStr, ", ")))
		} else {
			resultType := methodInfo.Type.Out(0)
			resultStr := []string{}
			switch resultType.Kind() {
			case reflect.Pointer:
				{
					resultStr = append(resultStr, g.getTypingName(resultType))
					usedTypes = append(usedTypes, resultType.Elem())
				}
			case reflect.Struct:
				{
					resultStr = append(resultStr, g.getTypingName(resultType))
					usedTypes = append(usedTypes, resultType)
				}
			case reflect.Slice:
				{
					resultStr = append(resultStr, g.getTypingName(resultType))
					usedTypes = append(usedTypes, resultType.Elem())
				}
			case reflect.Map:
				{
					resultStr = append(resultStr, g.getTypingName(resultType))
				}
			default:
				resultStr = append(resultStr, g.getTypingName(resultType))
				usedTypes = append(usedTypes, resultType)
			}

			if len(resultStr) > 1 {
				panic("not supported")
			}
			g.outLine(fmt.Sprintf("%s(%s): %s", methodInfo.Name, strings.Join(paramsStr, ", "), strings.Join(resultStr, "  ")))
		}
	}
	return struct {
		andAlso   []reflect.Type
		usedTypes []reflect.Type
	}{
		andAlso,
		usedTypes,
	}
}

func (g *definitionGenerator) getTypingName(t reflect.Type) string {
	tInfo := getTypeInfo(getUnderlyingType(t))
	isAlias := tInfo.IsAlias

	typeName := t.Name()
	if !isAlias {
		switch t.Kind() {
		case reflect.String,
			reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.Uintptr, reflect.Complex64, reflect.Complex128:
			typeName = g.getTypingNameForBase(t.Kind())

		case reflect.Pointer:
			typeName = fmt.Sprintf("null | %s", g.getTypingName(t.Elem()))
		case reflect.Slice:
			if t.Elem().Kind() == reflect.Pointer {
				typeName = fmt.Sprintf("(%s)[]", g.getTypingName(t.Elem()))
			} else {
				typeName = g.getTypingName(t.Elem()) + "[]"
			}
		case reflect.Interface:
			if tInfo.FullBaseTypeName == "interface {}" {
				typeName = "any"
			} else {
				typeName = fmt.Sprintf("null | %s", typeName)
			}
		case reflect.Map:
			keyType := g.getTypingName(t.Key())
			elemType := g.getTypingName(t.Elem())
			typeName = fmt.Sprintf("Record<%s, %s>", keyType, elemType)
		case reflect.Struct:
			isSubGeneric := getTypeInfo(t)
			if isSubGeneric.IsGenericType {
				typeName = fmt.Sprintf("%s<%s>", isSubGeneric.BaseType, isSubGeneric.TypeParam)
			} else if !g.shouldWriteType(t, tInfo) {
				typeName = "unknown"
			} else {
				typeName = t.Name()
			}
		default:
			typeName = t.Name()
		}
	}
	return typeName
}

func (g *definitionGenerator) getTypingNameForBase(k reflect.Kind) string {
	typeName := "unknown"

	switch k {
	case reflect.String:
		typeName = "string"
	case reflect.Bool:
		typeName = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		typeName = "number"
	case reflect.Uintptr, reflect.Complex64, reflect.Complex128:
		typeName = "object"
	}
	return typeName
}

func (g *definitionGenerator) getTypingNameForGeneric(fieldInfo reflect.StructField) string {
	_, isGenericParam := fieldInfo.Tag.Lookup("waxGeneric")

	if !isGenericParam {
		return g.getTypingName(fieldInfo.Type)
	}
	return getTypeForKind("T", fieldInfo.Type.Kind())
}

func getTypeForKind(t string, kind reflect.Kind) string {
	switch kind {
	case reflect.Pointer:
		return "null | " + t
	case reflect.Slice:
		return "" + t + "[]"
	default:
		return t
	}
}

func getUnderlyingType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Pointer:
		return t.Elem()
	default:
		return t
	}
}

func isAliasToBaseType(t reflect.Type) bool {
	return t.PkgPath() != "" && isBaseType(t)
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
