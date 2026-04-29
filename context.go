package nanoserve

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	// params map[string]string

	// handlers []HandlerFunction
	index    int

	contextData map[string] any
	statusCode  int
	
	// for aborting the Hook execution
	is_aborted bool
	
	matached_handler *RouteMatch
}

func NewContext(w http.ResponseWriter, r *http.Request, matched_handler *RouteMatch) *Context {
	return &Context{
		Writer:        w,
		Request:       r,
		statusCode:    http.StatusOK,
		matached_handler: matched_handler,
	}
}

func(c *Context) Abort(){
	c.is_aborted = true
}

func (c *Context) IsAborted() bool {
	return c.is_aborted
}

func (c *Context) Next() error {
	c.index++
	if c.index >= len(c.matached_handler.Middlewares) {
		return nil
	}
	return c.matached_handler.Middlewares[c.index](c)
}

func (c *Context) RunHandler() (bool, error) {
	if c.index == len(c.matached_handler.Middlewares) {
		if c.matached_handler.Handler == nil {
			return false, nil
		}
		return true, c.matached_handler.Handler(c)
	}
	return false, nil
}

func (c *Context) Status(code int) *Context {
	c.statusCode = code
	return c
}

func (c *Context) writeStatus() {
	if c.statusCode != 0 {
		c.Writer.WriteHeader(c.statusCode)
	}
}

func (c *Context) Set(key string, value any) {
	c.contextData[key] = value
}

func (c *Context) Get(key string) any {
	return c.contextData[key]
}

func (c *Context) Text(text string) error {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.writeStatus()
	_, err := c.Writer.Write([]byte(text))
	return err
}

func (c *Context) String(s string) error {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.writeStatus()
	_, err := c.Writer.Write([]byte(s))
	return err
}

func (c *Context) Url() *url.URL {
	return c.Request.URL
}

func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *Context) Param(key string) string {
	val := c.matached_handler.Params[key]
	if val != "" {
		return val
	}
	return ""
}

func (c *Context) Json(data any) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.writeStatus()
	return json.NewEncoder(c.Writer).Encode(data)
}
