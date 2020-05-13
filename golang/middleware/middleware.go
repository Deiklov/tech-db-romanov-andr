package middleware

import (
	"github.com/valyala/fasthttp"
)

func SetJson(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		h(ctx)
		ctx.Response.Header.Set("Content-Type", "application/json")
	})
}
