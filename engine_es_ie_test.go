package wax_test

import (
	"fmt"
	"testing"
)

func Test_Engine_esimport(t *testing.T) {
	meticulous_import_export := []TestSample{
		{
			source: `
                import defaultExport from "./module1.jsx"; 
                export function View() { return defaultExport({ title: "defaultExport" }) }`,
			expected: "<i>defaultExport</i>",
		},
		{
			source: `
		        import DefaultExport from "./module1.jsx";
		        export function View() { return <DefaultExport title="DefaultExport"/> }`,
			expected: "<i>DefaultExport</i>",
		},
		{
			source: `
		        import * as name from "./module1.jsx";
		        export function View() { return name.SimpleDiv({ title: "* as name" }) }`,
			expected: "<i>* as name</i>",
		},
		{
			source: `
		        import { SimpleDiv } from "./module1.jsx";
		        export function View() { return <SimpleDiv title="SimpleDiv"/> }`,
			expected: "<i>SimpleDiv</i>",
		},
		{
			source: `
		        import { SimpleDiv as SimpleDivAlias } from "./module1.jsx";
		        export function View() { return <SimpleDivAlias title="SimpleDivAlias"/> }`,
			expected: "<i>SimpleDivAlias</i>",
		},
		{
			source: `
		        import { SimpleDiv as simpleDivAlias } from "./module1.jsx";
		        export function View() { return simpleDivAlias({ title: "simpleDivAlias" }) }`,
			expected: "<i>simpleDivAlias</i>",
		},
		{
			source: `
		        import { default  as DefaultAlias } from "./module1.jsx";
		        export function View() { return <DefaultAlias title="DefaultAlias"/> }`,
			expected: "<i>DefaultAlias</i>",
		},
		{
			source: `
		        import DefaultExport, { default  as DefaultAlias } from "./module1.jsx";
		        export function View() { return <><DefaultExport title="DefaultExport"/><DefaultAlias title="DefaultAlias"/></> }`,
			expected: "<i>DefaultExport</i><i>DefaultAlias</i>",
		},
		{
			source: `
		        import defaultExport, * as name from "./module1.jsx";
		        export function View() { return <>{defaultExport({ title: "defaultExport" })}{name.SimpleDiv({ title: "* as name" })}</> }`,
			expected: "<i>defaultExport</i><i>* as name</i>",
		},
		{
			source: `
		        export function View() { return <SimpleDiv title="SimpleDiv-Hoist"/> };
		        import { SimpleDiv } from "./module1.jsx";`,
			expected: "<i>SimpleDiv-Hoist</i>",
		},
		{
			source: `
		        import { SimpleDiv as SimpleDivAlias, SimpleDiv  } from "./module_reexp";
                import * as imp from "./module_reexp" 
		        export function View() { return <>
                    <i1>{typeof SimpleDivAlias}</i1>
                    <i2>{typeof SimpleDiv}</i2>
                    <i3>{typeof imp.SimpleDiv}</i3>
                    <i4>{Object.keys(imp)}</i4>
                    <i5>{typeof imp["SimpleDiv"]}</i5>
                </> }`,
			expected: ` <i1>function</i1>
                        <i2>function</i2>
                        <i3>function</i3>
                        <i4>SimpleDiv</i4>
                        <i5>function</i5>`,
		},
	}

	modules := map[string]string{
		"module1.jsx": `
                    export default function SimpleDiv(p) { 
                        return <i>{p.title}</i>
                    }
                        
                    export function helper() {
                        return "helper-function-result"
                    }`,
		"module_reexp.tsx": `
            export {SimpleDiv} from "./module1"
        `,
	}
	for i, sample := range meticulous_import_export {
		name := fmt.Sprintf("es_module_meticulous_%d", i)
		if sample.name != "" {
			name = sample.name
		}
		t.Run(name, func(t *testing.T) {
			sample.modules = modules
			runSample(t, sample)
		})
	}
}
