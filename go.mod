module github.com/michal-laskowski/wax

go 1.23.0

require github.com/dop251/goja v0.0.0-20250531102226-cb187b08699c

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20250607225305-033d6d78b36a // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	golang.org/x/text v0.26.0 // indirect
)

require (
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/tree-sitter/tree-sitter-typescript v0.23.3-0.20250130221139-75b3874edb2d
	golang.org/x/net v0.41.0
)

retract v0.0.1 // WIP

retract v0.0.2 // to retract 0.0.1
