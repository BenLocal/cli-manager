package handler

import (
	chttp "github.com/benlocal/cli-manager/pkg/http"
	"github.com/cloudwego/hertz/pkg/route"
)

func init() {
	chttp.DefaultRegistry.Add(func(h *chttp.RegistryContext, router *route.Engine) {
		nodes(h, router)
		ui(h, router)
		ws(h, router)
	})
}
