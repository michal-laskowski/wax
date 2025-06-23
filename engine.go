package wax

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/dop251/goja"
)

type ViewResolver interface {
	ResolveViewFile(viewName string) (*url.URL, error)
	ResolveModuleFile(fromModule ModuleMeta, importPath string) (*url.URL, error)
	GetContent(url url.URL) (string, error)
}

type TypeScriptTranspiler interface {
	Transpile(fileName string, fileContent string) (string, error)
}

func New(viewResolver ViewResolver, options ...Option) *Engine {
	result := &Engine{
		globals:       make(map[string]any),
		globalScripts: []string{},
		viewResolver:  viewResolver,
		cache:         make(map[string]*goja.Program),
		transpiler:    NewTreeSitterTranspiler(),
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func WithGlobalScript(path string) Option {
	return func(e *Engine) {
		e.globalScripts = append(e.globalScripts, path)
	}
}

func WithGlobalObject(name string, o any) Option {
	return func(e *Engine) {
		e.globals[name] = o
	}
}

type Error struct {
	File  url.URL
	Stack string
	Phase string
	Err   error
}

var (
	PhaseLoading     = "load"
	PhaseCompilation = "compile"
	PhaseExec        = "execute"
	PhaseOther       = "other"
)

func (e Error) Error() string {
	if e.Phase == PhaseExec {
		return e.Err.Error()
	}
	return fmt.Sprintf("wax: %s", e.Err.Error())
}

func (e Error) ErrorDetailed() string {
	if e.Phase == PhaseExec {
		return fmt.Sprintf("wax error: %s: %s - %s - %s", e.Phase, e.File.Path, e.Err.Error(), e.Stack)
	}
	return fmt.Sprintf("wax error [%s]: '%s': %s", e.Phase, e.File.Path, e.Err.Error())
}

type (
	Option func(*Engine)
	Engine struct {
		globals       map[string]any
		globalScripts []string
		viewResolver  ViewResolver
		cache         map[string]*goja.Program
		cacheMu       sync.RWMutex

		transpiler TypeScriptTranspiler
	}
)

type RunBinding struct {
	ViewResolver ViewResolver
	Globals      map[string]any
	Model        any
}

type ModuleMeta struct {
	URL    *url.URL
	isMain bool
}

// TODO
// func (m ModuleMeta) toJSObject() any {
// 	return map[string]any{
// 		"dirname":  filepath.ToSlash(m.Dirname()),
// 		"filename": m.Filename(),
// 		"url":      m.URL.String(),
// 		"main":     m.isMain,
// 	}
// }

func (e *Engine) RenderWith(out io.Writer, viewName string, binding RunBinding) error {
	context := runContext{
		Model:        binding.Model,
		ViewResolver: binding.ViewResolver,
		Globals:      binding.Globals,
		out:          out,
	}
	viewURI, err := context.ViewResolver.ResolveViewFile(viewName)
	if err != nil {
		return err
	}

	if err := e.renderView(viewURI, viewName, &context); err != nil {
		// must be always WaxError
		return err
	}
	return nil
}

func (e *Engine) Render(out io.Writer, viewName string, model any) error {
	return e.RenderWith(out, viewName, RunBinding{
		Model:        model,
		ViewResolver: e.viewResolver,
	})
}

type runContext struct {
	ViewResolver ViewResolver
	Globals      map[string]any
	Model        any
	out          io.Writer
}

const InternalError = "internal error"

func (e *Engine) load(context *runContext, wax *waxJSObj, viewFilePath *url.URL) (goja.Value, error) {
	if moduleExports := wax.GetModule(viewFilePath.String()); moduleExports != nil {
		return moduleExports, nil
	}

	moduleMeta := ModuleMeta{
		URL: viewFilePath,
	}
	globalImport := wax.DefineModule(&moduleMeta)
	p, err := e.loadModuleImport(&moduleMeta, context)
	if err != nil {
		return nil, err
	}

	_, err = wax.vm.RunProgram(p)
	if err != nil {
		return nil, err
	}
	return globalImport, nil
}

func (e *Engine) loadModuleImport(module *ModuleMeta, context *runContext) (*goja.Program, error) {
	key := module.URL.String()
	e.cacheMu.RLock()
	pc, fromCache := e.cache[key]
	if fromCache {
		e.cacheMu.RUnlock()
		return pc, nil
	}

	e.cacheMu.RUnlock()
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	moduleCode, err := context.ViewResolver.GetContent(*module.URL)
	if err != nil {
		return nil, err
	}

	jsCode, err := e.transpiler.Transpile(key, moduleCode)
	if err != nil {
		return nil, Error{
			File:  *module.URL,
			Phase: PhaseLoading,
			Err:   err,
		}
	}

	jsCode = fmt.Sprintf(";(function (module) {;\n%s\n;})(wax.GetModule('%s'));", jsCode, key)
	compiled, err := goja.Compile(key, jsCode, true)
	e.cache[key] = compiled
	if err != nil {
		return nil, Error{
			File:  *module.URL,
			Phase: PhaseCompilation,
			Err:   err,
		}
	}
	return compiled, nil
}

func (e *Engine) renderView(moduleURI *url.URL, viewName string, context *runContext) error {
	viewModuleMeta := ModuleMeta{URL: moduleURI, isMain: true}

	vm := goja.New()
	waxObj := newWaxObj(e, vm, context)
	vm.GlobalObject().DefineDataProperty("wax", waxObj.obj, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)

	for k, v := range e.globals {
		vm.GlobalObject().Set(k, v)
	}
	for k, v := range context.Globals {
		vm.GlobalObject().Set(k, v)
	}

	for _, v := range e.globalScripts {
		moduleURI, err := context.ViewResolver.ResolveModuleFile(viewModuleMeta, v)
		if err != nil {
			return err
		}

		_, err = e.load(context, waxObj, moduleURI)
		if err != nil {
			return err
		}
	}

	mainModule, err := e.load(context, waxObj, moduleURI)
	if err != nil {
		return errors.Join(errors.New("could not load main module"), err)
	}
	if mainModule == nil {
		return fmt.Errorf("main module not loaded")
	}

	mainModuleExports, ok := mainModule.(*goja.Object).Get("exports").(*goja.Object)
	if !ok {
		panic("invalid state: no exports")
	}

	viewValue := mainModuleExports.Get(viewName)
	if viewValue == nil {
		viewValue = mainModule.ToObject(vm).Get("default")
	}

	if viewValue == nil {
		return Error{
			File:  *viewModuleMeta.URL,
			Phase: PhaseLoading,
			Err:   fmt.Errorf("could not find function for view '%s'", viewName),
		}
	}

	asCallable, ok := goja.AssertFunction(viewValue)
	if !ok {
		return Error{
			File:  *viewModuleMeta.URL,
			Phase: PhaseLoading,
			Err:   fmt.Errorf("expected to get function as view '%s'", viewName),
		}
	}

	gojaErr := vm.Try(func() {
		view, err := asCallable(goja.Undefined(), vm.ToValue(context.Model))
		if err != nil {
			panic(err)
		}

		writer := newWriter(context.out, vm)
		writer.process(view, vm)
	})

	if gojaErr != nil {
		stack := gojaErr.Error()

		return Error{
			File:  *viewModuleMeta.URL,
			Stack: stack,
			Phase: PhaseExec,
			Err:   gojaErr,
		}
	}

	return nil
}
