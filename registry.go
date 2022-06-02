/*
	Copyright NetFoundry, Inc.

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
	"github.com/sirupsen/logrus"
)

// Registry describes a registry of binding to ApiHandlerFactory registrations
type Registry interface {
	Add(factory ApiHandlerFactory) error
	Get(binding string) ApiHandlerFactory
}

// RegistryMap is a basic Registry implementation backed by a simple mapping of binding (string) to ApiHandlerFactory instances
type RegistryMap struct {
	factories map[string]ApiHandlerFactory
}

// NewRegistryMap creates a new RegistryMap
func NewRegistryMap() *RegistryMap {
	return &RegistryMap{
		factories: map[string]ApiHandlerFactory{},
	}
}

// Add adds a factory to the registry. Errors if a previous factory with the same binding is registered.
func (registry RegistryMap) Add(factory ApiHandlerFactory) error {
	logrus.Debugf("adding xweb factory with binding: %v", factory.Binding())
	if _, ok := registry.factories[factory.Binding()]; ok {
		return fmt.Errorf("binding [%s] already registered", factory.Binding())
	}

	registry.factories[factory.Binding()] = factory

	return nil
}

// Get retrieves a factory based on a binding or nil if no factory for the binding is registered
func (registry RegistryMap) Get(binding string) ApiHandlerFactory {
	return registry.factories[binding]
}
