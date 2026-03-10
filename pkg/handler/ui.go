package handler

import (
	"context"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/benlocal/cli-manager/app"
	chttp "github.com/benlocal/cli-manager/pkg/http"
	hertzApp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/route"
)

func ui(_ *chttp.RegistryContext, router *route.Engine) {
	uiFS := app.GetAppFS()
	fileServer := http.FileServer(http.FS(uiFS))
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		index, err := fs.ReadFile(uiFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}
	uiHandler := func(w http.ResponseWriter, r *http.Request) {
		servePath := path.Clean(r.URL.Path)
		servePath = strings.TrimPrefix(servePath, "/")
		if servePath == "" || servePath == "." {
			serveIndex(w, r)
			return
		}

		if info, err := fs.Stat(uiFS, servePath); err == nil {
			if info.IsDir() {
				serveIndex(w, r)
				return
			}
			r.URL.Path = "/" + servePath
			fileServer.ServeHTTP(w, r)
			return
		}
		serveIndex(w, r)
	}

	ui := adaptor.HertzHandler(http.HandlerFunc(uiHandler))
	router.GET("/*path", ui)
	router.HEAD("/*path", ui)

	// fallback for other 404s
	router.NoRoute(func(ctx context.Context, c *hertzApp.RequestContext) {
		c.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
		c.Response.SetBodyString(string("404 not found"))
	})
}
