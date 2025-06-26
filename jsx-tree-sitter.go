package wax

import (
	"fmt"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

func NewTreeSitterTranspiler(options ...TreeSitterTranspilerOption) TypeScriptTranspiler {
	return &treeSitterTranspiler{
		options: options,
	}
}

func WithDebug() TreeSitterTranspilerOption {
	return func(e *treeSitterVisitor) {
		e.debug = true
	}
}

type (
	TreeSitterTranspilerOption func(*treeSitterVisitor)
	treeSitterTranspiler       struct {
		options []TreeSitterTranspilerOption
	}
)

// https://raw.githubusercontent.com/tree-sitter/tree-sitter-typescript/refs/heads/master/tsx/src/grammar.json
var language = sitter.NewLanguage(typescript.LanguageTSX())

func (t *treeSitterTranspiler) Transpile(fileName string, fileContent string) (string, error) {
	source := strings.NewReader(fileContent)

	parser := sitter.NewParser()
	parser.SetLanguage(language)
	defer parser.Close()
	var buf [4096]byte
	input := sitter.Input{
		Read: func(offset uint32, position sitter.Point) []byte {
			n, _ := source.ReadAt(buf[:], int64(offset))
			return buf[:n]
		},

		Encoding: sitter.InputEncodingUTF8,
	}
	tree := parser.ParseInput(nil, input)

	visitor := &treeSitterVisitor{}

	for _, option := range t.options {
		option(visitor)
	}
	return visitor.process(tree, fileName, fileContent)
}

type treeSitterVisitor struct {
	out   *strings.Builder
	last  uint32
	debug bool
}

func (t *treeSitterVisitor) process(tree *sitter.Tree, fileName string, fileContent string) (string, error) {
	rootNode := tree.RootNode()

	if rootNode.HasError() {
		if t.debug {
			printNode(tree.RootNode(), []byte(fileContent), 0)
		}

		err := findErrorNodes(rootNode, []byte(fileContent))
		if err != nil {
			return "", err
		}
		// skip errors
	}
	t.out = &strings.Builder{}
	t.out.Grow(len(fileContent) + 500)
	t.last = 0
	t.visit(rootNode, []byte(fileContent), 0)
	return t.out.String(), nil
}

func findErrorNodes(node *sitter.Node, code []byte) error {
	if node.Type() == "ERROR" {
		parent := node.Parent()

		if parent != nil && parent.Type() == "string" {
			if node.Content(code)[0] == '&' {
				// skip this
				// https://github.com/tree-sitter/tree-sitter-typescript/issues/320
			}
		} else {
			start := node.StartByte()
			end := node.EndByte()

			if parent == nil || (start == 0 && end == uint32(len(code))) {
				return fmt.Errorf("syntax error: unexpected end of input")
			}

			return fmt.Errorf("error on lines %d:%d", node.StartPoint().Row+1, node.EndPoint().Row+1)
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		if err := findErrorNodes(node.Child(i), code); err != nil {
			return err
		}
	}
	return nil
}

func (t *treeSitterVisitor) visit(node *sitter.Node, sourceCode []byte, depth int) {
	nodeType := node.Type()
	nodeEnd := node.EndByte()
	switch nodeType {
	case "jsx_self_closing_element", "jsx_element":
		{
			t.out.Write(sourceCode[t.last:node.StartByte()])
			t.last = node.StartByte()
			t.visitJSX(node, sourceCode, depth)
		}
	case "as_expression":
		{
			t.visit(node.Child(0), sourceCode, depth+1)
			t.replaceWithSpacesFormat(node.Child(1), sourceCode)
			t.replaceWithSpacesFormat(node.Child(2), sourceCode)
		}
	case
		"type_alias_declaration",
		"type_annotation",
		"type_arguments",
		"type_parameters",
		"declare",
		"accessibility_modifier",
		"ambient_declaration":
		{
			t.replaceWithSpacesFormat(node, sourceCode)
		}
	case "non_null_expression":
		{
			t.visit(node.Child(0), sourceCode, depth+1)
		}
	case "import_statement":
		{
			i := node.Content(sourceCode)
			data := parseImportClause(i)
			replaceResult := ""
			if data.ImportedDefaultBinding != "" {
				replaceResult += "const " + data.ImportedDefaultBinding + " = (module.do_import('" + data.ModuleName + "').default ?? (()=> {throw `no default export in '" + data.ModuleName + "'`}));"
			}
			if data.NameSpaceImport != "" {
				replaceResult += "const " + data.NameSpaceImport + " = module.do_import('" + data.ModuleName + "').exports;"
			}
			namedImports := ""
			for k, v := range data.NamedImports {
				if v == "default" {
					replaceResult += "const " + k + " = (module.do_import('" + data.ModuleName + "').default ?? (()=> {throw `no default export in '" + data.ModuleName + "'`}));"
				} else {
					namedImports += fmt.Sprintf("%s: %s, ", v, k)
				}
			}
			if namedImports != "" {
				namedImports = "const {" + namedImports + "} = module.do_import('" + data.ModuleName + "').exports;"
			}
			replaceResult += namedImports

			t.out.Write(sourceCode[t.last:node.StartByte()])
			t.out.WriteString(replaceResult)
		}
	case "export_statement":
		if node.ChildCount() > 1 {
			keyword := node.Child(0).Type()
			switch keyword {
			case "export":
				if node.Child(1).Type() == "default" {
					replaceResult := "module.default = "
					t.formatTo(node, sourceCode)
					t.out.WriteString(replaceResult)
					t.last = node.Child(2).StartByte()
					t.visitExport(node.Child(2), sourceCode, depth)
				} else {
					t.formatTo(node.Child(0), sourceCode)
					t.last = node.Child(1).StartByte()
					t.visitExport(node.Child(1), sourceCode, depth+1)
				}
			default:
				for i := 0; i < int(node.ChildCount()); i++ {
					t.visit(node.Child(i), sourceCode, depth+1)
				}
				toRewrite := sourceCode[t.last:nodeEnd]
				t.out.Write(toRewrite)
			}
		}
	case "meta_property":
		if node.Child(0).Type() == "import" {
			t.out.WriteString("module.meta")
		} else {
			t.out.WriteString(node.Content(sourceCode))
		}

	case "jsx_expression":
		expressionBody := node.Child(1)
		t.last = expressionBody.StartByte()
		t.visit(node.Child(1), sourceCode, depth)
	case "jsx_text":
		t.out.WriteString("`" + node.Content(sourceCode) + "`")
	default:
		cc := int(node.ChildCount())
		for i := 0; i < cc; i++ {
			t.visit(node.Child(i), sourceCode, depth+1)
		}
		toRewrite := sourceCode[t.last:nodeEnd]
		t.out.Write(toRewrite)
	}
	t.last = nodeEnd
}

func (t *treeSitterVisitor) visitExport(body *sitter.Node, sourceCode []byte, depth int) {
	replaceResult := ""
	var bodyExpr *sitter.Node
	switch body.Type() {
	case "lexical_declaration", "variable_declaration":
		// export const X = 10;  →  const X = module.exports.X = 10;
		keyword := body.Child(0).Content(sourceCode)       // "let" lub "const"
		name := body.Child(1).Child(0).Content(sourceCode) // Nazwa zmiennej
		bodyExpr = body.Child(1).Child(2)
		replaceResult = fmt.Sprintf("%s %s = module.exports.%s = ", keyword, name, name)

	case "function_declaration":
		// export function add(...) {} → module.exports.add = function add(...) {};
		name := body.Child(1).Content(sourceCode)
		bodyExpr = body
		replaceResult = fmt.Sprintf("module.exports.%s = %s;", name, name)

	case "class_declaration":
		t.out.Write(sourceCode[t.last:body.StartByte()])
		t.last = body.StartByte()
		t.visit(body, sourceCode, depth)

		name := body.Child(1).Content(sourceCode)
		t.out.WriteString(fmt.Sprintf("; /* WAX */ module.exports.%s = %s", name, name))
		t.last = body.EndByte()
		return
	case "identifier":
		// export const X = 10;  →  const X = module.exports.X = 10;
		name := body.Content(sourceCode)
		replaceResult = fmt.Sprintf("module.exports.%s = %s;", name, name)
	case "default":
		// export default coś → module.exports.default = coś;
		bodyExpr = body.Child(2)
		replaceResult = "module.default = "
		t.out.Write(sourceCode[t.last:body.StartByte()])
		t.out.WriteString(replaceResult)
		t.visitExport(bodyExpr, sourceCode, depth)
		return

	case "export_clause":
		// export { foo, bar } → module.exports.foo = foo; module.exports.bar = bar;
		exportList := body
		replacement := []string{}
		if exportList.ChildCount() > 1 {
			for i := 1; i < int(exportList.ChildCount())-1; i += 2 {
				exportName := exportList.Child(i).Content(sourceCode)
				replacement = append(replacement, fmt.Sprintf("module.exports.%s = %s;", exportName, exportName))
			}
		} else {
			exportName := exportList.Child(0).Content(sourceCode)
			replacement = append(replacement, fmt.Sprintf("module.exports.%s = %s;", exportName, exportName))
		}
		replaceResult = strings.Join(replacement, " ")

	case "type_alias_declaration", "interface_declaration":
		// export type X = ... → "      type X = ..."
		// export interface X { ... } → "         interface X { ... }"
		// replace with whitespaces
		t.replaceWithSpacesFormat(body, sourceCode)
		return
	}
	t.out.Write(sourceCode[t.last:body.StartByte()])
	t.out.WriteString(replaceResult)
	if bodyExpr != nil {
		t.last = bodyExpr.StartByte()
		t.visit(bodyExpr, sourceCode, depth)
	}
	t.last = body.EndByte()
}

func (t *treeSitterVisitor) formatTo(node *sitter.Node, sourceCode []byte) {
	t.out.Write(sourceCode[t.last:node.StartByte()])
	t.last = node.StartByte()
}

func (t *treeSitterVisitor) replaceWithSpacesFormat(node *sitter.Node, sourceCode []byte) {
	t.out.Write(sourceCode[t.last:node.StartByte()])
	t.last = node.StartByte()
	start, end := node.StartByte(), node.EndByte()
	space := strings.Repeat(" ", int(end-start))
	t.out.WriteString(space)
	t.last = node.EndByte()
}

func (t *treeSitterVisitor) visitJSX(node *sitter.Node, sourceCode []byte, depth int) {
	nodeType := node.Type()
	switch nodeType {
	case "jsx_self_closing_element":
		t.out.WriteString("wax.Sub(w => w")
		{
			identifier := node.ChildByFieldName("name").Content(sourceCode)
			isComponent := len(identifier) > 0 && unicode.IsUpper([]rune(identifier)[0])
			if isComponent {
				t.out.WriteString(".WriteValue(")
				t.visitComponent(node, sourceCode, depth+1)
				t.out.WriteString(")")
				t.last = node.EndByte()

			} else {
				t.out.WriteString(".WriteHTML(`")
				for i := 0; i < int(node.ChildCount()-1); i++ {
					node := node.Child(i)
					t.visitTag(node, sourceCode, depth)
				}
				t.last = node.EndByte()

				if isVoidElement(identifier) {
					t.out.WriteString(">")
				} else if fixTagClosing {
					t.out.WriteString(">")
					t.out.WriteString("</")
					t.out.WriteString(identifier)
					t.out.WriteString(">")
				} else {
					t.out.WriteString("/>")
				}
				t.out.WriteString("`)")

			}
		}
		t.out.WriteString(")")
		return
	case "jsx_element":
		t.out.WriteString("wax.Sub(w => w")
		{
			t.out.WriteString(".WriteHTML(`")
			if node.Child(0).ChildByFieldName("name") == nil {
				t.last = node.Child(0).EndByte()
				if int(node.ChildCount()-2) == 0 {
					t.out.WriteString("undefined")
				} else {
					for i := 1; i < int(node.ChildCount()-1); i++ {
						node := node.Child(i)
						t.visitTag(node, sourceCode, depth)
					}
				}
				t.last = node.EndByte()

			} else {

				t.visitTag(node, sourceCode, depth+1)
				t.last = node.EndByte()

			}
			t.out.WriteString("`)")
		}
		t.out.WriteString(")")
		return
	}
}

func (t *treeSitterVisitor) visitComponent(node *sitter.Node, sourceCode []byte, depth int) {
	nodeType := node.Type()
	switch nodeType {
	case "jsx_self_closing_element":
		identifier := node.ChildByFieldName("name").Content(sourceCode)
		isComponent := len(identifier) > 0 && unicode.IsUpper([]rune(identifier)[0])
		if isComponent {
			t.last = node.Child(1).EndByte()

			t.out.WriteString(identifier)
			t.out.WriteString("(")
			t.out.WriteString("{")
			for i := 2; i < int(node.ChildCount())-1; i++ {

				node := node.Child(i)
				t.formatTo(node, sourceCode)
				t.visitComponentTag(node, sourceCode, depth)
			}

			t.out.WriteString("}")
			t.out.WriteString(")")
		} else {
			// t.out.WriteString(".WriteHTML(`")
			// for i := 0; i < int(node.ChildCount()); i++ {
			// 	node := node.Child(i)
			// 	t.visitTag(node, sourceCode, depth)
			// }

			// t.out.WriteString("`)")
			// t.out.WriteString(")")
		}
		return

	case "jsx_element":
		identifier := node.Child(0).ChildByFieldName("name").Content(sourceCode)
		isComponent := len(identifier) > 0 && unicode.IsUpper([]rune(identifier)[0])
		if isComponent {
			t.last = node.Child(0).Child(1).EndByte()
			t.out.WriteString(identifier)
			t.out.WriteString("(")
			t.out.WriteString("{")
			for i := 2; i < int(node.Child(0).ChildCount())-1; i++ {
				node := node.Child(0).Child(i)
				t.visitComponentTag(node, sourceCode, depth)
			}
			if int(node.ChildCount())-1 > 1 {
				if int(node.ChildCount()-2) == 0 {
					// t.out.WriteString("undefined")
				} else {
					t.out.WriteString(`children`)
					t.out.WriteString(":")
					t.out.WriteString("[")
					// t.printChilds(node, sourceCode, depth)
					for i := 1; i < int(node.ChildCount())-1; i++ {
						node := node.Child(i)
						t.last = node.StartByte()
						t.visit(node, sourceCode, depth)
						t.out.WriteString(",")
					}
					t.out.WriteString("]")
				}
			}
			t.out.WriteString("}")
			t.out.WriteString(")")
		} else {
			// t.out.WriteString(".WriteHTML(`")
			// for i := 0; i < int(node.ChildCount()); i++ {
			// 	node := node.Child(i)

			// 	t.visitTag(node, sourceCode, depth)
			// }
			// t.out.WriteString("`)")
			// t.out.WriteString(")")
		}
		return
	}
}

func (t *treeSitterVisitor) visitComponentTag(node *sitter.Node, sourceCode []byte, depth int) {
	// t.printChilds(node, sourceCode, depth)
	nodeType := node.Type()
	switch nodeType {
	case "jsx_expression":
		expressionBody := node.Child(1)
		t.last = expressionBody.StartByte()
		t.visit(expressionBody, sourceCode, depth)
		t.last = node.EndByte()
		t.out.WriteString(",")
		return
	case "jsx_attribute":
		{
			handled := false
			// t.last = node.Child(0).EndByte()
			t.formatTo(node, sourceCode)

			if node.Child(0).Type() == "property_identifier" {
				attrName := node.Child(0).Content(sourceCode)
				if node.ChildCount() == 1 {
					t.out.WriteString(`"` + attrName + `"`)
					t.out.WriteString(":")
					t.out.WriteString("true")
					handled = true
				} else {
					switch true {
					case node.Child(2).Type() == "string":
						t.out.WriteString(`"` + attrName + `"`)
						t.out.WriteString(":")

						toWrite := node.Child(2).Content(sourceCode)
						t.out.WriteString(toWrite)
						t.last = node.EndByte()
						handled = true

					default:
						t.out.WriteString(`"` + attrName + `"`)
						t.out.WriteString(":")
						expressionBody := node.Child(2).Child(1)
						t.last = expressionBody.StartByte()
						t.visit(expressionBody, sourceCode, depth)
						t.last = node.EndByte()
						handled = true

					}
				}
				if !handled {
					t.out.WriteString(attrName + "*WAS_NOT_HANDLED*")
				}
				t.last = node.EndByte()
				t.out.WriteString(",")
				return

			}
			panic("foo")
		}
	case "jsx_text":
		t.out.WriteString("`" + node.Content(sourceCode) + "`")
		t.out.WriteString(",")
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		node := node.Child(i)
		t.visitTag(node, sourceCode, depth+1)
	}
	t.out.Write(sourceCode[t.last:node.EndByte()])
	t.last = node.EndByte()
}

var fixTagClosing = true

func (t *treeSitterVisitor) visitTag(node *sitter.Node, sourceCode []byte, depth int) {
	nodeType := node.Type()
	switch nodeType {
	case "jsx_self_closing_element":
		{
			identifier := node.ChildByFieldName("name").Content(sourceCode)
			isComponent := len(identifier) > 0 && unicode.IsUpper([]rune(identifier)[0])
			if isComponent {
				t.out.WriteString("`)")
				t.formatTo(node, sourceCode)
				t.out.WriteString(".WriteValue(")
				t.visitComponent(node, sourceCode, depth+1)
				t.out.WriteString(")")
				t.out.WriteString(".WriteHTML(`")
				t.last = node.EndByte()
			} else {
				var innerNode *sitter.Node
				for i := 0; i < int(node.ChildCount()-1); i++ {
					node := node.Child(i)
					if node.Type() == "jsx_attribute" &&
						node.Child(0).Content(sourceCode) == "value" &&
						identifier == "textarea" {
						innerNode = node.Child(2)
					} else {
						t.visitTag(node, sourceCode, depth+1)
					}
				}

				if isVoidElement(identifier) {
					t.out.WriteString(">")
				} else if fixTagClosing {
					t.out.WriteString(">")
					if innerNode != nil {
						t.out.WriteString("`)")
						t.out.WriteString(".WriteValue(")
						t.visitExpression(innerNode, sourceCode, depth+1)
						t.out.WriteString(")")
						t.out.WriteString(".WriteHTML(`")
					}
					t.out.WriteString("</")
					t.out.WriteString(identifier)
					t.out.WriteString(">")
				} else {
					t.out.WriteString("/>")
				}
				t.last = node.EndByte()
			}
			return
		}
	case "jsx_element":
		{
			identifier := node.Child(0).ChildByFieldName("name").Content(sourceCode)
			isComponent := len(identifier) > 0 && unicode.IsUpper([]rune(identifier)[0])
			if isComponent {
				t.out.WriteString("`)")
				t.out.WriteString(".WriteValue(")
				t.visitComponent(node, sourceCode, depth+1)
				t.out.WriteString(")")
				t.out.WriteString(".WriteHTML(`")
				t.last = node.EndByte()
			} else {
				for i := 0; i < int(node.ChildCount()-1); i++ {
					child := node.Child(i)
					t.visitTag(child, sourceCode, depth+1)
				}
				t.last = node.EndByte()
				if isVoidElement(identifier) {
					// noop
				} else {
					t.out.WriteString("</")
					t.out.WriteString(identifier)
					t.out.WriteString(">")
				}
			}
			return
		}
	case "jsx_opening_element":
		{
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				t.formatTo(child, sourceCode)

				if child.Type() == "jsx_expression" {
					t.out.WriteString("`)")
					t.out.WriteString(".WriteAttributes(")
					{
						// spread_element
						expressionBody := child.Child(1).Child(1)
						t.visitExpression(expressionBody, sourceCode, depth)
					}

					t.out.WriteString(")")
					t.out.WriteString(".WriteHTML(`")

					t.last = child.EndByte()
				} else {
					t.visitTag(child, sourceCode, depth+1)
				}
			}
			return
		}
	case "jsx_expression":
		t.out.Write(sourceCode[t.last:node.StartByte()])
		t.out.WriteString("`)")
		t.out.WriteString(".WriteValue(")
		{
			expressionBody := node.Child(1)
			t.last = expressionBody.StartByte()
			t.visit(expressionBody, sourceCode, depth)
			t.last = node.EndByte()
		}
		t.out.WriteString(")")
		t.out.WriteString(".WriteHTML(`")
		t.last = node.EndByte()
		return
	case "jsx_attribute":
		handled := false
		t.out.Write(sourceCode[t.last:node.StartByte()])
		if node.Child(0).Type() == "property_identifier" {
			attrName := node.Child(0).Content(sourceCode)

			if node.ChildCount() == 1 {
				t.out.WriteString(attrName)
				handled = true
			} else {
				switch true {
				case node.Child(2).Type() == "string":
					t.out.WriteString(attrName)
					t.out.WriteString("=")

					toWrite := node.Child(2).Content(sourceCode)
					toWrite = strings.ReplaceAll(toWrite, "\\", "\\\\")
					toWrite = strings.ReplaceAll(toWrite, "`", "\\`")
					t.out.WriteString(toWrite)
					handled = true

				case node.Child(2).Type() == "jsx_expression":
					t.out.WriteString("`)")
					t.out.WriteString(".WriteAttribute(`")
					t.out.WriteString(attrName)
					t.out.WriteString("`, ")
					{
						expressionBody := node.Child(2).Child(1)
						t.visitExpression(expressionBody, sourceCode, depth)
					}

					t.out.WriteString(")")
					t.out.WriteString(".WriteHTML(`")
					handled = true
				}
			}

			if !handled {
				t.out.WriteString(attrName + "*WAS_NOT_HANDLED*")
			}
			t.last = node.EndByte()
			return

		}

		panic("foo")
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		node := node.Child(i)
		t.visitTag(node, sourceCode, depth+1)
	}

	t.out.Write(sourceCode[t.last:node.EndByte()])
	t.last = node.EndByte()
}

func (t *treeSitterVisitor) visitExpression(node *sitter.Node, sourceCode []byte, depth int) {
	t.last = node.StartByte()
	t.visit(node, sourceCode, depth)
	t.last = node.EndByte()
}

func printNode(node *sitter.Node, sourceCode []byte, depth int) {
	fmt.Printf("%s%s [%v]: %q\n", indent(depth+1), node.Type(), node.ChildCount(), node.Content(sourceCode))
	for i := 0; i < int(node.ChildCount()); i++ {
		node := node.Child(i)
		printNode(node, sourceCode, depth+1)
	}
}

func indent(level int) string {
	return strings.Repeat(".", level*2) + strings.Repeat(" ", level*2)
}
