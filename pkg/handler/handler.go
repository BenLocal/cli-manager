package handler

import (
	chttp "github.com/benlocal/cli-manager/pkg/http"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

func init() {
	chttp.DefaultRegistry.Add(func(h *chttp.RegistryContext, router *route.Engine) {
		nodes(h, router)
		sessions(h, router)
		files(h, router)
		ws(h, router)
		ui(h, router)
	})
}

type errorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type successPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func writeError(c *app.RequestContext, status int, message string) {
	c.JSON(consts.StatusOK, errorPayload{
		Code:    status,
		Message: message,
	})
}

func writeSuccess(c *app.RequestContext, data any) {
	c.JSON(consts.StatusOK, successPayload{
		Code:    consts.StatusOK,
		Message: "success",
		Data:    data,
	})
}
