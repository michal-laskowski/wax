# WAX – JSX-based Server-Side Rendering for Go

WAX is a Go library for server-side rendering (SSR) of JSX/TSX components, designed to provide a seamless, dynamic view layer without the need to regenerate templates after code changes.

It allows developers to generate dynamic HTML views using JSX syntax directly in Go.

Views are rendered on-the-fly, ensuring fast development cycles and simplified deployments.

WAX dynamically compiles and renders views at runtime, eliminating the need to manually regenerate or precompile templates.
This allows for a faster development workflow and easier maintenance.

With WAX separate Node.js/Deno/Bun process or JavaScript runtime on the server is not required.

Key Features:\
✅ Server-side rendering of JSX – Render JSX/TSX views directly in Go.\
🔄 Hot reload for views – Automatically refresh changes without restarting the server.\
✅ TypeScript model generation – Generate TypeScript typings from Go structs for type safety.\
✅ Seamless integration – Works with net/http, Echo, and other Go web frameworks.\
🚀 Single-file deployment – Bundle JSX views into a Go binary using embed.FS.\

You can use JSX Features that are commonly used in JS/TS SSR world.
WAX enables Go developers to leverage JSX for pre-rendering HTML on the server, taking advantage of:\
👉 Declarative UI rendering – Structure HTML using JSX syntax instead of templates.\
👉 Component-based views – Organize reusable server-rendered JSX components.\
👉 Props passing – Dynamically inject data into components before rendering.\
👉 Conditional rendering – Control visibility of elements using JavaScript expressions.\
👉 List rendering – Generate dynamic lists using .map() before sending HTML to the client.\
👉 Static site generation (SSG) – Pre-render content for fast page loads.\
👉 Module imports – Import and reuse JavaScript/TypeScript modules inside JSX views.\

With WAX, you get the power of JSX-based rendering in Go, making it easier to generate dynamic, SEO-friendly HTML while keeping your backend architecture simple and efficient.

🫶 Hypermedia & JavaScript Ecosystem\
WAX is designed to work seamlessly with hypermedia-driven frameworks like HTMX and Alpine.js.

This enables progressive enhancement, where HTML responses dynamically update parts of the UI without requiring a full-page reload.

## Getting started

### Installation

```shell
go get github.com/michal-laskowski/wax
```

### First usage - with ```net/http```

Grab it from [examples repository](https://github.com/michal-laskowski/wax-samples/tree/master/http-std)

or DIY:

#### Setup project

```shell
mkdir playground-wax-first-usage
cd playground-wax-first-usage
go mod init my-plyground/wax-first-usage
go get github.com/michal-laskowski/wax

```

#### Create view

Let's create ``` views/hello.tsx ``` file with exported view function.

```tsx title="views/hello.tsx"
export function Hello(name: string) {
    return <div>Hello, {name}</div>
}
```

#### Create server

Create server in the ``` cmd/main.go ``` file

```go title="cmd/main.go"
package main

import (
  "fmt"
  "net/http"
  "os"

  "github.com/michal-laskowski/wax"
)

func main() {
  renderer := wax.New(wax.NewFsViewResolver(os.DirFS("./views/")))

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
   renderer.Render(w, "Hello", "John")
  })

  fmt.Println("Listening on http://localhost:3000")
  http.ListenAndServe(":3000", nil)
}
```

#### That's it. Let's rock 🤘

Start server

```shell
go run cmd/main.go
```

Now open your favorite browser.

#### View hot reload

In this example view code is loaded from file system.
The engine will cache the view code, but when you change view file, it will hot reload it without the need to restarting the server.

Try this ... modify view, save and refresh page.

## Configure TypeScript

You can use [WAX-JSX](https://github.com/michal-laskowski/wax-jsx) to configure JSX module used by TypeScript language service.

Install package:

```shell
npm install github:michal-laskowski/wax-jsx
```

In your views folder, create a tsconfig.json file:

```json title="views/tsconfig.json"
{
    "compilerOptions": {
        "jsx": "react-jsx",
        "jsxImportSource": "wax-jsx",
        "moduleResolution": "Bundler",
        "target": "ES2018",
        "noEmit": true,
        "allowImportingTsExtensions": true,
        "baseUrl": ".",
        "types": [
            "wax-jsx"
        ],
    },
    "exclude": [
        "node_modules"
    ],
    "include": [
        "*"
    ]
}
```

See [WAX-JSX](https://github.com/michal-laskowski/wax-jsx)  for more details.

## Go to TS - generate typings (.d.ts) from Go structs

You can pass Go struct, map or any other Go type as view model.

Having type definitions for your models in TSX/JSX improves type safety and auto-completion, making the connection between Go code and JSX-based view templates more seamless.

We've got you covered with [GoTS](https://github.com/michal-laskowski/wax-libs/tree/master/gots).\
It allows you to generate TypeScript typings directly from your Go types.

## View live reload

Editing, saving, and seeing the results is easy with [LiveReload](https://github.com/michal-laskowski/wax-libs/tree/master/livereload) module.

In ```First usage example``` you could check view hot reloading. With live reloading, you can stay in your favorite code editor.

## Live reloading for your server

We do not provide live reloading for Go applications.
You might check [wgo](https://github.com/bokwoon95/wgo) or [Air](https://github.com/air-verse/air).

## Single file application deployment

For production you can pass ```embed.FS``` to view file provider.
This way, your application can be built into a single file without needing to copy views to the server.

## More examples

You can check out the [Echo example](https://github.com/michal-laskowski/wax-samples/tree/master/http-echo), which covers all the above aspects of DX.

It:

- It uses [Labstack Echo v4](https://github.com/labstack/echo) as a web framework.
- shows how you can implement 'DEV' mode – views from os.FS using [live-reload](https://github.com/michal-laskowski/wax-libs/tree/master/livereload)
- shows how you can implement 'PROD' mode – views from embed.FS with live reloading disabled
- shows how you can use [GoTS](https://github.com/michal-laskowski/wax-libs/tree/master/gots) to generate type definitions for TypeScript from a Go model

## Usage

### View resolving

WAX uses a ViewResolver to locate view files and module content.

You can utilize the built-in resolver by calling ```NewFsViewResolver```.

FsViewResolver searches for a view file with the same name as the requested view to render. It looks for files with the ```.tsx``` or ```.jsx``` extensions.

### Module imports

WAX supports ESM export and import.

```javascript
import defaultExport from "./module-name.tsx";
import * as name from "./module-name.tsx";
import { export1 } from "./module-name.tsx";
import { export1 as alias1 } from "./module-name.tsx";
import { default as alias } from "./module-name.tsx";
import { export1, export2 } from "./module-name.tsx";
import { export1, export2 as alias2, /* … */ } from "./module-name.tsx";
import { "string name" as alias } from "./module-name.tsx";
import defaultExport, { export1, /* … */ } from "./module-name.tsx";
import defaultExport, * as name from "./module-name.tsx";
import "./module-name.tsx";
```
