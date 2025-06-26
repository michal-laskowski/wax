package wax

import (
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
)

type waxJSObj struct {
	engine  *Engine
	context *runContext
	vm      *goja.Runtime
	modules map[string]goja.Value
	obj     goja.Value
}

func newWaxObj(engine *Engine, vm *goja.Runtime, context *runContext) *waxJSObj {
	o := vm.NewObject()
	ret := &waxJSObj{
		engine:  engine,
		vm:      vm,
		modules: make(map[string]goja.Value, 3),
		obj:     o,
		context: context,
	}
	o.Set("Sub", vm.ToValue(ret.sub))
	o.Set("Raw", vm.ToValue(ret.raw))
	o.Set("Now", vm.ToValue(ret.now))
	o.Set("GetModule", vm.ToValue(ret.getModule))
	return ret
}

func (c *waxJSObj) raw(fc goja.FunctionCall) goja.Value {
	v := fc.Arguments[0].String()
	return c.vm.ToValue(templateResult(v))
}

func (c *waxJSObj) now(fc goja.FunctionCall) goja.Value {
	sb := new(strings.Builder)
	wr := newWriter(sb, c.vm)
	v := fc.Arguments[0]
	wr.process(v, c.vm)
	return c.vm.ToValue(templateResult(sb.String()))
}

func (c *waxJSObj) sub(fc goja.FunctionCall) goja.Value {
	return fc.Arguments[0]
}

func (c *waxJSObj) getModule(fc goja.FunctionCall) goja.Value {
	moduleName := fc.Arguments[0].String()
	module := c.GetModule(moduleName)
	return module
}

func (c *waxJSObj) GetModule(s string) goja.Value {
	if moduleExports, ok := c.modules[s]; ok {
		return moduleExports
	}
	return nil
}

func (c *waxJSObj) DefineModule(m *ModuleMeta) goja.Value {
	module := buildModuleObjStruct(m, c)
	c.modules[m.URL.String()] = module
	return module
}

func buildModuleObjStruct(m *ModuleMeta, c *waxJSObj) *goja.Object {
	module := map[string]any{
		"meta": map[string]any{
			"dirname":  filepath.ToSlash(filepath.Dir(m.URL.Path)),
			"filename": filepath.ToSlash(m.URL.Path),
			"url":      m.URL.String(),
			"main":     m.isMain,
		},
		"exports": c.vm.NewObject(),
		"do_import": func(arg goja.FunctionCall) goja.Value {
			v := arg.Arguments[0].String()

			p, err := c.engine.viewResolver.ResolveModuleFile(*m, v)
			if err != nil {
				c.vm.Interrupt(err)
				return nil
			}

			m, err := c.engine.load(c.context, c, p)
			if err != nil {
				c.vm.Interrupt(err)
				return nil
			}
			return m
		},
	}
	return c.vm.ToValue(module).(*goja.Object)
}
