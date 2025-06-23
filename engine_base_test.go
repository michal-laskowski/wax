package wax_test

import (
	"errors"
	"testing"
)

func Test_Engine_Base(t *testing.T) {
	baseTests := []TestSample{
		{
			name:        "engine_meta",
			description: "You can use module.meta",
			source: `export function View(){ 

                const keys = Object.keys(import.meta).sort().map(k => <><dt>{k}</dt><dd>{import.meta[k]}</dd></>)
                return <>
                    <dl>{keys}</dl>
                </>
            }`,
			expected: `
                <dl>
                  <dt>dirname</dt>
                  <dd>/</dd>
                  <dt>filename</dt>
                  <dd>/View.jsx</dd>
                  <dt>main</dt>
                  <dd></dd>
                  <dt>url</dt>
                  <dd>file:///View.jsx?ts=-dcbffeff2bc000</dd>
                </dl>
            `,
		},

		{
			name:        "call_view_function",
			description: "You must export your view from your JSX/TSX file",
			source:      "export function View(){ return <div>Hello</div> }",
			expected:    "<div>Hello</div>",
		},

		{
			name:        "call_view_fallback",
			description: "Test is requesting 'View' to bew called. Engine will fallback to default exported function if it will not find function named as requested view",
			source:      "export default function fallback(){ return <div>Hello</div> }",
			expected:    "<div>Hello</div>",
			/*TODO: make this a option */
		},

		{
			name:        "call_view_arrow_function",
			description: "Your view/component can be also an arrow unction ",
			source:      "export const View = () => { return <div>Hello</div> }",
			expected:    "<div>Hello</div>",
		},

		{
			name:        "call_component",
			description: "You can use other components just like in regular JSX/TSX",
			source: `
            export function View()
            { 
                return <div>
                <span>Main content</span>
                <OtherComponent/>
                </div>
            }
                
            export function OtherComponent()
            { 
                return <div>Hello from component</div>
            }`,
			expected: `
            <div>
                <span>Main content</span>
                <div>Hello from component</div>
            </div>`,
		},

		{
			name:        "use_model_gostruct",
			description: "You can pass Go struct as a model. We do not serialize them, so you can access only public members",
			source: `
            export function View(model) { 
                return <i>{model.Contact} - {model.Email} - {model.FullAddress()}</i> 
            }`,
			expected: `<i>John Doe - johndoe@example.com - Earth at Solar System</i>`,
			model: Contact{
				Contact: "John Doe",
				Email:   "johndoe@example.com",
				System:  "Solar System",
				Planet:  "Earth",
			},
		},
		{
			name:        "use_model_gomap",
			description: "You can pass Go map as a model",
			source: `
            export function View(model) { 
                return <i>{model.contact} - {model.email} - {model.fullAddress()}</i> 
            }`,
			expected: `<i>John Doe - johndoe@example.com - Earth at Solar System</i>`,
			model: map[string]any{
				"contact":     "John Doe",
				"email":       "johndoe@example.com",
				"fullAddress": func() string { return "Earth" + " at " + "Solar System" },
			},
		},
		{
			name:        "use_model_goanonymousstruct",
			description: "You can pass Go anonymous struct as a model",
			source: `
            export function View(model) { 
                return <i>{model.Contact} - {model.Email} - {model.FullAddress()}</i> 
            }`,
			expected: `<i>John Doe - johndoe@example.com - Earth at Solar System</i>`,
			model: struct {
				Contact string
				Email   string

				System      string
				Planet      string
				FullAddress func() string
			}{
				Contact:     "John Doe",
				Email:       "johndoe@example.com",
				System:      "Solar System",
				Planet:      "Earth",
				FullAddress: func() string { return "Earth" + " at " + "Solar System" },
			},
		},

		{
			name:        "we_autoescape_values",
			description: "We encode values",
			source:      "export function View(model){ return <div>{model.unsafe}</div> }",
			expected:    "<div>&lt;script&gt;alert(&#39;attack&#39;)&lt;/script&gt;</div>",
			model: map[string]any{
				"unsafe": "<script>alert('attack')</script>",
			},
		},
		{
			name:        "we_autoescape_but_you_can_decide_otherwise",
			description: "You can use wax.Raw to write value without encoding",
			source:      "export function View(model){ return <div>{wax.Raw(model.unsafe)}</div> }",
			expected:    "<div><script>alert('attack')</script></div>",
			model: map[string]any{
				"unsafe": "<script>alert('attack')</script>",
			},
		},
		{
			name:        "jsx_attribute_use_spread_operator",
			description: "yes you can",
			source: `
            export function View(model)
            { 
                return <div>
                    <DumpObject {...model} />
                    <DumpObject x-data={model.Contact} />
                    <DumpObject x-data={model.Contact} {...model} />
                    <DumpObject {...model} Contact={"foo"} />
                
                    <ContactView {...model} />
                    <ContactView {...model} Contact={"foo"}/>
                    <ContactView Contact={"foo"} {...model} />
                </div>
            }
            
            export const DumpObject  = (p) => <span>Keys: {Object.keys(p).sort().join(", ")}</span>
            export const ContactView = (p) => <span>{p.Contact}</span>`,
			model: Contact{
				Contact: "John Doe",
				Email:   "johndoe@example.com",
				System:  "Solar System",
				Planet:  "Earth",
			},
			expected: `
            <div>
                <span>Keys: Contact, Email, FullAddress, Planet, System</span>
                <span>Keys: x-data</span>
                <span>Keys: Contact, Email, FullAddress, Planet, System, x-data</span>
                <span>Keys: Contact, Email, FullAddress, Planet, System</span>
        
                <span>John Doe</span>
                <span>foo</span>
                <span>John Doe</span>
            </div>`,
		},

		{
			name:        "jsx_children",
			description: "You can use children property to render passed child content. It will be a safe-string",
			source: `
            export function View()
            { 
                return <div>
                    <span>Main content</span>
                    <WrappingComponent>
                        <span>I'm sexy</span>
                    </WrappingComponent>
                </div>
            }
                
            export function WrappingComponent({children})
            { 
                return <div>
                    {children}
                    <span>And I Know It</span>
                </div>
            }`,
			expected: `
            <div>
                <span>Main content</span>
                <div>
                    <span>I'm sexy</span>
                    <span>And I Know It</span>
                </div>
            </div>`,
		},

		{
			name:        "jsx_elements_are_not_objects_like_in_react",
			description: "JSX element are not JS objects like in 'react' or other JS frameworks. WAX is using it's internal rendering function as children",
			source: `
            export function View()
            {
                const asVar = Component({value: "in-var"})
                return <div>
                    <i>{<Component value="hello" />}</i>
                    <i>{typeof <Component value="hello" />}</i>
                    <i>{typeof asVar}</i>
                    <i>{asVar}</i>
                </div>
            }
                
            export const Component = (p) => <span>{p.value}</span>`,
			expected: `
            <div>
                <i><span>hello</span></i>
                <i>function</i>
                <i>function</i>
                <i><span>in-var</span></i>
            </div>`,
		},

		{
			name:        "es_module_import",
			description: "You can use ES import/export syntax to work with modules",
			source: `
            import { default as Top, SimpleDiv as JustImportedComponent } from "./components/base.jsx"

            export function View(){ 
                return <div>
                    <Top/>
                    <JustImportedComponent/>
                </div> 
            }`,
			modules: map[string]string{
				"components/base.jsx": `
                export function SimpleDiv() { 
                    return <div>simple div content</div>
                }

                export default function Welcome() { 
                    return <div>Hello</div>
                }`,
			},
			expected: `
                <div>
                    <div>Hello</div>
                    <div>simple div content</div>
                </div>`,
		},

		{
			name:        "children_is_called_on_render",
			description: "Components are rendered when needed",
			source: `

            var childCall = 0
            export function View(p)
            { 
                return <div>
                    <step-1>Child call {childCall}</step-1>
                    <Component/>
                    <step-2>Child call {childCall}</step-2>
                    {false ?  <Component/>: <i>some check</i>}
                    <step-3>Child call {childCall}</step-3>
                    <ErrorHandler
                        fallback={<FailingComponent/>}
                        children={<i>success</i>}
                    />
                    <step-4>Child call {childCall}</step-4>
                    <ErrorHandler
                        fallback={<i>error 1</i>}
                        children={<FailingComponent/>}
                    />
                    <step-5>Child call {childCall}</step-5>
                    <ErrorHandler
                        fallbackRender={e=> <i>error - {e?.message ?? e}</i>}
                        children={<Component/>}
                    />
                    <step-6>Child call {childCall}</step-6>
                    <ErrorHandler
                        fallbackRender={e=> <i>error - {e.message}</i>}
                        children={<>{p.GoErrorFunc()}</>}
                    />
                    <step-7>Child call {childCall}</step-7>
                </div>
            }
                
            function Component() {
                childCall++
                if (childCall > 1) throw 'exception from component'
                return <i>Component</i>
            } 

            function FailingComponent() {
                throw 'some exception form FailingComponent'
            } 

            function ErrorHandler(p: PropsWithChildren<{ fallback?: JSX.Child, fallbackRender?: (e)=> JSX.Child}>) {
                try {
                    // use wax.Now to render component to string
                    return wax.Now(p.children)
                } catch (e) {
                    if (p.fallbackRender){
                        return wax.Now(p.fallbackRender(e))
                    }
                    return wax.Now(p.fallback)
                }
            }
        `,
			model: map[string]any{
				"GoErrorFunc": func() error { return errors.New("some error from go") },
			},
			expected: `
            <div>
              <step-1>Child call 0</step-1>
              <i>Component</i>
              <step-2>Child call 1</step-2>
              <i>some check</i>
              <step-3>Child call 1</step-3>
              <i>success</i>
              <step-4>Child call 1</step-4>
              <i>error 1</i>
              <step-5>Child call 1</step-5>
              <i>error - exception from component</i>
              <step-6>Child call 2</step-6>
              <i>error - some error from go</i>
              <step-7>Child call 2</step-7>
            </div>`,
		},

		{
			name:        "engine_you_can_define_global_objects",
			description: "You can specify global objects while creating Engine. It will be in context for each engine call.",
			source: `export const View = () => { return <div>
                    <div>{customGlobal.stringValue}</div>
                    <div>{customGlobal.GoFunc()}</div>
                </div>
                }`,
			globalObjects: map[string]any{
				"customGlobal": map[string]any{
					"stringValue": "test string in global object",
					"GoErrorFunc": func() error { return errors.New("some error from go") },
					"GoFunc":      func() string { return "value from go func" },
				},
			},
			expected: `
                <div>
                    <div>test string in global object</div>
                    <div>value from go func</div>
                </div>`,
		},

		{ // TODO more on exceptions
			name:        "engine_will_get_error_on_exception",
			description: "You can specify global objects while creating Engine. It will be in context for each engine call.",
			source: `export const View = () => { 
                customGlobal.GoErrorFunc() //<--- throws exception
                return <div>
                    <div>{customGlobal.stringValue}</div>
                    <div>{customGlobal.GoFunc()}</div>
                </div>
                }`,
			errorPhase:   "execute",
			errorMessage: "GoError: some error from go at github.com/michal-laskowski/wax_test.Test_Engine_Base.func6 (native)",
			globalObjects: map[string]any{
				"customGlobal": map[string]any{
					"stringValue": "test string in global object",
					"GoErrorFunc": func() error { return errors.New("some error from go") },
					"GoFunc":      func() string { return "value from go func" },
				},
			},
			expected: `
                <div>
                    <div>test string in global object</div>
                    <div>value from go func</div>
                </div>`,
		},
		{ // TODO more on exceptions
			name:        "engine_will_get_error_on_exception_02",
			description: "You can specify global objects while creating Engine. It will be in context for each engine call.",
			source: `export const View = () => { 
                
                return <div>
                    <div>{customGlobal.stringValue}</div>
                    <div>{customGlobal.GoFunc()}</div>
                    <div>{customGlobal.GoErrorFunc() /*<--- throws exception*/}</div>
                </div>
                }`,
			errorPhase:   "execute",
			errorMessage: "GoError: some error from go at github.com/michal-laskowski/wax_test.Test_Engine_Base.func8 (native)",
			globalObjects: map[string]any{
				"customGlobal": map[string]any{
					"stringValue": "test string in global object",
					"GoErrorFunc": func() error { return errors.New("some error from go") },
					"GoFunc":      func() string { return "value from go func" },
				},
			},
			expected: `
                <div>
                    <div>test string in global object</div>
                    <div>value from go func</div>
                </div>`,
		},
	}

	runSamples(t, baseTests)
}
