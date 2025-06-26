
import * as AllImports from "./someclass"

import { SomeClass as SingleImport } from "./someclass.tsx"

export default function View() {
    const fromAll = new AllImports.SomeClass("from-all")
    const fromSingle = new SingleImport("SingleImport")
 
    return <>
        <case-1>{ fromAll.getValue() }</case-1>
        <case-2>{ fromSingle.getValue() }</case-2>
    </>
}
