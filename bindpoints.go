/*
	Copyright NetFoundry Inc.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package xweb

import (
	gotls "crypto/tls"
	"net"
	"net/http"

	"github.com/openziti/identity"
)

// BindPointListenerFactoryRegistry is used to add BindPointListenerFactory instances that are used to return
// new BindPoint instances
var BindPointListenerFactoryRegistry []BindPointListenerFactory

// BindPointListenerFactory is an interface that will generate new BindPoint instances based on configuration
type BindPointListenerFactory interface {
	New(map[interface{}]interface{}) (BindPoint, error)
}

// The BindPoint interface is used to provide necessary information to xweb. Primarily, it is used to provide
// listeners to the http server xweb controls.
type BindPoint interface {
	Listener(serverName string, tlsConfig *gotls.Config) (net.Listener, error) // a listener to be used with the http server
	BeforeHandler(next http.Handler) http.Handler                              // called before xweb handlers execute
	AfterHandler(prev http.Handler) http.Handler                               // called after xweb handlers complete
	Validate(identity.Identity) []error                                        // validates the BindPoint
	ServerAddress() string                                                     // the address the server
}
