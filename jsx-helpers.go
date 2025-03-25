package wax

import (
	"regexp"
	"slices"
	"strings"
)

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
	// todo ??
	"reference",
}

func isVoidElement(name string) bool {
	return slices.Contains(voidElements, name)
}

type ImportClause struct {
	ImportedDefaultBinding string
	NameSpaceImport        string
	NamedImports           map[string]string
	ModuleName             string
}

var (
	moduleOnlyRe   = regexp.MustCompile(`^["']([^"']+)["']$`)
	moduleRe       = regexp.MustCompile(`\bfrom\s+["']([^"']+)["']`)
	namespaceRe    = regexp.MustCompile(`\*\s+as\s+([a-zA-Z_$][a-zA-Z0-9_$]*)`)
	namedImportsRe = regexp.MustCompile(`\{([^}]*)\}`)
)

func parseImportClause(input string) *ImportClause {
	ic := &ImportClause{NamedImports: make(map[string]string)}

	input = strings.TrimSpace(strings.TrimPrefix(input, "import "))
	input = strings.TrimSuffix(input, ";")

	// `import "module-name"`
	if match := moduleOnlyRe.FindStringSubmatch(input); match != nil {
		ic.ModuleName = match[1]
		return ic
	}

	var moduleMatch []string
	if moduleMatch = moduleRe.FindStringSubmatch(input); moduleMatch != nil {
		ic.ModuleName = moduleMatch[1]
		input = strings.TrimSpace(strings.Replace(input, moduleMatch[0], "", 1))
	}

	// (* as ns)
	if match := namespaceRe.FindStringSubmatch(input); match != nil {
		ic.NameSpaceImport = match[1]
		input = strings.TrimSpace(strings.Replace(input, match[0], "", 1))
	}

	// ({ ... })
	if match := namedImportsRe.FindStringSubmatch(input); match != nil {
		for _, item := range strings.Split(match[1], ",") {
			parts := strings.Split(strings.TrimSpace(item), " as ")
			if len(parts) == 2 {
				ic.NamedImports[strings.TrimSpace(parts[1])] = strings.TrimSpace(parts[0])
			} else {
				ic.NamedImports[parts[0]] = parts[0]
			}
		}
		input = strings.TrimSpace(strings.Replace(input, match[0], "", 1))
	}

	// ImportedDefaultBinding
	if input != "" && !strings.HasPrefix(input, "{") && !strings.HasPrefix(input, "*") {
		ic.ImportedDefaultBinding = strings.SplitN(input, ",", 2)[0]
		ic.ImportedDefaultBinding = strings.TrimSpace(ic.ImportedDefaultBinding)
	}

	return ic
}
