Docs you can give to you agent:

```
ap = tiny Go web/app helper lib.

Core:
- web := ap.New(app)
- web.Handle(pattern, func(c *ap.Ctx[*App]) { ... })
- web.HandleHTTP(pattern, httpHandler)
- web.SetWrap(func(c *ap.Ctx[*App]) { ... })           // middleware-ish per request
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
- c.App      // project app instance passed to ap.New(app)
- c.Req/c.Res
- c.Status
- c.Params   // query + form + "id" path value
- c.Values   // request-scoped template values
- c.Done     // set when response already sent
- helpers: c.Json(), c.Path(), c.Param(), c.Header(), c.GetCookie(), c.SetCookie(), c.Redirect(), c.Text(), c.Render()

Utility types:
- ap.J map[string]any with Get/GetI/GetB/GetJ/Set
- ap.Values = map[string]interface{}

Reusable helpers:
- ap.Check / Checkm / Assert
- ap.Env / Or / UUID
- ap.StringToJSON / JSONToString
- ap.MarkdownToHTML / Truncate / TimeAgo / FormatDatetime
- ap.HTTPGet / HTTPPost
- ap.PrettyStack

Reusable services:
- ap.Database: NewDatabase, SetURL, SetDebug, Connect, Save, Where, Exec, First, Query
- ap.Cache: NewCache, Get, GetStale, Set, Delete, Ensure, EnsureStale
- ap.Storage: NewStorageLocal(dir), NewStorageS3(bucket, clientID, secretID, region)
- ap.Crypto: CreateToken, ValidateToken, SignHMAC256, SumSHA256, Base64 helpers

Pattern:
- put app-specific auth/current-user logic on your App, not in ap
- use web.SetWrap(...) to populate c.Values / redirects / auth checks
- use web.AddFuncs(...) for project template helpers
```
