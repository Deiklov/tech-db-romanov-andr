package middleware

import (
	"github.com/valyala/fasthttp"
	"net/http"
)

func SetApplJson(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
func SetJson(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		h(ctx)
		ctx.Response.Header.Set("Content-Type", "application/json")
	})
}
