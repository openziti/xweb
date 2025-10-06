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
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// BindPointConfig represents the interface:port address of where a http.Server should listen for a ServerConfig and the public
// address that should be used to address it.
type BindPointConfig struct {
	InterfaceAddress string //<interface>:<port>
	Address          string //<ip/host>:<port>
	NewAddress       string //<ip/host>:<port> sent out as a header for clients to alternatively swap to (ip -> hostname moves)
	Identity         IdentityConfig
}

// IdentityConfig represents the BindPointConfig when an identity is supplied as opposed to an address
type IdentityConfig struct {
	Identity       []byte //an openziti identity
	Service        string //name of the service to bind
	ClientAuthType tls.ClientAuthType
	ServeTLS       bool
}

// Parse the configuration map for a BindPointConfig.
func (bindPoint *BindPointConfig) Parse(config map[interface{}]interface{}) error {
	if identityVal, ok := config["identity"]; ok {
		identCfg := identityVal.(map[interface{}]interface{})
		if fileVal, ok := identCfg["file"]; ok {
			if file, ok := fileVal.(string); ok {
				var err error
				bindPoint.Identity.Identity, err = os.ReadFile(file)
				if err != nil {
					return err
				}
			}
		}
		if envValCfg, ok := identCfg["env"]; ok {
			b64Id := os.Getenv(envValCfg.(string))
			idReader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64Id))
			var err error
			bindPoint.Identity.Identity, err = io.ReadAll(idReader)
			if err != nil {
				return err
			}
		}
		if len(bindPoint.Identity.Identity) < 1 {
			return errors.New("no identity configured. file or env must be supplied when using an identity binding")
		}
		if serviceVal, ok := identCfg["service"]; ok {
			if service, ok := serviceVal.(string); ok {
				bindPoint.Identity.Service = service
			}
		} else {
			return errors.New("service must be supplied when using an identity binding")
		}
		if certRequired, ok := identCfg["tlsClientAuthenticationPolicy"].(string); ok {
			switch strings.ToLower(certRequired) {
			case "noclientcert":
				bindPoint.Identity.ClientAuthType = tls.NoClientCert
			case "requestclientcert":
				bindPoint.Identity.ClientAuthType = tls.RequestClientCert
			case "requireanyclientcert":
				bindPoint.Identity.ClientAuthType = tls.RequireAnyClientCert
			case "verifyclientcertifgiven":
				bindPoint.Identity.ClientAuthType = tls.VerifyClientCertIfGiven
			case "requireandverifyclientcert":
				bindPoint.Identity.ClientAuthType = tls.RequireAndVerifyClientCert
			default:
				bindPoint.Identity.ClientAuthType = tls.VerifyClientCertIfGiven
			}
		}
		if serveTls, ok := identCfg["serveTLS"].(bool); ok {
			bindPoint.Identity.ServeTLS = serveTls
		} else {
			bindPoint.Identity.ServeTLS = true // default to true if not supplied
		}
	}

	if interfaceVal, ok := config["interface"]; ok {
		if address, ok := interfaceVal.(string); ok {
			bindPoint.InterfaceAddress = address
		} else {
			return fmt.Errorf("could not use value for address, not a string")
		}
	}

	if interfaceVal, ok := config["address"]; ok {
		if address, ok := interfaceVal.(string); ok {
			bindPoint.Address = address
		} else {
			return errors.New("could not use value for address, not a string")
		}
	}

	if interfaceVal, ok := config["newAddress"]; ok {
		if address, ok := interfaceVal.(string); ok {
			bindPoint.NewAddress = address
		} else {
			return errors.New("could not use value for newAddress, not a string")
		}
	}

	return nil
}

// Validate this configuration object.
func (bindPoint *BindPointConfig) Validate() error {
	idCfg := bindPoint.Identity
	if idCfg.Identity == nil { //validate underlay settings
		// required
		if err := validateHostPort(bindPoint.InterfaceAddress); err != nil {
			return fmt.Errorf("invalid interface address [%s]: %v", bindPoint.InterfaceAddress, err)
		}

		// required
		if err := validateHostPort(bindPoint.Address); err != nil {
			return fmt.Errorf("invalid advertise address [%s]: %v", bindPoint.Address, err)
		}

		//optional
		if bindPoint.NewAddress != "" {
			if err := validateHostPort(bindPoint.NewAddress); err != nil {
				return fmt.Errorf("invalid new address [%s]: %v", bindPoint.NewAddress, err)
			}
		}
	}

	return nil
}

func validateHostPort(address string) error {
	address = strings.TrimSpace(address)

	if address == "" {
		return errors.New("must not be an empty string or unspecified")
	}

	host, port, err := net.SplitHostPort(address)

	if err != nil {
		return errors.Errorf("could not split host and port: %v", err)
	}

	if host == "" {
		return errors.New("host must be specified")
	}

	if port == "" {
		return errors.New("port must be specified")
	}

	if port, err := strconv.ParseInt(port, 10, 32); err != nil {
		return errors.New("invalid port, must be a integer")
	} else if port < 1 || port > 65535 {
		return errors.New("invalid port, must 1-65535")
	}

	return nil
}
