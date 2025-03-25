package wax_test

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/andreyvit/diff"

	"github.com/michal-laskowski/wax"
)

type TestSample struct {
	skip        string
	name        string
	description string
	source      string
	expected    string
	modules     map[string]string
	model       any

	errorPhase   string
	errorMessage string
}

type DummyTest struct {
	SomeStringArr           []string
	SomeStringPtrArr        []*string
	PtrArr                  *[]string
	PtrArrPtr               *[]*string
	AliasToString           StringAlias
	Ballance                float32
	Deposit                 float64
	Other                   *DummyTest
	OtherDummySimple        DummySimple
	OtherDummyMaps          DummyMaps
	OtherDummySimpleGeneric DummySimpleGeneric[int]
	OtherDummyBasicTypes    DummyBasicTypes
	DummySimple
	goPrivateField int
}

type Contact struct {
	Contact string
	Email   string

	System string
	Planet string
}

func (c Contact) FullAddress() string {
	return c.Planet + " at " + c.System
}

type StringAlias2 string

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
type DummyBasicTypes struct {
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
type StringAlias string

func (t *StringAlias) SomeAliasTrueMethod() bool {
	return true
}

func (t StringAlias) SomeAliasFalseMethod() bool {
	return false
}

type TestGeneric[T any] struct {
	Data []T `waxGeneric:""`
	P1   string
	P2   bool
	P3   T `waxGeneric:""`
}

func runSamples(t *testing.T, toRunSamples []TestSample) {
	for _, sample := range toRunSamples {
		t.Run(sample.name, func(t *testing.T) {
			runSample(t, sample)
		})
	}
}

func runSample(t *testing.T, sample TestSample) {
	fs := fstest.MapFS{}
	fs["View.jsx"] = &fstest.MapFile{Data: []byte(sample.source)}
	for k, v := range sample.modules {
		fs[k] = &fstest.MapFile{Data: []byte(v)}
	}
	engine := wax.New(wax.NewFsViewResolver(fs))

	buf := bytes.NewBufferString("")
	err := engine.Render(buf, "View", sample.model)

	if err != nil {
		t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored %s - %+v", sample.name, err)
	} else {
		actual := buf.String()
		compareHTML(t, sample.name, sample.expected, actual)
	}
}

func execSample(sample TestSample) (string, error) {
	fs := fstest.MapFS{}
	fs["View.jsx"] = &fstest.MapFile{Data: []byte(sample.source)}
	for k, v := range sample.modules {
		fs[k] = &fstest.MapFile{Data: []byte(v)}
	}
	engine := wax.New(wax.NewFsViewResolver(fs))

	buf := bytes.NewBufferString("")
	err := engine.Render(buf, "View", sample.model)
	return buf.String(), err
}

func compareHTML(t *testing.T, tName string, e string, a string) bool {
	eF := formatHTML(e)
	aF := formatHTML(a)

	if strings.Compare(eF, aF) != 0 {
		t.Errorf("Result not as expected:\n\n%v\n\n\n", diff.LineDiff(eF, aF))
		t.Errorf("\n-----------------------\nResult was incorrect %s,\n\n.....got :\n%s\n..\n\n.....want:\n%s\n..\n\n", tName, aF, eF)
		return false
	}
	return true
}
