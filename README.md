Docs you can give to you agent:

```
ap = tiny Go web/app helper lib.

Core:
- web := ap.New(app)
- web.Handle(pattern, func(c *ap.Ctx[*App]) { ... })
- web.HandleHTTP(pattern, httpHandler)
- web.SetWrap(func(c *ap.Ctx[*App]) { ... }) // middleware-ish per request
- web.SetPanicHandler(func(c *ap.Ctx[*App], err any) { ... })
- web.AddFuncs(func(c *ap.Ctx[*App]) template.FuncMap { ... })
- web.SetTemplate(tmpl)
- web.ListenAndServe(addr)

Templates:
- parse with template.New("").Funcs(web.TemplateFuncs()).ParseFS(...)
- per-request funcs come from AddFuncs/SetFuncs
- c.Render("name", values) renders "name.html"
- c.Values is shared request-scoped render data merged into Render values

Ctx:
- c.App // project app instance passed to ap.New(app)
- c.Req/c.Res
- c.Status
- c.Params // query + form + "id" path value
- c.Values // request-scoped template values
- c.Done // set when response already sent
- req helpers: c.Path, c.Param, c.GetCookie
- res helpers: c.Header, c.SetCookie, c.Text, c.Json, c.Render, c.Redirect

Utilities:
- type ap.J map[string]any with Get/GetI/GetB/GetJ/Set
- ap.Check / Checkm / Assert
- ap.Env / Or / UUID
- ap.StringToJSON / JSONToString
- ap.MarkdownToHTML / Truncate / TimeAgo / FormatDatetime
- ap.HTTPGet / HTTPPost
- ap.Crypto: CreateToken, ValidateToken, SignHMAC256, SumSHA256, Base64 helpers

Reusable services:
- ap.Database: NewDatabase, Connect, Save, Where, Exec, First, Query
- ap.Cache: NewCache, Get, GetStale, Set, Delete, Ensure, EnsureStale
- ap.Storage: NewStorage{Local,S3}, Read, Write

Pattern:
- make app wide helpers take App, request helpers take Ctx 
- use web.SetWrap to populate c.Values / auth checks / redirects
- use web.AddFuncs(...) for project template helpers
```
