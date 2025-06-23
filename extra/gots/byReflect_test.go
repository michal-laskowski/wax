package gots_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/andreyvit/diff"

	"github.com/michal-laskowski/wax/extra/gots"
)

type TestStruct struct {
	SomeString             string
	SomeStringPtr          *string
	SomeStringArr          []string
	SomeStringPtrArr       []*string
	ArrPtr                 *[]string
	AliasToString          StringAlias
	GenericDataString      TestGeneric[string]
	GenericDataInt         TestGeneric[int]
	GenericDataStringAlias TestGeneric[StringAlias]
	GenericDataContact     TestGeneric[Contact]
	ArrayWithGeneric       []TestGeneric[Contact]
}

type Contact struct {
	Contact string
	Email   string
}

type (
	StringAlias        string
	TestGeneric[T any] struct {
		Data []T `waxGeneric:""`
		P1   string
		P2   bool
		P3   T `waxGeneric:""`
	}
)

type DummyTest struct {
	SomeStringArr    []string
	SomeStringPtrArr []*string
	PtrArr           *[]string
	PtrArrPtr        *[]*string
	Ballance         float32
	Deposit          float64
	Other            *DummyTest
	DummySimple
	OtherSimple   DummySimple
	GenericSimple DummySimpleGeneric[int]
}

type DummySimple struct {
	DummySimpleField string
}

type DummySimpleGeneric[T any] struct {
	GenericField T `waxGeneric:""`
}

type DummyMaps struct {
	Map1 map[string]int
	Map2 map[string]DummySimple
	Map3 map[DummySimple]int
	Map4 map[string]DummySimpleGeneric[DummySimple]
	Map5 map[string]any
	Map6 map[any]int
}

func (t *StringAlias) SomeAliasTrueMethod() bool {
	return true
}

func (t StringAlias) SomeAliasFalseMethod() bool {
	return false
}

type Dummy_BasicTypes struct {
	P_bool       bool
	P_string     string
	P_int        int
	P_int8       int8
	P_int16      int16
	P_int32      int32
	P_int64      int64
	P_uint       uint
	P_uint8      uint8
	P_uint16     uint16
	P_uint32     uint32
	P_uint64     uint64
	P_uintptr    uintptr
	P_byte       byte // alias for uint8
	P_rune       rune // alias for int32
	P_float32    float32
	P_float64    float64
	P_complex64  complex64
	P_complex128 complex128
}

type DummyWithNestedStruct struct {
	StringProp string
	Proxy      struct {
		Address string
		Port    uint32
	}
	ProxyPtr *struct {
		Address string
		Port    uint32
	}
}

type DummyWithOtherPkgType struct {
	Time     time.Time
	TestingB testing.B
	Foo      any
}

type DummyFunc struct{}

func (t DummyFunc) ReturnsBool() bool {
	return false
}

func (t DummyFunc) ReturnsBoolParams(v int) bool {
	return false
}

type DummyInterface interface {
	Func1() string
	Func2(p int) string
	Func3(p DummyInterface) string
	Func4(p any) string
}

func thisPackageOnly() string {
	pp := reflect.TypeOf(TestStruct{})
	return pp.PkgPath()
}

func Test_BasicTypes(t *testing.T) {
	buf := bytes.NewBufferString("")

	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), Dummy_BasicTypes{})
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `        
type Dummy_BasicTypes = {
  P_bool: boolean
  P_string: string
  P_int: number
  P_int8: number
  P_int16: number
  P_int32: number
  P_int64: number
  P_uint: number
  P_uint8: number
  P_uint16: number
  P_uint32: number
  P_uint64: number
  P_uintptr: object
  P_byte: number
  P_rune: number
  P_float32: number
  P_float64: number
  P_complex64: object
  P_complex128: object
}
`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_base(t *testing.T) {
	buf := bytes.NewBufferString("")

	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), Contact{}, TestStruct{})
	expected := `
type Contact = {
  Contact: string
  Email: string
}

type TestStruct = {
  SomeString: string
  SomeStringPtr: null | string
  SomeStringArr: string[]
  SomeStringPtrArr: (null | string)[]
  ArrPtr: null | string[]
  AliasToString: StringAlias
  GenericDataString: TestGeneric<string>
  GenericDataInt: TestGeneric<int>
  GenericDataStringAlias: TestGeneric<StringAlias>
  GenericDataContact: TestGeneric<Contact>
  ArrayWithGeneric: TestGeneric<Contact>[]
}

type StringAlias = object & { 
  SomeAliasFalseMethod(): boolean
  SomeAliasTrueMethod(): boolean
}

type TestGeneric<T> = {
  Data: T[]
  P1: string
  P2: boolean
  P3: T
}

`
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_Maps(t *testing.T) {
	buf := bytes.NewBufferString("")
	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), DummyMaps{})
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `
type DummyMaps = {
  Map1: Record<string, number>
  Map2: Record<string, DummySimple>
  Map3: Record<DummySimple, number>
  Map4: Record<string, DummySimpleGeneric<DummySimple>>
  Map5: Record<string, any>
  Map6: Record<any, number>
}

type DummySimple = {
  DummySimpleField: string
}

type DummySimpleGeneric<T> = {
  GenericField: T
}

`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_Interface(t *testing.T) {
	buf := bytes.NewBufferString("")

	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), (*DummyInterface)(nil))
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `        
type DummyInterface = {
  Func1(): string
  Func2(p1: number): string
  Func3(p1: null | DummyInterface): string
  Func4(p1: any): string
}
`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_DummyWithNestedStruct(t *testing.T) {
	buf := bytes.NewBufferString("")

	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), DummyWithNestedStruct{})
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `        
type DummyWithNestedStruct = {
  StringProp: string
  Proxy: {
    Address: string
    Port: number
  }
  ProxyPtr: null | {
    Address: string
    Port: number
  }
}
`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_DummyWithOtherPkgType(t *testing.T) {
	buf := bytes.NewBufferString("")

	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), DummyWithOtherPkgType{})
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `        
type DummyWithOtherPkgType = {
  Time: unknown
  TestingB: unknown
  Foo: any
}
`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_StructFunctions(t *testing.T) {
	buf := bytes.NewBufferString("")

	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), DummyFunc{})
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `        
type DummyFunc = {
  ReturnsBool(): boolean
  ReturnsBoolParams(p1: number): boolean
}
`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}

func Test_complex1(t *testing.T) {
	buf := bytes.NewBufferString("")
	err := gots.GenerateTypeDefinition(buf, "", thisPackageOnly(), DummyTest{})
	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored - %+v", err)
	}
	expected := `        
type DummyTest = {
  SomeStringArr: string[]
  SomeStringPtrArr: (null | string)[]
  PtrArr: null | string[]
  PtrArrPtr: null | (null | string)[]
  Ballance: number
  Deposit: number
  Other: null | DummyTest
  OtherSimple: DummySimple
  GenericSimple: DummySimpleGeneric<int>
} & DummySimple

type DummySimple = {
  DummySimpleField: string
}

type DummySimpleGeneric<T> = {
  GenericField: T
}
`
	actual := buf.String()
	if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
		t.Errorf("Result not as expected:\n%v", diff.LineDiff(e, a))
	}
}
