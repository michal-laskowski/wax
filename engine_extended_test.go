package wax_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/michal-laskowski/wax"
)

func Test_Engine_ExtendedOut(t *testing.T) {
	testDataFS := os.DirFS("./testdata/")

	engine := wax.New(wax.NewFsViewResolver(testDataFS))
	buf := bytes.NewBufferString("")

	toCheck, _ := fs.Glob(testDataFS, "*.expected")

	testModel := DummyTest{
		SomeStringArr:    []string{"str1", "str2"},
		SomeStringPtrArr: []*string{},
		PtrArr:           &[]string{"str1", "str2"},
		PtrArrPtr:        &[]*string{},
		AliasToString:    StringAlias("some-str-value-in-alias"),
		Ballance:         3.4e+38,
		Deposit:          1.7e+308,
		Other: &DummyTest{
			AliasToString: StringAlias("other-some-str-value"),
		},
		OtherDummySimple: DummySimple{
			DummySimpleField: "OtherDummySimple-str-value",
		},
		OtherDummyMaps: DummyMaps{},
		OtherDummySimpleGeneric: DummySimpleGeneric[int]{
			GenericField: 888888,
		},
		OtherDummyBasicTypes: DummyBasicTypes{
			P_string: "OtherDummyBasicTypes-str",
		},
		DummySimple: DummySimple{
			DummySimpleField: "DummySimple-Embedded-str-value",
		},
		goPrivateField: 999,

		SomeMap: map[string]any{"items": []any{"one", false, "two", "three"}},
	}
	for _, ef := range toCheck {
		buf.Reset()
		testName := strings.TrimSuffix(ef, ".expected")

		expected, err := fs.ReadFile(testDataFS, ef)
		err = engine.Render(buf, testName, testModel)

		if err != nil {
			var waxError wax.Error
			if errors.As(err, &waxError) == false {
				t.Errorf("wax must always return WaxError")
			}
			t.Errorf("\n-----------------------\n !!!!!!!!!!!!!! Errored %s\n%+v", testName, err)
		} else {
			actual := buf.String()
			compareHTML(t, testName, string(expected), actual)
		}
	}
}
