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

type WaxError struct {
	File  url.URL
	Stack string
	Phase string
	Err   error
}

var (
	PHASE_loading     = "load"
	PHASE_compilation = "compile"
	PHASE_Exec        = "execute"
	PHASE_Other       = "other"
)

func (e WaxError) Error() string {
	if e.Phase == PHASE_Exec {
		return fmt.Sprintf("wax: %s", e.Err.Error())
	}
	return fmt.Sprintf("wax: %s", e.Err.Error())
}

func (e WaxError) ErrorDetailed() string {
	if e.Phase == PHASE_Exec {
		return fmt.Sprintf("wax error: %s: %s - %s - %s", e.Phase, e.File.Path, e.Err.Error(), e.Stack)
	}
	return fmt.Sprintf("wax error [%s]: '%s': %s", e.Phase, e.File.Path, e.Err.Error())
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

type (
	Option    func(*WaxEngine)
	WaxEngine struct {
		globals       map[string]any
		globalScripts []string
		viewResolver  ViewResolver
		cache         map[string]*goja.Program
		cacheMu       sync.RWMutex
	}
)

type WaxRunBinding struct {
	ViewResolver ViewResolver
	Globals      map[string]any
	Model        any
}

type ModuleMeta struct {
	URL    *url.URL
	isMain bool
}

func (m ModuleMeta) Dirname() string {
	return filepath.Dir(m.URL.Path)
}

func (m ModuleMeta) Filename() string {
	return filepath.Base(m.URL.Path)
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

func (e *WaxEngine) RenderWith(out io.Writer, viewName string, binding WaxRunBinding) error {
	context := runContext{
		Model:        binding.Model,
		ViewResolver: binding.ViewResolver,
		Globals:      binding.Globals,
		Writer:       e.getNewWriter(),
		modules:      map[string]any{},
	}
	viewURI, err := context.ViewResolver.ResolveViewFile(viewName)
	if err != nil {
		return err
	}

	if renderedView, err := e.renderView(viewURI, viewName, &context); err != nil {
		var waxError WaxError
		if errors.As(err, &waxError) {
			return waxError
		}
		return WaxError{
			File:  *viewURI,
			Phase: PHASE_Other,
			Err:   err,
		}
	} else {
		_, err = out.Write([]byte(renderedView))
		return err
	}
}

func (e *WaxEngine) Render(out io.Writer, viewName string, model any) error {
	return e.RenderWith(out, viewName, WaxRunBinding{
		Model:        model,
		ViewResolver: e.viewResolver,
	})
}

type runContext struct {
	ViewResolver ViewResolver
	Globals      map[string]any
	Model        any
	Writer       *waxWriter
	modules      map[string]any
}

const InternalError = "internal error"

func (e *WaxEngine) getNewWriter() *waxWriter {
	return newWriter()
}

func (e *WaxEngine) getDoImport(mVM *goja.Runtime, fromModule ModuleMeta, context *runContext) func(name string) (any, error) {
	return func(importPath string) (any, error) {
		viewFilePath, err := context.ViewResolver.ResolveModuleFile(fromModule, importPath)
		if err != nil {
			panic(errors.Join(fmt.Errorf("could not resolve module import '%s' from '%s'", importPath, fromModule.URL.String()), err))
		}

		if moduleExports, ok := context.modules[viewFilePath.String()]; ok {
			return moduleExports, nil
		}

		moduleMeta := ModuleMeta{
			URL: viewFilePath,
		}
		globalImport := map[string]any{
			"meta": map[string]any{
				"dirname":  filepath.ToSlash(filepath.Dir(viewFilePath.Path)),
				"filename": filepath.Base(viewFilePath.Path),
				"url":      viewFilePath.String(),
				"main":     false,
				"resolve": func(m string) (any, error) {
					if r, err := context.ViewResolver.ResolveModuleFile(fromModule, m); err != nil {
						return nil, err
					} else {
						return r.String(), nil
					}
				},
			},
			"do_import": e.getDoImport(mVM, moduleMeta, context),
			"exports":   mVM.NewObject(),
			"resolve": func(m string) (any, error) {
				if r, err := context.ViewResolver.ResolveModuleFile(fromModule, m); err != nil {
					return nil, err
				} else {
					return r.String(), nil
				}
			},
		}
		context.modules[viewFilePath.String()] = globalImport

		p, err := e.loadModuleImport(moduleMeta, context)
		if err != nil {
			return nil, err
		}

		_, err = mVM.RunProgram(p)
		if err != nil {
			// TODO
			// if exception, ok := err.(*goja.Exception); ok {
			// 	stack = exception.String()
			// }

			// return "", WaxError{
			// 	File:  *viewModuleMeta.URL,
			// 	Stack: stack,
			// 	Err:   err,
			// }
			return nil, err
		}

		result := context.modules[viewFilePath.String()]
		return result, nil
	}
}

func (e *WaxEngine) loadModuleImport(module ModuleMeta, context *runContext) (*goja.Program, error) {
	e.cacheMu.RLock()
	pc, fromCache := e.cache[module.URL.String()]
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

	if jsCode, err := NewTreeSitterTranspiler().Transpile(module.URL.String(), moduleCode); err != nil {
		return nil, WaxError{
			File:  *module.URL,
			Phase: PHASE_loading,
			Err:   err,
		}
	} else {
		jsCode = fmt.Sprintf(";(function (module) {;\n%s\n;})(GetModule('%s'));", jsCode, module.URL.String())
		println(jsCode)
		compiled, err := goja.Compile(module.URL.String(), jsCode, true)
		e.cache[module.URL.String()] = pc
		if err != nil {
			return nil, WaxError{
				File:  *module.URL,
				Phase: PHASE_compilation,
				Err:   err,
			}
		}
		return compiled, nil
	}
}

func (c *runContext) getModule(s string) (map[string]any, error) {
	if moduleExports, ok := c.modules[s]; ok {
		return moduleExports.(map[string]any), nil
	}
	return nil, fmt.Errorf("could not resolve module '%s'", s)
}

func (e *WaxEngine) renderView(moduleURI *url.URL, viewName string, context *runContext) (string, error) {
	viewModuleMeta := ModuleMeta{URL: moduleURI, isMain: true}

	vm := goja.New()
	// globalImportObject := map[string]any{
	// 	"meta": viewModuleMeta.toJSObject(),
	// }
	// TODO vm.GlobalObject().Set("import", globalImportObject)
	// globalImportObject["do_import"] = mainDoImport

	// exports := make(map[string]any)
	// mainModule := map[string]any{
	// 	"meta":      viewModuleMeta.toJSObject(),
	// 	"do_import": e.getDoImport(vm, viewModuleMeta, context),
	// 	"exports":   exports,
	// }
	// vm.Set("module", mainModule)

	vm.Set("GetModule", context.getModule)

	for k, v := range e.globals {
		vm.GlobalObject().Set(k, v)
	}
	for k, v := range context.Globals {
		vm.GlobalObject().Set(k, v)
	}
	vm.GlobalObject().Set("wax", waxJSObj{
		engine: e,
		vm:     vm,
	})

	mainDoImport := e.getDoImport(vm, viewModuleMeta, context)
	for _, v := range e.globalScripts {
		mainDoImport(v)
	}

	_, err := mainDoImport(viewModuleMeta.Filename())
	if err != nil {
		return "", err
	}
	vm.ClearInterrupt()

	mainModule, err := context.getModule(moduleURI.String())
	if err != nil {
		panic("invalid state: main module not found")
	}
	mainModuleExports, ok := mainModule["exports"].(*goja.Object)
	if !ok {
		panic("invalid state: no exports")
	}

	render, ok := goja.AssertFunction(vm.ToValue(mainModuleExports.Get(viewName)))
	if !ok {
		render, ok = goja.AssertFunction(vm.ToValue(mainModule["default"]))
	}

	if !ok {
		return InternalError, fmt.Errorf("could not find function for view '%s' in %s'", viewName, viewModuleMeta.URL.String())
	}

	view, err := render(goja.Undefined(), vm.ToValue(context.Model))
	if err != nil {
		stack := "no stack"
		if exception, ok := err.(*goja.Exception); ok {
			stack = exception.String()
		}

		return "", WaxError{
			File:  *viewModuleMeta.URL,
			Stack: stack,
			Err:   err,
		}
	}

	toWrite := view.Export()
	context.Writer.WriteValue(toWrite)
	return (string)(context.Writer.Result(nil)), nil
}

type waxJSObj struct {
	engine *WaxEngine
	vm     *goja.Runtime
}

func (w *waxJSObj) Raw(v string) templateResult {
	return templateResult(v)
}

type renderFunc func(*waxWriter) (templateResult, error)

func (w *waxJSObj) Now(v any) (templateResult, error) {
	wr := w.engine.getNewWriter()
	_, e := wr.WriteValue(v)
	if e != nil {
		return "", e
	}
	return wr.Result(nil), nil
}

func (w *waxJSObj) Sub(v goja.Value) (renderFunc, error) {
	if _, ok := goja.AssertFunction(v); !ok {
		return nil, fmt.Errorf("expected to get arrow function")
	}

	var f renderFunc
	err := w.vm.ExportTo(v, &f)
	return f, err
}
