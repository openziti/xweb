package xweb

import (
	gotls "crypto/tls"
	"errors"
	"net"
	"net/http"

	"github.com/michaelquigley/pfxlog"
)

type BindPoint interface {
	Listener(serverName string, tlsConfig *gotls.Config) (net.Listener, error) // a listener to be used with the http server
	BeforeHandler(next http.Handler) http.Handler                              // called before xweb handlers execute
	AfterHandler(prev http.Handler) http.Handler                               // called after xweb handlers complete
	Validate() error                                                           //validates the BindPoint
	ServerAddress() string                                                     //the address the server
	Configure(config []interface{}) error                                      // configures the BindPoint using the provided map
}

var BindPointFactories = &BindPointFactoryRegistry{}

type BindPointFactory interface {
	Binding() string
	FactoryForConfig(config []interface{}) bool
	New(config []interface{}) (BindPoint, error)
}

type BindPointFactoryRegistry struct {
	factories []BindPointFactory
}

func (registry *BindPointFactoryRegistry) Register(bpf BindPointFactory) error {
	for _, f := range registry.factories {
		if f.Binding() == bpf.Binding() {
			pfxlog.Logger().Warnf("ignore bindpoint factory already registered: %s", bpf.Binding())
			return nil // errors.New(bpf.Binding() + " already registered")
		}
	}
	registry.factories = append(registry.factories, bpf)
	return nil
}

func (registry *BindPointFactoryRegistry) FindFactory(config []interface{}) (BindPointFactory, error) {
	for _, f := range registry.factories {
		if f.FactoryForConfig(config) {
			return f, nil
		}
	}
	return nil, errors.New("BindPointFactoryRegistry.FindFactory not implemented yet")
}
