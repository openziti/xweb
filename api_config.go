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

import "github.com/pkg/errors"

// ApiConfig represents some "api" or "site" by binding name. Each ApiConfig configuration is used against a Registry
// to locate the proper factory to generate a ApiHandler. The options provided by this structure are parsed by the
// ApiHandlerFactory and the behavior, valid keys, and valid values are not defined by xweb components, but by that
// ApiHandlerFactory and its resulting ApiHandler's.
type ApiConfig struct {
	binding string
	options map[interface{}]interface{}
}

// Binding returns the string that uniquely identifies bo the ApiHandlerFactory and resulting ApiHandler instances that
// will be attached to some ServerConfig and its resulting Server.
func (api *ApiConfig) Binding() string {
	return api.binding
}

// Options returns the options associated with this ApiConfig binding.
func (api *ApiConfig) Options() map[interface{}]interface{} {
	return api.options
}

// Parse the configuration map for an ApiConfig.
func (api *ApiConfig) Parse(apiConfigMap map[interface{}]interface{}) error {
	if bindingInterface, ok := apiConfigMap["binding"]; ok {
		if binding, ok := bindingInterface.(string); ok {
			api.binding = binding
		} else {
			return errors.New("binding must be a string")
		}
	} else {
		return errors.New("binding is required")
	}

	if optionsInterface, ok := apiConfigMap["options"]; ok {
		if optionsMap, ok := optionsInterface.(map[interface{}]interface{}); ok {
			api.options = optionsMap //leave to bindings to interpret further
		} else {
			return errors.New("options if declared must be a map")
		}
	} //no else optional

	return nil
}

// Validate this configuration object.
func (api *ApiConfig) Validate() error {
	if api.Binding() == "" {
		return errors.New("binding must be specified")
	}

	return nil
}
