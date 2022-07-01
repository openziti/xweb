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

import "context"

const (
	HandlerContextKey = ContextKey("xweb.ApiHandler.ContextKey")
	ServerContextKey  = ContextKey("xweb.Server.ContextKey")
)

// HandlerFromRequestContext us a utility function to retrieve a ApiHandler reference, that the demux http.Handler
// deferred to, during downstream  http.Handler processing from the http.Request context.
func HandlerFromRequestContext(ctx context.Context) *ApiHandler {
	if val := ctx.Value(HandlerContextKey); val != nil {
		if handler, ok := val.(*ApiHandler); ok {
			return handler
		}
	}
	return nil
}

// ServerContextFromRequestContext is a utility function to retrieve a *ServerContext reference from the http.Request
// that provides access to XWeb configuration like BindPointConfig, ServerConfig, and InstanceConfig values.
func ServerContextFromRequestContext(ctx context.Context) *ServerContext {
	if val := ctx.Value(ServerContextKey); val != nil {
		if serverContext, ok := val.(*ServerContext); ok {
			return serverContext
		}
	}
	return nil
}
