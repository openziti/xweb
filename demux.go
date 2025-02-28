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
	"errors"
	"fmt"
	"github.com/michaelquigley/pfxlog"
	"net/http"
	"strings"
)

// DemuxFactory generates a http.Handler that interrogates a http.Request and routes them to ApiHandler instances. The selected
// ApiHandler is added to the context with a key of HandlerContextKey. Each DemuxFactory implementation must define
// its own behaviors for an unmatched http.Request.
type DemuxFactory interface {
	Build(handlers []ApiHandler) (DemuxHandler, error)
}

type DemuxHandler interface {
	DefaultHttpHandlerProvider
	http.Handler
}

type DemuxHandlerImpl struct {
	DefaultHttpHandlerProviderImpl
	Handler http.Handler
}

var _ DemuxHandler = &DemuxHandlerImpl{}

func (d *DemuxHandlerImpl) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	d.Handler.ServeHTTP(writer, request)
}

// PathPrefixDemuxFactory is a DemuxFactory that routes http.Request requests to a specific ApiHandler from a set of
// ApiHandler's by URL path prefixes. A http.Handler for NoHandlerFound can be provided to specify behavior to perform
// when a ApiHandler is not selected. By default an empty response with a http.StatusNotFound (404) will be sent.
type PathPrefixDemuxFactory struct {
	DefaultHttpHandlerProviderImpl
}

var _ DemuxFactory = &PathPrefixDemuxFactory{}

// Build performs ApiHandler selection based on URL path prefixes
func (factory *PathPrefixDemuxFactory) Build(handlers []ApiHandler) (DemuxHandler, error) {
	defaultApi, err := getDefault(handlers)

	if err != nil {
		return nil, err
	}

	handlerMap := map[string]ApiHandler{}

	for _, handler := range handlers {
		if existing, ok := handlerMap[handler.RootPath()]; ok {
			return nil, fmt.Errorf("duplicate root path [%s] detected for both bindings [%s] and [%s]", handler.RootPath(), handler.Binding(), existing.Binding())
		}
		handlerMap[handler.RootPath()] = handler
	}

	return &DemuxHandlerImpl{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			for _, handler := range handlers {
				if strings.HasPrefix(request.URL.Path, handler.RootPath()) {

					//store this ApiHandler on the request context, useful for logging by downstream http handlers
					ctx := context.WithValue(request.Context(), HandlerContextKey, handler)
					newRequest := request.WithContext(ctx)
					handler.ServeHTTP(writer, newRequest)
					return
				}
			}

			if defaultApi != nil {
				ctx := context.WithValue(request.Context(), HandlerContextKey, defaultApi)
				newRequest := request.WithContext(ctx)
				defaultApi.ServeHTTP(writer, newRequest)
				return
			}

			if defaultHttpHandler := factory.GetDefaultHttpHandler(); defaultHttpHandler != nil {
				defaultHttpHandler.ServeHTTP(writer, request)
				return
			}

			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte{})
		}),
	}, nil
}

// getDefault determines from a slice of ApiHandler which will act as the default handlers
// should a request not match any handler. The default is determined in one of two ways:
// 1) a handler declares itself the default
// 2) no handler declares itself the default
//
// If a handler declares itself the default, only one is allowed to do so and if another
// handler does so, it will generate an error. If no handler declares itself, the
// last handler will be used.
func getDefault(handlers []ApiHandler) (ApiHandler, error) {
	var defaults []ApiHandler

	if len(handlers) == 0 {
		return nil, errors.New("no handlers provided")
	}

	for _, handler := range handlers {
		if curHandler, ok := handler.(DefaultApiHandler); ok {
			if curHandler.IsDefault() {
				defaults = append(defaults, curHandler)
			}
		}
	}

	if len(defaults) == 0 {
		lastHandler := handlers[len(handlers)-1]
		pfxlog.Logger().Warnf("no default handlers were found, using the last handler [Binding: %s, Type: %T] as the default", lastHandler.Binding(), lastHandler)
		return lastHandler, nil
	}

	if len(defaults) > 1 {
		var names []string
		for _, handler := range defaults {
			name := fmt.Sprintf("[Binding: %s, Type: %T]", handler.Binding(), handler)
			names = append(names, name)
		}

		strNames := strings.Join(names, ",")
		return nil, errors.New("too many default handlers found, ensure that only one handler is marked as the default: " + strNames)
	}

	return defaults[0], nil
}

// IsHandledDemuxFactory is a DemuxFactory that routes http.Request requests to a specific ApiHandler by delegating
// to the ApiHandler's IsHandled function.
type IsHandledDemuxFactory struct {
	DefaultHttpHandlerProviderImpl
}

var _ DemuxFactory = &IsHandledDemuxFactory{}

// Build performs ApiHandler selection based on IsHandled()
func (factory *IsHandledDemuxFactory) Build(handlers []ApiHandler) (DemuxHandler, error) {
	defaultApi, err := getDefault(handlers)

	if err != nil {
		return nil, err
	}

	return &DemuxHandlerImpl{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			for _, handler := range handlers {
				if handler.IsHandler(request) {
					ctx := context.WithValue(request.Context(), HandlerContextKey, handler)
					newRequest := request.WithContext(ctx)
					handler.ServeHTTP(writer, newRequest)
					return
				}

			}

			if defaultApi != nil {
				ctx := context.WithValue(request.Context(), HandlerContextKey, defaultApi)
				newRequest := request.WithContext(ctx)
				defaultApi.ServeHTTP(writer, newRequest)
				return
			}

			if defaultHttpHandler := factory.GetDefaultHttpHandler(); defaultHttpHandler != nil {
				defaultHttpHandler.ServeHTTP(writer, request)
				return
			}

			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write([]byte{})
		}),
	}, nil
}

type DefaultApiHandler interface {
	ApiHandler
	IsDefault() bool
}
