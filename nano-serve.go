package nanoserve

import (
	"net/http"
)

type HandlerFunction func(*Context) error
type ErrorHandlerFunc func(*Context, error)

type Hooks struct {
	OnRequest  []HandlerFunction
	PreHandler []HandlerFunction
}

type HookType string

const (
	OnRequest  HookType = "OnRequest"
	PreHandler HookType = "PreHandler"
	OnError    HookType = "OnError"
)

type NanoServe struct {
	router              Router
	ErrorHandler        ErrorHandlerFunc
	Hooks               Hooks
	is_on_req_hook      bool
	is_pre_handler_hook bool
	is_on_error_hook    bool
}

func New() *NanoServe {
	return &NanoServe{
		router: NewTrieRouter(),
		ErrorHandler: func(c *Context, err error) {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		},
	}
}

func (n *NanoServe) GET(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodGet, path, h...)
}

func (n *NanoServe) POST(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodPost, path, h...)
}

func (n *NanoServe) PUT(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodPut, path, h...)
}

func (n *NanoServe) PATCH(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodPatch, path, h...)
}

func (n *NanoServe) DELETE(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodDelete, path, h...)
}

func (n *NanoServe) HEAD(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodHead, path, h...)
}

func (n *NanoServe) OPTIONS(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodOptions, path, h...)
}

func (n *NanoServe) CONNECT(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodConnect, path, h...)
}

func (n *NanoServe) TRACE(path string, h ...HandlerFunction) {
	n.addRoute(http.MethodTrace, path, h...)
}

func (n *NanoServe) Handle(method, path string, h ...HandlerFunction) {
	n.addRoute(method, path, h...)
}

func (n *NanoServe) AddHooks(hook_type HookType, hooks ...HandlerFunction) {
	switch hook_type {
	case "OnRequest":
		n.Hooks.OnRequest = append(n.Hooks.OnRequest, hooks...)
	case "PreHandler":
		n.Hooks.PreHandler = append(n.Hooks.PreHandler, hooks...)
	}
}	

// for serving static files
func (n *NanoServe) Static(urlPrefix string, rootDir string) {
	fs := http.FileServer(http.Dir(rootDir))

	handler := func(ctx *Context) error {
		http.StripPrefix(urlPrefix, fs).ServeHTTP(ctx.Writer, ctx.Request)
		return nil
	}

	n.GET(urlPrefix+"/*", handler)
}

func (n *NanoServe) addRoute(method string, path string, handlers ...HandlerFunction) {
	if len(handlers) == 0 {
		panic("route must have at least one handler")
	}

	middlewareFunctions := handlers[:len(handlers)-1]
	if len(middlewareFunctions) > 0 {
		n.router.AddMiddleware(path, middlewareFunctions...)
	}

	handler := handlers[len(handlers)-1]
	n.router.Insert(method, path, handler)
}

func (n *NanoServe) Run(addr string) error {
	// Check if any hooks are defined and set the corresponding flags
	n.is_on_req_hook = len(n.Hooks.OnRequest) > 0
	n.is_pre_handler_hook = len(n.Hooks.PreHandler) > 0
	// n.is_on_error_hook = len(n.Hooks.OnError) > 0

	return http.ListenAndServe(addr, n)
}

func (n *NanoServe) Use(pathOrHandler any, handlers ...HandlerFunction) {
	switch v := pathOrHandler.(type) {
	case string:
		n.router.AddMiddleware(v, handlers...)
	case HandlerFunction:
		all := append([]HandlerFunction{v}, handlers...)
		n.router.AddMiddleware("/", all...)
	case func(*Context) error:
		all := append([]HandlerFunction{v}, handlers...)
		n.router.AddMiddleware("/", all...)
	}
}

// Our Main Handler which will handle the incoming request
func (n *NanoServe) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	match := n.router.Search(r.Method, r.URL.Path)
	c := &Context{
		Writer:   w,
		Request:  r,
		handlers: match.Handler,
		index:    0,
		params:   match.Params,
	}

	// on request hooks execution
	if n.is_on_req_hook {
		for _, hook := range n.Hooks.OnRequest {
			if err := hook(c); err != nil {
				n.ErrorHandler(c, err)
				return
			}
			// if user aborted, return early
			if c.IsAborted() {
				return
			}
		}
	}

	// pre handler hook ( no matter if the handler exist or not , this must be executed )
	if n.is_pre_handler_hook {
		for _, hook := range n.Hooks.PreHandler {
			if err := hook(c); err != nil {
				n.ErrorHandler(c, err)
				return
			}
			if c.IsAborted() {
				return
			}
		}
	}
	
	// handler execution with ( if middlewares )
	if len(c.handlers) > 0 {
		if err := c.handlers[0](c); err != nil {
			n.ErrorHandler(c, err)
		}
		return
	}
	http.NotFound(w, r)
}