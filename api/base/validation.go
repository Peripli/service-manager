package base

import (
	"github.com/Peripli/service-manager/pkg/extension"
	"github.com/Peripli/service-manager/pkg/log"
)

func (c *Controller) validateCreateInterceptorProvidersNames(name string) {
	found := false
	for _, p := range c.CreateInterceptorProviders {
		if p.Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (c *Controller) validateCreateInterceptorProviders(newProviders []extension.CreateInterceptorProvider) {
	for _, newProvider := range newProviders {
		if ordered, ok := newProvider.(extension.Ordered); ok {
			positionAPI, nameAPI := ordered.PositionAPI()
			positionTx, nameTx := ordered.PositionTx()
			if positionAPI != extension.PositionNone {
				c.validateCreateInterceptorProvidersNames(nameAPI)
			}
			if positionTx != extension.PositionNone {
				c.validateCreateInterceptorProvidersNames(nameTx)
			}
		}
		for _, p := range c.CreateInterceptorProviders {
			if n, ok := p.(extension.Named); ok {
				if n.Name() == newProvider.Name() {
					log.D().Panicf("%s create interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (c *Controller) validateUpdateInterceptorProvidersNames(name string) {
	found := false
	for _, p := range c.UpdateInterceptorProviders {
		if p.Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (c *Controller) validateUpdateInterceptorProviders(newProviders []extension.UpdateInterceptorProvider) {
	for _, newProvider := range newProviders {
		if ordered, ok := newProvider.(extension.Ordered); ok {
			positionAPI, nameAPI := ordered.PositionAPI()
			positionTx, nameTx := ordered.PositionTx()
			if positionAPI != extension.PositionNone {
				c.validateUpdateInterceptorProvidersNames(nameAPI)
			}
			if positionTx != extension.PositionNone {
				c.validateUpdateInterceptorProvidersNames(nameTx)
			}
		}
		for _, p := range c.UpdateInterceptorProviders {
			if n, ok := p.(extension.Named); ok {
				if n.Name() == newProvider.Name() {
					log.D().Panicf("%s update interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (c *Controller) validateDeleteInterceptorProvidersNames(name string) {
	found := false
	for _, p := range c.DeleteInterceptorProviders {
		if p.Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (c *Controller) validateDeleteInterceptorProviders(newProviders []extension.DeleteInterceptorProvider) {
	for _, newProvider := range newProviders {
		if ordered, ok := newProvider.(extension.Ordered); ok {
			positionAPI, nameAPI := ordered.PositionAPI()
			positionTx, nameTx := ordered.PositionTx()
			if positionAPI != extension.PositionNone {
				c.validateDeleteInterceptorProvidersNames(nameAPI)
			}
			if positionTx != extension.PositionNone {
				c.validateDeleteInterceptorProvidersNames(nameTx)
			}
		}
		for _, p := range c.DeleteInterceptorProviders {
			if n, ok := p.(extension.Named); ok {
				if n.Name() == newProvider.Name() {
					log.D().Panicf("%s delete interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}
