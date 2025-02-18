package wax

import (
	"errors"
	"fmt"

	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/matthewmueller/jsx"
	"github.com/matthewmueller/jsx/ast"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

var ErrNotSupported = errors.New("wax does not support this syntax")

func transpile(fileName string, fileContent string) (string, error) {
	transformResult := esbuild.Transform(fileContent, esbuild.TransformOptions{
		Loader:           esbuild.LoaderTSX,
		JSX:              esbuild.JSXPreserve,
		Format:           esbuild.FormatDefault,
		MinifyWhitespace: false,
		MinifySyntax:     false,
	})
	if len(transformResult.Errors) != 0 {
		return "", fmt.Errorf("error while parsing JSX/TSX from file %s - %+v", fileName, transformResult.Errors)
	}
	jsScript := string(transformResult.Code)
	return transpileJSX(fileName, jsScript)
}

func transpileJSX(fileName string, fileContent string) (string, error) {
	jsScript := string(fileContent)

	script, err := jsx.Parse(fileName, jsScript)
	if err != nil {
		return "", fmt.Errorf("error while parsing JSX from file %s - %w", fileName, err)
	}

	printer := &modulePrinter{}
	script.Visit(printer)

	result := printer.String()
	return result, nil
}

var impReg = regexp.MustCompile(`(?m)import(?:(?:(?:[ \n\t]+([^ *\n\t\{\},]+)[ \n\t]*(?:,|[ \n\t]+))?([ \n\t]*\{(?:[ \n\t]*[^ \n\t"'\{\}]+[ \n\t]*,?)+\})?[ \n\t]*)|[ \n\t]*\*[ \n\t]*as[ \n\t]+([^ \n\t\{\}]+)[ \n\t]+)from[ \n\t]*(?:['"])([^'"\n]+)(['"])`)
var importDefaultReg = regexp.MustCompile(`(?m)import\s+(\w+)\sfrom[ \n\t]*(?:['"])([^'"\n]+)(['"])`)
var importPlain = regexp.MustCompile(`(?m)import\s+[ \n\t]*(?:['"])([^'"\n]+)(['"])`)

func toWAXImport(input string) string {
	result := input
	result = importDefaultReg.ReplaceAllString(result, "var $1 = (globalThis.import.do_import('$2').default ?? (()=> {throw 'no default export'})())")
	result = impReg.ReplaceAllString(result, "var $1$2$3 = globalThis.import.do_import('$4').exports")
	result = importPlain.ReplaceAllString(result, "globalThis.import.do_import('$1').exports")
	return result
}

var exportRe = regexp.MustCompile(`(?m)export(?:(?:(?:[ \n\t]+([^ *\n\t\{\},]+)[ \n\t]*(?:,|[ \n\t]+))?([ \n\t]*\{(?:[ \n\t]*[^ \n\t"'\{\}]+[ \n\t]*,?)+\})?[ \n\t]*)|[ \n\t]*\*[ \n\t]*as[ \n\t]+([^ \n\t\{\}]+)[ \n\t]+)from[ \n\t]*(?:['"])([^'"\n]+)(?:['"])`)
var exportNamed = regexp.MustCompile(`(?m)export\s+(const|let|var)\s+(\w+)`)
var exportClass = regexp.MustCompile(`(?m)export\s+(class)\s+(\w+)`)
var exportNamed2 = regexp.MustCompile(`(?m)export\s+(function)\s+(\w+)`)
var exportDefaultNF = regexp.MustCompile(`(?m)export\s+default\s+(const|let|var|class)\s+(\w+)`)
var exportDefaultF = regexp.MustCompile(`(?m)export\s+default\s+(function)\s+(\w+)`)

// var exportReIm = regexp.MustCompile(`(?m)\{\s*(?:((\w+)\s?(?:as)?\s?(\w+))(?:\,?\s*)?)*\}`)
var exportReIm2 = regexp.MustCompile(`(\w+)(?:\s+as\s+(\w+))?`)

func toWAXExport(input string) string {
	result := input
	if exportRe.MatchString(input) {
		result = exportRe.ReplaceAllStringFunc(input, func(m string) string {
			main := exportRe.FindStringSubmatch(m)
			output := exportReIm2.ReplaceAllStringFunc(main[2], func(match string) string {
				parts := exportReIm2.FindStringSubmatch(match)
				if parts[2] != "" {
					return fmt.Sprintf("%s: _f.%s", parts[2], parts[1])
				}
				return fmt.Sprintf("%s: _f.%s", parts[1], parts[1])
			})
			output = fmt.Sprintf(";(function() {let _f = globalThis.import.do_import('%s').exports; Object.assign(module.exports, %s)})();", main[4], output)
			return output
		})
	}
	result = exportNamed.ReplaceAllString(result, "$1 $2 = module.exports.$2")
	result = exportClass.ReplaceAllString(result, "module.exports.$2 = class $2")
	result = exportNamed2.ReplaceAllString(result, "module.exports.$2 = $2;$1 $2")

	result = exportDefaultNF.ReplaceAllString(result, "module.default = ()=> module.exports.$2;module.exports.$2")
	result = exportDefaultF.ReplaceAllString(result, "module.default = ()=> module.exports.$2;module.exports.$2 = $2;$1 $2")

	return result
}

type modulePrinter struct {
	s strings.Builder
}

func (p *modulePrinter) VisitScript(s *ast.Script) {
	for _, fragment := range s.Body {
		fragment.Visit(p)
	}
}

func (p *modulePrinter) VisitText(t *ast.Text) {
	toWrite := toWAXImport(t.Value)
	toWrite = toWAXExport(toWrite)
	p.s.WriteString(toWrite)
}

func (p *modulePrinter) VisitComment(c *ast.Comment) {
	p.s.WriteString(c.String())
}

func (p *modulePrinter) VisitField(f *ast.Field) {
	switch f.Value.(type) {

	case *ast.StringValue:
		p.s.WriteString(f.Name)
		p.s.WriteString("=")
		p.s.WriteString(f.Value.String())

	case *ast.BoolValue:
		p.s.WriteString(f.Name)

	case *ast.Expr:
		p.s.WriteString(f.Name)
		p.s.WriteString("=")
		p.s.WriteString("\"")
		f.Value.Visit(p)
		p.s.WriteString("\"")
	}
}

func (p *modulePrinter) VisitStringValue(s *ast.StringValue) {
	toWrite := s.Value
	toWrite = strings.ReplaceAll(toWrite, "`", "\\`")
	p.s.WriteString(toWrite)
}

func (p *modulePrinter) VisitExpr(e *ast.Expr) {
	if len(e.Fragments) == 1 {
		switch e.Fragments[0].(type) {

		case *ast.Comment:
			return
		}
	}

	p.s.WriteString("`)")
	p.s.WriteString(".WriteValue(")
	for _, fragment := range e.Fragments {
		fragment.Visit(p)
	}
	p.s.WriteString(")")
	p.s.WriteString(".WriteHTML(`")
}

func (p *modulePrinter) VisitBoolValue(b *ast.BoolValue) {
	p.s.WriteString(strconv.Quote(strconv.FormatBool(b.Value)))
}

func (p *modulePrinter) VisitElement(toVisit *ast.Element) {
	visitor := visitor_TAG{}
	visitor.Process(toVisit)
	content, err := visitor.String()
	if err != nil {
		panic(err)
	}
	p.s.WriteString(content)
}

func isWhitespaceOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func (p *modulePrinter) String() string {
	return p.s.String()
}

type visitor_TAG struct {
	s   strings.Builder
	err error

	inSVG bool
}

func (p *visitor_TAG) Process(toVisit ast.Fragment) {
	p.s.WriteString("wax.NewWriter()")
	p.s.WriteString(".WriteHTML(`")
	toVisit.Visit(p)
	p.s.WriteString("`)")
	p.s.WriteString(".Result()")
}

var voidElements = []string{
	"area",
	"base",
	"br",
	"col",
	"command",
	"embed",
	"hr",
	"img",
	"input",
	"keygen",
	"link",
	"meta",
	"param",
	"source",
	"track",
	"wbr",
	//todo ??
	"reference",
}

func isVoidElement(name string) bool {
	return slices.Contains(voidElements, name)
}

func isCustomTag(name string) bool {
	if strings.ToLower(name) == name && strings.Contains(name, "-") {
		return true
	}
	return false
}

func (p *visitor_TAG) VisitElement(toVisit *ast.Element) {

	isComponent := len(toVisit.Name) > 0 && unicode.IsUpper([]rune(toVisit.Name)[0])

	if !isComponent {
		isTextArea := toVisit.Name == "textarea"
		var textAreaValueAttribute *ast.Field
		if toVisit.Name != "" {
			p.s.WriteString("<")
			p.s.WriteString(toVisit.Name)

			if len(toVisit.Attrs) > 0 {
				p.s.WriteString(" ")
				for i, attr := range toVisit.Attrs {
					if i > 0 {
						p.s.WriteString(" ")
					}
					if isTextArea {
						switch attr := attr.(type) {
						case *ast.Field:
							if attr.Name != "value" {
								attr.Visit(p)
							} else {
								textAreaValueAttribute = attr
							}
						default:
							attr.Visit(p)
						}
					} else {
						attr.Visit(p)

					}
				}
			}
			p.s.WriteString(">")
		}

		if isVoidElement(toVisit.Name) && len(toVisit.Children) != 0 {
			p.err = fmt.Errorf("invalid: void element with child")
			return
		}

		if !isComponent && !isCustomTag(toVisit.Name) && toVisit.SelfClosing != isVoidElement(toVisit.Name) {
			if toVisit.Name == "script" {

			} else if p.inSVG == false {
				//p.err = fmt.Errorf("use proper closing. '%s' should be SelfClosing=%v", toVisit.Name, isVoidElement(toVisit.Name))
			}
		}

		if isTextArea {
			if len(toVisit.Children) != 0 {
				p.err = fmt.Errorf("textarea should not have child. Use value attribute.")
				return
			}
			if textAreaValueAttribute != nil {
				textAreaValueAttribute.Value.Visit(p)
			}
		} else {
			p.inSVG = (toVisit.Name == "svg")
			for _, child := range toVisit.Children {
				child.Visit(p)
			}
			p.inSVG = false
		}
		if toVisit.Name != "" && !isVoidElement(toVisit.Name) {
			p.s.WriteString("</")
			p.s.WriteString(toVisit.Name)
			p.s.WriteString(">")
		}
	} else {
		p.s.WriteString("`)")
		p.s.WriteString(".WriteValue(")
		p.VisitComponentElement(toVisit)
		p.s.WriteString(")")
		p.s.WriteString(".WriteHTML(`")
	}
}

var isAlpha = regexp.MustCompile(`^[A-Za-z]+$`).MatchString

func (p *visitor_TAG) VisitComponentElement(toVisit *ast.Element) {

	p.s.WriteString(toVisit.Name)
	p.s.WriteString("(")

	{
		p.s.WriteString("{")
		for i, attr := range toVisit.Attrs {
			switch attr := attr.(type) {

			case *ast.Field:
				if isAlpha(attr.Name) {
					p.s.WriteString(attr.Name)
				} else {
					p.s.WriteString(strconv.Quote(attr.Name))

				}
				p.s.WriteString(fmt.Sprintf("/* %v*/: ", i+1))

				jsP := visitor_JS{}
				attr.Value.Visit(&jsP)
				toWrite, err := jsP.String()
				if err != nil {
					panic(jsP.err)
				}
				p.s.WriteString(toWrite)

			case *ast.Expr:
				p.s.WriteString(attr.Fragments[0].String())
			}
			p.s.WriteString(",")
		}

		rChildContent := []string{}
		for _, child := range toVisit.Children {
			switch child := child.(type) {
			case *ast.Text:
				if isWhitespaceOnly(child.String()) {
					continue
				}
			}

			visitor := visitor_TAG{}
			visitor.Process(child)
			content, err := visitor.String()
			if err != nil {
				panic(err)
			}
			rChildContent = append(rChildContent, content)
		}

		if len(rChildContent) > 0 {
			p.s.WriteString("children: ")
			if len(rChildContent) > 1 {
				p.s.WriteString("[")
				{
					for _, cc := range rChildContent {
						p.s.WriteString(cc)
						p.s.WriteString(", ")
					}
				}
				p.s.WriteString("]")
			} else if len(rChildContent) == 1 {
				p.s.WriteString(rChildContent[0])
			}
		}

		p.s.WriteString("}")
	}

	p.s.WriteString(")")
}

func (p *visitor_TAG) VisitScript(n *ast.Script) { p.notSupported(n) }
func (p *visitor_TAG) VisitText(n *ast.Text) {
	toWrite := n.String()
	toWrite = strings.ReplaceAll(toWrite, "`", "\\`")
	p.s.WriteString(toWrite)
}

func (p *visitor_TAG) VisitField(f *ast.Field) {
	attrName := strings.ToLower(f.Name)
	switch f.Value.(type) {

	case *ast.StringValue:
		p.s.WriteString(attrName)
		p.s.WriteString("=")
		p.s.WriteString(strings.ReplaceAll(f.Value.String(), "$", "\\$"))

	case *ast.BoolValue:
		p.s.WriteString(attrName)

	case *ast.Expr:
		p.s.WriteString("`)")
		p.s.WriteString(".WriteAttribute(`")
		p.s.WriteString(attrName)
		p.s.WriteString("`, ")
		jsP := visitor_JS{}
		f.Value.Visit(&jsP)
		toWrite, err := jsP.String()
		if err != nil {
			panic(jsP.err)
		}
		p.s.WriteString(toWrite)

		p.s.WriteString(")")
		p.s.WriteString(".WriteHTML(`")
	}
}

func (p *visitor_TAG) VisitStringValue(n *ast.StringValue) { p.notSupported(n) }
func (p *visitor_TAG) VisitExpr(toVisit *ast.Expr) {
	if len(toVisit.Fragments) == 1 {
		switch toVisit.Fragments[0].(type) {
		case *ast.Comment:
			return
		}
	}

	p.s.WriteString("`)")
	p.s.WriteString(".WriteValue(")
	for _, fragment := range toVisit.Fragments {
		switch fragment := fragment.(type) {
		case *ast.Text:
			p.s.WriteString(fragment.String())
		case *ast.Element:
			visitor := visitor_TAG{}
			visitor.Process(fragment)
			content, err := visitor.String()
			if err != nil {
				panic(err)
			}
			p.s.WriteString(content)
		case *ast.Comment:
			continue
		default:
			fragment.Visit(p)

		}
	}
	p.s.WriteString(")")
	p.s.WriteString(".WriteHTML(`")
}
func (p *visitor_TAG) VisitBoolValue(n *ast.BoolValue) { p.notSupported(n) }
func (p *visitor_TAG) VisitComment(n *ast.Comment)     {}
func (p *visitor_TAG) notSupported(n any) {
	p.err = errors.Join(ErrNotSupported, fmt.Errorf("TSXTagVisitor:Node %T: %+v", n, n))
}
func (p *visitor_TAG) String() (string, error) {
	return p.s.String(), p.err
}

type visitor_JS struct {
	s   strings.Builder
	err error
}

func (p *visitor_JS) notSupported(n any) {
	p.err = errors.Join(ErrNotSupported, fmt.Errorf("JSPrinter:Node %T: %+v", n, n))
}

func (p *visitor_JS) VisitScript(n *ast.Script) { p.notSupported(n) }

func (p *visitor_JS) VisitText(n *ast.Text) {
	toWrite := n.String()
	p.s.WriteString(toWrite)
}

func (p *visitor_JS) VisitField(f *ast.Field) {
	switch v := f.Value.(type) {

	case *ast.StringValue:
		p.s.WriteString(v.String())

	case *ast.BoolValue:
		p.s.WriteString(v.String())

	case *ast.Expr:
		p := modulePrinter{}
		p.VisitExpr(v)

	}
}

func (p *visitor_JS) VisitStringValue(n *ast.StringValue) {
	toWrite := n.String()
	toWrite = strings.ReplaceAll(toWrite, "`", "\\`")
	p.s.WriteString(toWrite)
}

func (p *visitor_JS) VisitExpr(n *ast.Expr) {
	for _, fragment := range n.Fragments {
		fragment.Visit(p)
	}
}

func (p *visitor_JS) VisitBoolValue(n *ast.BoolValue) {
	p.s.WriteString(n.String())
}

func (p *visitor_JS) VisitElement(n *ast.Element) {
	visitor := visitor_TAG{}
	visitor.Process(n)
	content, _ := visitor.String()
	p.s.WriteString(content)
}

func (p *visitor_JS) VisitComment(n *ast.Comment) {
}

func (p *visitor_JS) String() (string, error) {
	return p.s.String(), p.err
}
