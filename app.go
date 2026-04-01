package ap

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

type Handler[TApp any] func(*Ctx[TApp])

type PanicHandler[TApp any] func(*Ctx[TApp], interface{})
type WrapFunc[TApp any] func(*Ctx[TApp])
type FuncsFunc[TApp any] func(*Ctx[TApp]) template.FuncMap

type App[TApp any] struct {
	app          TApp
	mux          *http.ServeMux
	template     *template.Template
	wrap         WrapFunc[TApp]
	panicHandler PanicHandler[TApp]
	funcs        []FuncsFunc[TApp]
}

func New[TApp any](app TApp) *App[TApp] {
	a := &App[TApp]{app: app, mux: &http.ServeMux{}}
	a.panicHandler = a.defaultPanicHandler
	return a
}

func (a *App[TApp]) defaultPanicHandler(c *Ctx[TApp], err interface{}) {
	c.Render("error", map[string]interface{}{"error": err})
}

func (a *App[TApp]) Mux() *http.ServeMux {
	return a.mux
}

func (a *App[TApp]) Handle(pattern string, handler Handler[TApp]) {
	a.mux.Handle(pattern, a.wrapHandler(handler))
}

func (a *App[TApp]) HandleHTTP(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

func (a *App[TApp]) HandleAssets(assetsFS http.FileSystem) {
	a.mux.Handle("/assets/", http.FileServer(assetsFS))
}

func (a *App[TApp]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *App[TApp]) Template() *template.Template {
	return a.template
}

func (a *App[TApp]) LoadTemplates(fs fs.FS) {
	t, err := template.New("").Funcs(a.Funcs(nil)).ParseFS(fs, "templates/*.html")
	Check(err)
	a.template = t
}

func (a *App[TApp]) SetTemplate(t *template.Template) {
	a.template = t
}

func (a *App[TApp]) SetWrap(fn WrapFunc[TApp]) {
	a.wrap = fn
}

func (a *App[TApp]) SetPanicHandler(fn PanicHandler[TApp]) {
	a.panicHandler = fn
}

func (a *App[TApp]) AddFuncs(fn FuncsFunc[TApp]) {
	a.funcs = append(a.funcs, fn)
}

func (a *App[TApp]) SetFuncs(fn FuncsFunc[TApp]) {
	if fn == nil {
		a.funcs = nil
		return
	}
	a.funcs = []FuncsFunc[TApp]{fn}
}

func (a *App[TApp]) Funcs(c *Ctx[TApp]) template.FuncMap {
	funcs := template.FuncMap{}
	for _, fn := range a.funcs {
		if fn == nil {
			continue
		}
		for key, value := range fn(c) {
			funcs[key] = value
		}
	}
	return funcs
}

func (a *App[TApp]) TemplateFuncs() template.FuncMap {
	return a.Funcs(nil)
}

func (a *App[TApp]) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, a)
}

func (a *App[TApp]) wrapHandler(h Handler[TApp]) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := NewCtx(a, r, w)
		if err != nil {
			panic(err)
		}
		start := time.Now()
		defer func() {
			log.Printf("request %s %s %v\n", r.Method, r.URL.Path, time.Since(start))
			if rec := recover(); rec != nil {
				if a.panicHandler != nil {
					defer func() {
						if rec2 := recover(); rec2 != nil {
							ctx.Text(fmt.Sprintf("panic: %v", rec2))
						}
					}()
					log.Printf("server panic: %v\n%s\n", rec, PrettyStack())
					ctx.Status = 500
					a.panicHandler(ctx, rec)
					return
				}
				panic(rec)
			}
		}()
		if a.wrap != nil {
			a.wrap(ctx)
			if ctx.Done {
				return
			}
		}
		h(ctx)
	})
}

func PrettyStack() string {
	stackLines := strings.Split(string(debug.Stack()), "\n")
	out := ""
	filenameRegexp := regexp.MustCompile("([a-zA-Z0-9_]+/[a-zA-Z0-9_]+.go:[0-9]+) ")
	for i := 9; i < len(stackLines)-1; i += 2 {
		out += filenameRegexp.FindString(stackLines[i+1])
		out += strings.Split(stackLines[i], "0x")[0] + ")\n"
	}
	return out
}
