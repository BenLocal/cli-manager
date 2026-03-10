package http

import "github.com/cloudwego/hertz/pkg/route"

type Registry struct {
	routeBindings []func(baseHandler *RegistryContext, router *route.Engine)
}

var DefaultRegistry *Registry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{
		routeBindings: []func(baseHandler *RegistryContext, router *route.Engine){},
	}
}

func (r *Registry) Add(binding func(baseHandler *RegistryContext, router *route.Engine)) {
	r.routeBindings = append(r.routeBindings, binding)
}

func (r *Registry) Bindings() []func(baseHandler *RegistryContext, router *route.Engine) {
	return r.routeBindings
}
