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

type Ctx[TApp any] struct {
	App      TApp
	Req      *http.Request
	Res      http.ResponseWriter
	Status   int
	Params   map[string]string
	Values   map[string]interface{}
	Done     bool
	template *template.Template
	web      *App[TApp]
}

func NewCtx[TApp any](web *App[TApp], req *http.Request, res http.ResponseWriter) (*Ctx[TApp], error) {
	c := &Ctx[TApp]{App: web.app, Req: req, Res: res, web: web}
	c.Status = 200
	c.Params = map[string]string{}
	c.Values = map[string]interface{}{}

	t, err := web.template.Clone()
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
	c.Params["id"] = Or(c.Params["id"], c.Req.PathValue("id"))
	return c, nil
}

func (c *Ctx[TApp]) Json() J {
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

func (c *Ctx[TApp]) Path() string {
	return c.Req.URL.Path
}

func (c *Ctx[TApp]) Param(key, alt string) string {
	return Or(c.Params[key], alt)
}

func (c *Ctx[TApp]) Header(key, value string) {
	c.Res.Header().Set(key, value)
}

func (c *Ctx[TApp]) GetCookie(name string) string {
	if cookie, err := c.Req.Cookie(name); err == nil {
		return cookie.Value
	}
	return ""
}

func (c *Ctx[TApp]) SetCookie(name, value string) {
	http.SetCookie(c.Res, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   2147483647,
	})
}

func (c *Ctx[TApp]) Redirect(url string, args ...interface{}) {
	if len(args) > 0 {
		url = fmt.Sprintf(url, args...)
	}
	c.Res.Header().Set("Location", url)
	c.Res.WriteHeader(302)
	_, _ = c.Res.Write([]byte("Redirecting..."))
	c.Done = true
}

func (c *Ctx[TApp]) Text(text string) {
	c.Res.WriteHeader(c.Status)
	_, _ = c.Res.Write([]byte(text))
	c.Done = true
}

func (c *Ctx[TApp]) Render(name string, values Values) {
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

func (c *Ctx[TApp]) Funcs() template.FuncMap {
	if c.web == nil {
		return template.FuncMap{}
	}
	return c.web.Funcs(c)
}
