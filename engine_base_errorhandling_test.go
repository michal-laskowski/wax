package wax_test

import (
	"errors"
	"testing"

	"github.com/michal-laskowski/wax"
)

func Test_Engine_source_error(t *testing.T) {
	check_error_reporting := []TestSample{
		{
			name:        "error_always_will_be_WaxError",
			description: "",
			source: `///
			            export function View(model) {
			                return <div>ok
			            `,
			errorPhase:   wax.PhaseLoading,
			errorMessage: "wax error [load]: '/View.jsx': error on lines 2:4",
		},
		{
			name:        "error_when_no_main_view_resolved",
			description: "",
			source: `///
			export function not_default_no_view(model) {
			    return "ok"
			}`,
			errorPhase:   wax.PhaseLoading,
			errorMessage: "wax error [load]: '/View.jsx': could not find function 'View'",
		},
		{
			name:        "error_es_module_import",
			description: "",
			source: `///
import * as otherModule from "./syntaxerrormodule.jsx"
export const foo = () => 1`,
			modules: map[string]string{
				"syntaxerrormodule.jsx": `
    export const missing_parentheses = () => {}
    ) /* <<<*/ `,
			},
			errorPhase:   wax.PhaseLoading,
			errorMessage: "wax error [load]: '/syntaxerrormodule.jsx': error on lines 3:3",
		},
		{
			name:        "error_on_jsx_attribute",
			description: "",
			source: `///
            function doThrow(){
                throw "some exception" 
            }
			export function View() {
			    return <div x-attr={doThrow()}></div>
			}`,
			errorPhase:   wax.PhaseExec,
			errorMessage: "wax error: execute: /View.jsx - some exception at doThrow (file:///View.jsx?ts=-dcbffeff2bc000:3:17(2)) - some exception at doThrow (file:///View.jsx?ts=-dcbffeff2bc000:3:17(2))",
		},
	}

	for _, sample := range check_error_reporting {
		t.Run(sample.name, func(t *testing.T) {
			runErrorReportingSample(t, sample)
		})
	}
}

func runErrorReportingSample(t *testing.T, sample TestSample) {
	actual, err := execSample(sample)

	if err == nil {
		t.Fatal("expected to get error")
	}
	var waxError wax.Error
	if errors.As(err, &waxError) == false {
		t.Errorf("wax must always return WaxError")
	}

	expectedPhase := sample.errorPhase
	expectedMessage := sample.errorMessage
	if waxError.Phase != expectedPhase {
		t.Errorf("invalid phase > \n\tgot      : %s\n\texpected : %s", waxError.Phase, expectedPhase)
	}

	if waxError.ErrorDetailed() != expectedMessage {
		t.Errorf("invalid message > \n\tgot      : %s\n\texpected : %s", waxError.ErrorDetailed(), expectedMessage)
	}

	 if actual != ""  && expectedPhase != wax.PhaseExec{
	 	t.Errorf("invalid output:" + actual)
	}
}
