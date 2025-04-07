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
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/michaelquigley/pfxlog"
	"github.com/openziti/foundation/v2/debugz"
	transporttls "github.com/openziti/transport/v2/tls"
	"github.com/openziti/xweb/v2/middleware"
	"io"
	"log"
	"net"
	"net/http"
)

type ContextKey string

const (
	ZitiCtrlAddressHeader = "ziti-ctrl-address"
)

type ServerContext struct {
	BindPoint    *BindPointConfig
	ServerConfig *ServerConfig
	Config       *InstanceConfig
}

type namedHttpServer struct {
	*http.Server
	ApiBindingList  []string
	BindPointConfig *BindPointConfig
	ServerConfig    *ServerConfig
	InstanceConfig  *InstanceConfig
}

func (s namedHttpServer) NewBaseContext(_ net.Listener) context.Context {
	serverContext := &ServerContext{
		BindPoint:    s.BindPointConfig,
		ServerConfig: s.ServerConfig,
		Config:       s.InstanceConfig,
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, ServerContextKey, serverContext)

	return ctx
}

// Server represents all the http.Server's and http.Handler's necessary to run a single xweb.ServerConfig
type Server struct {
	DefaultHttpHandlerProviderImpl
	HttpServers    []*namedHttpServer
	logWriter      *io.PipeWriter
	options        *ServerConfigOptions
	config         interface{}
	Handle         http.Handler
	OnHandlerPanic func(writer http.ResponseWriter, request *http.Request, panicVal interface{})
	ServerConfig   *ServerConfig
}

// NewServer creates a new Server from a ServerConfig. All necessary http.Handler's will be created from the supplied
// DemuxFactory and Registry.
func NewServer(instance Instance, serverConfig *ServerConfig) (*Server, error) {
	logWriter := pfxlog.Logger().Writer()

	tlsConfig := serverConfig.Identity.ServerTLSConfig()
	tlsConfig.ClientAuth = tls.RequestClientCert
	tlsConfig.MinVersion = uint16(serverConfig.Options.MinTLSVersion)
	tlsConfig.MaxVersion = uint16(serverConfig.Options.MaxTLSVersion)

	server := &Server{
		logWriter:    logWriter,
		config:       &serverConfig,
		HttpServers:  []*namedHttpServer{},
		ServerConfig: serverConfig,
	}

	server.SetParent(instance)

	var handlers []ApiHandler
	var apiBindingList []string

	for _, api := range serverConfig.APIs {
		if apiFactory := instance.GetRegistry().Get(api.Binding()); apiFactory != nil {
			if handler, err := apiFactory.New(serverConfig, api.Options()); err != nil {
				pfxlog.Logger().Fatalf("encountered error building handler for api binding [%s]: %v", api.Binding(), err)
			} else {
				handlers = append(handlers, handler)
				apiBindingList = append(apiBindingList, api.binding)
			}
		} else {
			pfxlog.Logger().Fatalf("encountered api binding [%s] which has no associated factory registered", api.Binding())
		}
	}

	demuxHandler, err := instance.GetDemuxFactory().Build(handlers)

	if err != nil {
		return nil, fmt.Errorf("error creating server: %v", err)
	}

	demuxHandler.SetParent(server)

	for _, bindPoint := range serverConfig.BindPoints {
		namedServer := &namedHttpServer{
			ApiBindingList:  apiBindingList,
			ServerConfig:    serverConfig,
			BindPointConfig: bindPoint,
			InstanceConfig:  instance.GetConfig(),
			Server: &http.Server{
				Addr:         bindPoint.InterfaceAddress,
				WriteTimeout: serverConfig.Options.WriteTimeout,
				ReadTimeout:  serverConfig.Options.ReadTimeout,
				IdleTimeout:  serverConfig.Options.IdleTimeout,
				Handler:      server.wrapHandler(serverConfig, bindPoint, demuxHandler),
				TLSConfig:    tlsConfig,
				ErrorLog:     log.New(logWriter, "", 0),
			},
		}

		namedServer.BaseContext = namedServer.NewBaseContext

		server.HttpServers = append(server.HttpServers, namedServer)
	}

	for _, mutator := range instance.GetConfig().Options.ServerMutators {
		if err = mutator(instance, serverConfig, server); err != nil {
			return nil, fmt.Errorf("encountered error mutating server instance: %v", err)
		}
	}

	return server, nil
}

func (server *Server) wrapHandler(_ *ServerConfig, point *BindPointConfig, handler http.Handler) http.Handler {
	//innermost/bottom -> outermost/top
	handler = server.wrapSetCtrlAddressHeader(point, handler)
	handler = server.wrapPanicRecovery(handler)
	handler = middleware.NewCompressionHandler(handler)
	return handler
}

// wrapPanicRecovery wraps a http.Handler with another http.Handler that provides recovery.
func (server *Server) wrapPanicRecovery(handler http.Handler) http.Handler {
	wrappedHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if panicVal := recover(); panicVal != nil {
				if server.OnHandlerPanic != nil {
					server.OnHandlerPanic(writer, request, panicVal)
					return
				}
				pfxlog.Logger().Errorf("panic caught by server handler: %v\n%v", panicVal, debugz.GenerateLocalStack())
			}
		}()

		handler.ServeHTTP(writer, request)
	})

	return wrappedHandler
}

// wrapSetCtrlAddressHeader will check to see if the bindPoint is configured to advertise a "new address". If so
// the value is added to the ZitiCtrlAddressHeader which will be sent out on every response. Clients can check this
// header to be notified that the controller is or will be moving from one ip/hostname to another. When the
// new address value is set, both the old and new addresses should be valid as the clients will begin using the
// new address on their next connect.
func (server *Server) wrapSetCtrlAddressHeader(point *BindPointConfig, handler http.Handler) http.Handler {
	wrappedHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if point.NewAddress != "" {
			address := "https://" + point.NewAddress
			writer.Header().Set(ZitiCtrlAddressHeader, address)
		}

		handler.ServeHTTP(writer, request)
	})

	return wrappedHandler
}

// Start the server and all underlying http.Server's
func (server *Server) Start() error {
	logger := pfxlog.Logger()

	for _, httpServer := range server.HttpServers {
		logger.Infof("starting ApiConfig to listen and serve tls on %s for server %s with APIs: %v", httpServer.Addr, httpServer.ServerConfig.Name, httpServer.ApiBindingList)

		cfg := httpServer.TLSConfig
		// make sure to listen to the expected protocols
		cfg.NextProtos = append(cfg.NextProtos, "h2", "http/1.1", "")
		l, err := transporttls.ListenTLS(httpServer.Addr, httpServer.ServerConfig.Name, cfg)
		if err != nil {
			return fmt.Errorf("error listening: %s", err)
		}
		err = httpServer.Serve(l)

		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("error listening: %s", err)
		}
	}

	return nil
}

// Shutdown stops the server and all underlying http.Server's
func (server *Server) Shutdown(ctx context.Context) {
	_ = server.logWriter.Close()

	for _, httpServer := range server.HttpServers {
		localServer := httpServer
		func() {
			_ = localServer.Shutdown(ctx)
		}()
	}
}
