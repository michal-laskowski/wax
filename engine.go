package wax

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/dop251/goja"
)

type ViewResolver interface {
	ResolveViewFile(viewName string) (*url.URL, error)
	ResolveModuleFile(fromModule ModuleMeta, importPath string) (*url.URL, error)
	GetContent(url url.URL) (string, error)
}

type ExecutionError struct {
	File  url.URL
	Stack string
	Err   error
}

func (e ExecutionError) Error() string {
	return fmt.Sprintf("wax error while executing %s - %s", e.File.Path, e.Err.Error())
}

func New(viewResolver ViewResolver, options ...Option) *WaxEngine {
	result := &WaxEngine{
		globals:       make(map[string]any),
		globalScripts: []string{},
		viewResolver:  viewResolver,
		cache:         make(map[string]*goja.Program),
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func WithGlobalScript(path string) Option {
	return func(e *WaxEngine) {
		e.globalScripts = append(e.globalScripts, path)
	}
}

func WithGlobalObject(name string, o any) Option {
	return func(e *WaxEngine) {
		e.globals[name] = o
	}
}

type Option func(*WaxEngine)
type WaxEngine struct {
	globals       map[string]any
	globalScripts []string
	viewResolver  ViewResolver
	cache         map[string]*goja.Program
	cacheMu       sync.RWMutex
}

type WaxRunBinding struct {
	ViewResolver ViewResolver
	Globals      map[string]any
	Model        any
}

type ModuleMeta struct {
	URL    *url.URL
	isMain bool
}

func (this ModuleMeta) Dirname() string {
	return filepath.Dir(this.URL.Path)
}
func (this ModuleMeta) Filename() string {
	return filepath.Base(this.URL.Path)
}
func (this ModuleMeta) toJSObject() any {
	return map[string]any{
		"dirname":  filepath.ToSlash(this.Dirname()),
		"filename": this.Filename(),
		"url":      this.URL.String(),
		"main":     this.isMain,
		//resolve
	}
}

func (e *WaxEngine) RenderWith(out io.Writer, viewName string, binding WaxRunBinding) error {
	context := runContext{
		Model:        binding.Model,
		ViewResolver: binding.ViewResolver,
		Globals:      binding.Globals,
		Writer:       newWriter(),
	}
	if renderedView, err := e.renderView(viewName, context); err != nil {
		return err
	} else {
		_, err = out.Write([]byte(renderedView))
		return err
	}
}

func (this *WaxEngine) Render(out io.Writer, viewName string, model any) error {
	context := runContext{
		Model:        model,
		ViewResolver: this.viewResolver,
		Writer:       newWriter(),
	}
	if renderedView, err := this.renderView(viewName, context); err != nil {
		return err
	} else {
		_, err = out.Write([]byte(renderedView))
		return err
	}
}

type runContext struct {
	ViewResolver ViewResolver
	Globals      map[string]any
	Model        any
	Writer       *waxWriter
}

const InternalError = "internal error"

func (this *WaxEngine) getDoImport(fromModule ModuleMeta, context *runContext) func(name string) any {
	return func(importPath string) any {
		viewFilePath, err := context.ViewResolver.ResolveModuleFile(fromModule, importPath)
		if err != nil {
			panic(errors.Join(errors.New("undable to resolve module import - "+importPath), err))
		}

		vm := goja.New()

		moduleMeta := ModuleMeta{
			URL: viewFilePath,
		}
		globalImport := map[string]any{
			"meta":      moduleMeta.toJSObject(),
			"do_import": this.getDoImport(moduleMeta, context),
		}
		vm.GlobalObject().Set("import", globalImport)

		for k, v := range this.globals {
			vm.GlobalObject().Set(k, v)
		}
		for k, v := range context.Globals {
			vm.GlobalObject().Set(k, v)
		}
		vm.GlobalObject().Set("wax", WaxObj{})

		module := vm.NewObject()
		vm.Set("module", module)
		module.Set("exports", vm.NewObject())
		pc := this.loadProgram(moduleMeta, context)
		vm.RunProgram(pc)
		delete(globalImport, "do_import")

		result := module.Export()
		return result
	}
}

func (this *WaxEngine) loadProgram(module ModuleMeta, context *runContext) *goja.Program {
	this.cacheMu.RLock()
	pc, fromCache := this.cache[module.URL.String()]
	this.cacheMu.RUnlock()
	if !fromCache {
		this.cacheMu.Lock()
		defer this.cacheMu.Unlock()
		moduleCode, err := context.ViewResolver.GetContent(*module.URL)
		if err != nil {
			panic(err)
		} else if jsCode, err := transpile(module.URL.String(), moduleCode); err != nil {
			panic(err)
		} else {
			pc, err = goja.Compile(module.URL.String(), jsCode, true)
			this.cache[module.URL.String()] = pc
			if err != nil {
				panic(err)
			}
		}

	}
	return pc
}

func (this *WaxEngine) renderView(viewName string, context runContext) (string, error) {
	viewURI, err := context.ViewResolver.ResolveViewFile(viewName)
	if err != nil {
		return "", err
	}

	viewModuleMeta := ModuleMeta{URL: viewURI, isMain: true}

	program := this.loadProgram(viewModuleMeta, &context)

	vm := goja.New()
	globalImportObject := map[string]any{
		"meta": viewModuleMeta.toJSObject(),
	}
	vm.GlobalObject().Set("import", globalImportObject)
	module := make(map[string]any)
	exports := make(map[string]any)
	module["exports"] = exports
	vm.Set("module", module)

	for k, v := range this.globals {
		vm.GlobalObject().Set(k, v)
	}
	for k, v := range context.Globals {
		vm.GlobalObject().Set(k, v)
	}
	vm.GlobalObject().Set("wax", WaxObj{})
	for _, v := range this.globalScripts {

		moduleURI, err := context.ViewResolver.ResolveModuleFile(viewModuleMeta, v)
		if err != nil {
			panic(err)
		}
		moduleMeta := ModuleMeta{URL: moduleURI}
		scriptProgram := this.loadProgram(moduleMeta, &context)

		globalImportObject["do_import"] = this.getDoImport(moduleMeta, &context)
		_, err = vm.RunProgram(scriptProgram)
		if err != nil {
			panic(err)
		}

		delete(globalImportObject, "do_import")
	}
	globalImportObject["do_import"] = this.getDoImport(viewModuleMeta, &context)

	_, err = vm.RunProgram(program)
	delete(globalImportObject, "do_import")
	if err != nil {
		stack := "no stack"
		if exception, ok := err.(*goja.Exception); ok {

			stack = exception.String()
		}

		return "", ExecutionError{
			File:  *viewURI,
			Stack: stack,
			Err:   err,
		}
	}
	vm.ClearInterrupt()
	render, ok := goja.AssertFunction(vm.ToValue(exports[viewName]))

	if !ok {
		var defaulF goja.Callable
		defaulF, ok = goja.AssertFunction(vm.ToValue(module["default"]))

		if ok {
			var defaultV goja.Value

			defaultV, _ = defaulF(goja.Null())
			render, ok = goja.AssertFunction(defaultV)
		}
	}

	if !ok {
		return InternalError, fmt.Errorf("could not find function for view '%s' in %s'", viewName, viewURI.String())
	}

	view, err := render(goja.Undefined(), vm.ToValue(context.Model))
	if err != nil {
		stack := "no stack"
		if exception, ok := err.(*goja.Exception); ok {

			stack = exception.String()
		}

		return "", ExecutionError{
			File:  *viewURI,
			Stack: stack,
			Err:   err,
		}

	}
	result := view.String()

	return result, nil
}

type WaxObj struct {
}

func (w *WaxObj) NewWriter() *waxWriter {
	return newWriter()
}

func (w *WaxObj) Raw(v string) templateResult {
	return templateResult(v)
}
