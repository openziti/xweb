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
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/openziti/identity"
	"github.com/stretchr/testify/require"
)

var _ BindPoint = (*errListenerBindPoint)(nil)

// errListenerBindPoint is a BindPoint whose Listener always fails, modeling a
// misconfigured bind point (e.g. an overlay listener that cannot authenticate).
type errListenerBindPoint struct{}

func (errListenerBindPoint) Listener(string, *gotls.Config) (net.Listener, error) {
	return nil, errors.New("listener failed")
}
func (errListenerBindPoint) BeforeHandler(next http.Handler) http.Handler { return next }
func (errListenerBindPoint) AfterHandler(prev http.Handler) http.Handler  { return prev }
func (errListenerBindPoint) Validate(identity.Identity) error             { return nil }
func (errListenerBindPoint) ServerAddress() string                        { return "test-bind-point" }
func (errListenerBindPoint) Type() BindPointType                          { return "test" }

// Test_Server_Start_ListenerError verifies that when a bind point's Listener
// returns an error, Start logs it and returns rather than calling Serve with a
// nil listener (which panics inside net/http).
func Test_Server_Start_ListenerError(t *testing.T) {
	req := require.New(t)

	server := &Server{
		ServerConfig: &ServerConfig{Name: "test-server"},
		HttpServers: []*namedHttpServer{
			{
				ApiBindingList:  []string{"test"},
				ServerConfig:    &ServerConfig{Name: "test-server"},
				BindPointConfig: errListenerBindPoint{},
				Server: &http.Server{
					Addr:      "test-bind-point",
					TLSConfig: &gotls.Config{},
				},
			},
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- server.Start()
	}()

	select {
	case err := <-done:
		req.NoError(err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server.Start did not return after a bind point listener error")
	}
}
