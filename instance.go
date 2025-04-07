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
	"context"
	"github.com/michaelquigley/pfxlog"
	"github.com/openziti/identity"
	"net/http"
	"time"
)

// Instance implements config.Subconfig to allow Instance implementations to be used during the normal component startup
// and configuration phase.
type Instance interface {
	DefaultHttpHandlerProvider
	Enabled() bool
	LoadConfig(cfgmap map[interface{}]interface{}) error

	Shutdown()
	GetRegistry() Registry
	GetDemuxFactory() DemuxFactory
	GetConfig() *InstanceConfig

	// Run will build and start all components
	Run()

	// Build will build all components. Usually called before a Start()
	Build()

	// Start will start all built components. Usually called after Build()
	Start()
}

const (
	DefaultIdentitySection = "identity"
	DefaultConfigSection   = "web"
)

// InstanceImpl is a basic implementation of Instance.
type InstanceImpl struct {
	DefaultHttpHandlerProviderImpl
	Config       *InstanceConfig
	servers      []*Server
	Registry     Registry
	DemuxFactory DemuxFactory
}

// InstanceValidator allows custom validation logic to be run. Added during instance creatation. See NewInstance(...).
type InstanceValidator func(config *InstanceConfig) error

type ServerMutator func(instance Instance, serverConfig *ServerConfig, server *Server) error

// InstanceOptions a set of options used by an instance
type InstanceOptions struct {
	// InstanceValidators custom instance validation, may be nil/empty
	InstanceValidators []InstanceValidator

	// DefaultIdentitySection is the configuration section that should load an identity that will be used for all servers that do not specify their own identity.
	DefaultIdentitySection string

	// The default identity to use if already loaded. If nil, DefaultIdentitySection will be used to load one.
	DefaultIdentity identity.Identity

	// DefaultConfigSection is the configuration section for the xweb instance.
	DefaultConfigSection string

	// ServerMutators allow the mutation of Server objects after they are built.
	ServerMutators []ServerMutator
}

var _ Instance = &InstanceImpl{}

// NewInstance creates a new xweb instance with a given API registry.
func NewInstance(registry Registry, options InstanceOptions) *InstanceImpl {
	return &InstanceImpl{
		Registry:     registry,
		DemuxFactory: &IsHandledDemuxFactory{},
		Config: &InstanceConfig{
			Options: options,
		},
	}
}

func NewDefaultInstance(registry Registry, defaultIdentity identity.Identity) *InstanceImpl {
	options := InstanceOptions{
		InstanceValidators:     nil,
		DefaultIdentity:        defaultIdentity,
		DefaultIdentitySection: DefaultIdentitySection,
		DefaultConfigSection:   DefaultConfigSection,
	}
	return NewInstance(registry, options)
}

// GetRegistry returns the associated Registry
func (i *InstanceImpl) GetRegistry() Registry {
	return i.Registry
}

// GetDemuxFactory returns the associated DemuxFactory
func (i *InstanceImpl) GetDemuxFactory() DemuxFactory {
	return i.DemuxFactory
}

// GetConfig returns the associated InstanceConfig
func (i *InstanceImpl) GetConfig() *InstanceConfig {
	return i.Config
}

// Enabled returns true/false on whether this subconfig should be considered enabled
func (i *InstanceImpl) Enabled() bool {
	return i.Config.Enabled()
}

// LoadConfig handles subconfig operations for xweb.Instance components
func (i *InstanceImpl) LoadConfig(cfgmap map[interface{}]interface{}) error {
	if err := i.Config.Parse(cfgmap); err != nil {
		return err
	}

	//validate sets enabled flag to true on success
	if err := i.Config.Validate(i.Registry); err != nil {
		return err
	}

	return nil
}

// Build assembles all the xweb components from configuration and prepares to have Start() called.
func (i *InstanceImpl) Build() {
	for _, serverConfig := range i.Config.ServerConfigs {
		server, err := NewServer(i, serverConfig)

		if err != nil {
			pfxlog.Logger().Fatalf("error starting xweb server for %s: %v", serverConfig.Name, err)
		}

		i.servers = append(i.servers, server)
	}
}

// Start calls Start() on all Servers that were built by calling Build().
func (i *InstanceImpl) Start() {
	for _, server := range i.servers {
		s := server //avoid closure scoping issues
		go func() {
			if err := s.Start(); err != nil {
				pfxlog.Logger().Errorf("error starting server %s: %v", s.ServerConfig.Name, err)
			}
		}()
	}
}

// Run builds and starts the necessary xweb.Server's
func (i *InstanceImpl) Run() {
	i.Build()
	i.Start()
}

// Shutdown stop all running xweb.Server's
func (i *InstanceImpl) Shutdown() {
	for _, server := range i.servers {
		localServer := server
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
			localServer.Shutdown(ctx)
		}()
	}
}

// DefaultHttpHandlerProvider is an interface that allows different levels of xweb's components: Instance, ServerConfig,
// Server. The default handler used when no matching ApiHandler is found is: Instance > ServerConfig > Server
type DefaultHttpHandlerProvider interface {
	GetDefaultHttpHandler() http.Handler
	SetDefaultHttpHandler(handler http.Handler)
	SetParent(parent DefaultHttpHandlerProvider)
}

type DefaultHttpHandlerProviderImpl struct {
	Parent      DefaultHttpHandlerProvider
	HttpHandler http.Handler
}

var _ DefaultHttpHandlerProvider = &DefaultHttpHandlerProviderImpl{}

func handler404(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusNotFound)
	_, _ = rw.Write([]byte{})
}

func (d *DefaultHttpHandlerProviderImpl) GetDefaultHttpHandler() http.Handler {
	if d.HttpHandler == nil && d.Parent != nil {
		if handler := d.Parent.GetDefaultHttpHandler(); handler == nil {
			h := http.HandlerFunc(handler404)
			return &h
		} else {
			return handler
		}
	}

	return d.HttpHandler
}

func (d *DefaultHttpHandlerProviderImpl) SetDefaultHttpHandler(handler http.Handler) {
	d.HttpHandler = handler
}

func (d *DefaultHttpHandlerProviderImpl) SetParent(parent DefaultHttpHandlerProvider) {
	d.Parent = parent
}
