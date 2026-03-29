package ap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
)

type Values = map[string]interface{}

type Ctx[TApp any, TUser any] struct {
	App         TApp
	Req         *http.Request
	Res         http.ResponseWriter
	Status      int
	Params      map[string]string
	Values      map[string]interface{}
	Done        bool
	template    *template.Template
	currentUser TUser
	userLoaded  bool
	framework   *App[TApp, TUser]
}

func NewCtx[TApp any, TUser any](framework *App[TApp, TUser], app TApp, req *http.Request, res http.ResponseWriter) (*Ctx[TApp, TUser], error) {
	c := &Ctx[TApp, TUser]{App: app, Req: req, Res: res, framework: framework}
	c.Status = 200
	c.Params = map[string]string{}
	c.Values = map[string]interface{}{}

	t, err := framework.template.Clone()
	if err != nil {
		return nil, err
	}
	c.template = t.Funcs(c.Funcs())

	if err := c.Req.ParseForm(); err != nil {
		return nil, err
	}
	for k := range c.Req.URL.Query() {
		c.Params[k] = c.Req.URL.Query().Get(k)
	}
	for k := range c.Req.Form {
		c.Params[k] = c.Req.Form.Get(k)
	}
	c.Params["id"] = or(c.Params["id"], c.Req.PathValue("id"))
	return c, nil
}

func (c *Ctx[TApp, TUser]) Json() J {
	body, err := io.ReadAll(c.Req.Body)
	if err != nil {
		panic(err)
	}
	payload := J{}
	if err := json.Unmarshal(body, &payload); err != nil {
		panic(err)
	}
	return payload
}

func (c *Ctx[TApp, TUser]) Path() string {
	return c.Req.URL.Path
}

func (c *Ctx[TApp, TUser]) Param(key, alt string) string {
	return or(c.Params[key], alt)
}

func (c *Ctx[TApp, TUser]) Header(key, value string) {
	c.Res.Header().Set(key, value)
}

func (c *Ctx[TApp, TUser]) GetCookie(name string) string {
	if cookie, err := c.Req.Cookie(name); err == nil {
		return cookie.Value
	}
	return ""
}

func (c *Ctx[TApp, TUser]) SetCookie(name, value string) {
	http.SetCookie(c.Res, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   2147483647,
	})
}

func (c *Ctx[TApp, TUser]) CurrentUser() TUser {
	if c.framework.currentUser == nil {
		var zero TUser
		return zero
	}
	if c.userLoaded {
		return c.currentUser
	}
	c.currentUser = c.framework.currentUser(c)
	c.userLoaded = true
	return c.currentUser
}

func (c *Ctx[TApp, TUser]) Redirect(url string, args ...interface{}) {
	if len(args) > 0 {
		url = fmt.Sprintf(url, args...)
	}
	c.Res.Header().Set("Location", url)
	c.Res.WriteHeader(302)
	_, _ = c.Res.Write([]byte("Redirecting..."))
	c.Done = true
}

func (c *Ctx[TApp, TUser]) Text(text string) {
	c.Res.WriteHeader(c.Status)
	_, _ = c.Res.Write([]byte(text))
	c.Done = true
}

func (c *Ctx[TApp, TUser]) Render(name string, values Values) {
	if values == nil {
		values = Values{}
	}
	for k, v := range c.Values {
		if _, ok := values[k]; !ok {
			values[k] = v
		}
	}

	b := bytes.NewBuffer(nil)
	if err := c.template.ExecuteTemplate(b, name+".html", values); err != nil {
		panic(err)
	}
	c.Res.Header().Set("Content-Type", "text/html")
	c.Res.WriteHeader(c.Status)
	_, _ = c.Res.Write(b.Bytes())
	c.Done = true
}

func (c *Ctx[TApp, TUser]) Funcs() template.FuncMap {
	funcs := template.FuncMap{}
	if c.framework == nil || c.framework.funcs == nil {
		return funcs
	}
	for key, value := range c.framework.funcs(c) {
		funcs[key] = value
	}
	return funcs
}
