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
                export default function Render() { return defaultExport({ title: "defaultExport" }) }`,
			expected: "<i>defaultExport</i>",
		},
		{
			source: `
		        import DefaultExport from "./module1.jsx";
		        export default function Render() { return <DefaultExport title="DefaultExport"/> }`,
			expected: "<i>DefaultExport</i>",
		},
		{
			source: `
		        import * as name from "./module1.jsx";
		        export default function Render() { return name.SimpleDiv({ title: "* as name" }) }`,
			expected: "<i>* as name</i>",
		},
		{
			source: `
		        import { SimpleDiv } from "./module1.jsx";
		        export default function Render() { return <SimpleDiv title="SimpleDiv"/> }`,
			expected: "<i>SimpleDiv</i>",
		},
		{
			source: `
		        import { SimpleDiv as SimpleDivAlias } from "./module1.jsx";
		        export default function Render() { return <SimpleDivAlias title="SimpleDivAlias"/> }`,
			expected: "<i>SimpleDivAlias</i>",
		},
		{
			source: `
		        import { SimpleDiv as simpleDivAlias } from "./module1.jsx";
		        export default function Render() { return simpleDivAlias({ title: "simpleDivAlias" }) }`,
			expected: "<i>simpleDivAlias</i>",
		},
		{
			source: `
		        import { default  as DefaultAlias } from "./module1.jsx";
		        export default function Render() { return <DefaultAlias title="DefaultAlias"/> }`,
			expected: "<i>DefaultAlias</i>",
		},
		{
			source: `
		        import DefaultExport, { default  as DefaultAlias } from "./module1.jsx";
		        export default function Render() { return <><DefaultExport title="DefaultExport"/><DefaultAlias title="DefaultAlias"/></> }`,
			expected: "<i>DefaultExport</i><i>DefaultAlias</i>",
		},
		{
			source: `
		        import defaultExport, * as name from "./module1.jsx";
		        export default function Render() { return <>{defaultExport({ title: "defaultExport" })}{name.SimpleDiv({ title: "* as name" })}</> }`,
			expected: "<i>defaultExport</i><i>* as name</i>",
		},
		{
			source: `
		        export default function Render() { return <SimpleDiv title="SimpleDiv-Hoist"/> };
		        import { SimpleDiv } from "./module1.jsx";`,
			expected: "<i>SimpleDiv-Hoist</i>",
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
