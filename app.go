package ap

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

type Handler[TApp any, TUser any] func(*Ctx[TApp, TUser])

type PanicHandler[TApp any, TUser any] func(*Ctx[TApp, TUser], interface{})
type WrapFunc[TApp any, TUser any] func(*Ctx[TApp, TUser])
type CurrentUserFunc[TApp any, TUser any] func(*Ctx[TApp, TUser]) TUser
type FuncsFunc[TApp any, TUser any] func(*Ctx[TApp, TUser]) template.FuncMap

type App[TApp any, TUser any] struct {
	mux          *http.ServeMux
	template     *template.Template
	wrap         WrapFunc[TApp, TUser]
	panicHandler PanicHandler[TApp, TUser]
	currentUser  CurrentUserFunc[TApp, TUser]
	funcs        FuncsFunc[TApp, TUser]
}

func New[TApp any, TUser any]() *App[TApp, TUser] {
	return &App[TApp, TUser]{mux: &http.ServeMux{}}
}

func (a *App[TApp, TUser]) Mux() *http.ServeMux {
	return a.mux
}

func (a *App[TApp, TUser]) Handle(pattern string, app TApp, handler Handler[TApp, TUser]) {
	a.mux.Handle(pattern, a.wrapHandler(app, handler))
}

func (a *App[TApp, TUser]) HandleHTTP(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

func (a *App[TApp, TUser]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *App[TApp, TUser]) Template() *template.Template {
	return a.template
}

func (a *App[TApp, TUser]) SetTemplate(t *template.Template) {
	a.template = t
}

func (a *App[TApp, TUser]) SetWrap(fn WrapFunc[TApp, TUser]) {
	a.wrap = fn
}

func (a *App[TApp, TUser]) SetPanicHandler(fn PanicHandler[TApp, TUser]) {
	a.panicHandler = fn
}

func (a *App[TApp, TUser]) SetCurrentUser(fn CurrentUserFunc[TApp, TUser]) {
	a.currentUser = fn
}

func (a *App[TApp, TUser]) SetFuncs(fn FuncsFunc[TApp, TUser]) {
	a.funcs = fn
}

func (a *App[TApp, TUser]) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, a)
}

func (a *App[TApp, TUser]) wrapHandler(app TApp, h Handler[TApp, TUser]) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := NewCtx(a, app, r, w)
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

func Funcs(c *Ctx[any, any]) template.FuncMap {
	return template.FuncMap{}
}
