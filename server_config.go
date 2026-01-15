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
	"fmt"

	"github.com/michaelquigley/pfxlog"
	"github.com/openziti/identity"
	"github.com/pkg/errors"
)

// ServerConfig is the configuration that will eventually be used to create a xweb.Server (which in turn houses all
// the components necessary to run multiple http.Server's).
type ServerConfig struct {
	DefaultHttpHandlerProviderImpl
	Name       string
	APIs       []*ApiConfig
	BindPoints []BindPoint
	Options    ServerConfigOptions

	DefaultIdentity identity.Identity
	Identity        identity.Identity
}

// Parse parses a configuration map to set all relevant ServerConfig values.
func (config *ServerConfig) Parse(configMap map[interface{}]interface{}, pathContext string) error {
	//parse name, required, string
	if nameInterface, ok := configMap["name"]; ok {
		if name, ok := nameInterface.(string); ok {
			config.Name = name
		} else {
			return errors.New("name is required to be a string")
		}
	} else {
		return errors.New("name is required")
	}

	//parse apis, require 1, objet, defer
	if apiInterface, ok := configMap["apis"]; ok {
		if apiArrayInterfaces, ok := apiInterface.([]interface{}); ok {
			for i, apiInterface := range apiArrayInterfaces {
				if apiMap, ok := apiInterface.(map[interface{}]interface{}); ok {
					api := &ApiConfig{}
					if err := api.Parse(apiMap); err != nil {
						return fmt.Errorf("error parsing api configuration at index [%d]: %v", i, err)
					}

					config.APIs = append(config.APIs, api)
				} else {
					return fmt.Errorf("error parsing api configuration at index [%d]: not a map", i)
				}
			}
		} else {
			return errors.New("api section must be an array")
		}
	} else {
		return errors.New("apis section is required")
	}

	//parse bindPoints
	if bindPointArrVal, ok := configMap["bindPoints"]; ok {
		if bindPointArr, ok := bindPointArrVal.([]interface{}); ok {
			for i, bp := range bindPointArr {
				if bpMap, ok := bp.(map[interface{}]interface{}); ok {
					if len(BindPointListenerFactoryRegistry) == 0 {
						return fmt.Errorf("cannot configure bindPoints, no BindPointFactory Registered")
					}
					for _, bpf := range BindPointListenerFactoryRegistry {
						if b, bpe := bpf.New(bpMap); bpe != nil {
							return errors.Wrapf(bpe, "error parsing bindPoint configuration at index [%d]", i)
						} else {
							config.BindPoints = append(config.BindPoints, b)
						}
					}
				} else {
					return fmt.Errorf("error parsing bindPoint configuration at index [%d]: not a map", i)
				}
			}
		} else {
			return errors.New("bindPoints must be an array")
		}
	} else {
		return errors.New("bindPoints is required")
	}

	//parse identity
	if identityInterface, ok := configMap["identity"]; ok {
		if identityMap, ok := identityInterface.(map[interface{}]interface{}); ok {
			if identityConfig, err := parseIdentityConfig(identityMap, pathContext+".identity"); err == nil {
				config.Identity, err = identity.LoadIdentity(*identityConfig)
				if err != nil {
					return fmt.Errorf("error loading identity: %v", err)
				}

				if err := config.Identity.WatchFiles(); err != nil {
					pfxlog.Logger().Warnf("could not enable file watching on bind point identity: %v", err)
				}
			} else {
				return fmt.Errorf("error parsing identity section: %v", err)
			}

		} else {
			return errors.New("identity section must be a map if defined")
		}

	} //no else, optional, will defer to router identity

	//parse options
	config.Options = ServerConfigOptions{}
	config.Options.Default()

	if optionsInterface, ok := configMap["options"]; ok {
		if optionMap, ok := optionsInterface.(map[interface{}]interface{}); ok {
			if err := config.Options.Parse(optionMap); err != nil {
				return fmt.Errorf("error parsing options section: %v", err)
			}
		} //no else, options are optional
	}

	return nil
}

// Validate all ServerConfig values
func (config *ServerConfig) Validate(registry Registry) error {
	if config.Name == "" {
		return errors.New("name must not be empty")
	}

	if len(config.APIs) <= 0 {
		return errors.New("no APIs specified, must specify at least one")
	}

	for i, api := range config.APIs {
		if err := api.Validate(); err != nil {
			return fmt.Errorf("invalid ApiConfig at index [%d]: %v", i, err)
		}

		//check if binding is valid
		if binding := registry.Get(api.Binding()); binding == nil {
			return fmt.Errorf("invalid ApiConfig at index [%d]: invalid binding %s", i, api.Binding())
		}
	}

	if len(config.BindPoints) <= 0 {
		return errors.New("no bindPoint specified, must specify at lest one")
	}

	for i, bp := range config.BindPoints {
		if bp != nil {
			id := config.Identity
			if id == nil {
				id = config.DefaultIdentity
			}
			if err := bp.Validate(id); err != nil {
				return fmt.Errorf("invalid bindPoint at index [%d]: %v", i, err)
			}
		} else {
			return errors.New("a nil bindPoint was processed")
		}
	}

	if config.Identity == nil {
		if config.DefaultIdentity == nil {
			return errors.New("no default identity specified and no identity specified")
		}

		config.Identity = config.DefaultIdentity
	}

	if err := config.Options.TlsVersionOptions.Validate(); err != nil {
		return fmt.Errorf("invalid TLS version option: %v", err)
	}

	if err := config.Options.TimeoutOptions.Validate(); err != nil {
		return fmt.Errorf("invalid timeout option: %v", err)
	}

	return nil
}
