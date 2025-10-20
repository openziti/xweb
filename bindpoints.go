package xweb

import (
	gotls "crypto/tls"
	"net"
	"net/http"

	"github.com/openziti/identity"
)

var BindPointListenerFactoryRegistry = make([]BindPointListenerFactory, 0)

type BindPointListenerFactory interface {
	New(map[interface{}]interface{}) (BindPoint, error)
}

type BindPoint interface {
	Listener(serverName string, tlsConfig *gotls.Config) (net.Listener, error) // a listener to be used with the http server
	BeforeHandler(next http.Handler) http.Handler                              // called before xweb handlers execute
	AfterHandler(prev http.Handler) http.Handler                               // called after xweb handlers complete
	Validate(identity.Identity) []error                                        // validates the BindPoint
	ServerAddress() string                                                     // the address the server
	Configure(config []interface{}) error                                      // configures the BindPoint using the provided map
}
