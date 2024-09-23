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
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

var _ ApiHandler = (*mockHandler)(nil)
var _ DefaultApiHandler = (*mockHandler)(nil)

type mockHandler struct {
	isDefault bool
}

func (m *mockHandler) IsDefault() bool {
	return m.isDefault
}

func (m *mockHandler) Binding() string {
	return "mockHandler"
}

func (m *mockHandler) Options() map[interface{}]interface{} {
	return make(map[interface{}]interface{})
}

func (m *mockHandler) RootPath() string {
	return "/mock-handler"
}

func (m *mockHandler) IsHandler(r *http.Request) bool {
	return false
}

func (m *mockHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(m.Binding()))
}

func Test_getDefault(t *testing.T) {

	t.Run("a nil slice results in an error", func(t *testing.T) {
		var handlers []ApiHandler = nil

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.Error(err)
		req.Nil(defaultHandler)
	})

	t.Run("an empty slice results in an error", func(t *testing.T) {
		var handlers []ApiHandler

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.Error(err)
		req.Nil(defaultHandler)
	})

	t.Run("a slice with one non-defaulting entry returns that entry", func(t *testing.T) {
		h1 := &mockHandler{isDefault: false}
		handlers := []ApiHandler{
			h1,
		}

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.NoError(err)
		req.Equal(h1, defaultHandler)
	})

	t.Run("a slice with one defaulting entry returns that entry", func(t *testing.T) {
		h1 := &mockHandler{isDefault: true}
		handlers := []ApiHandler{
			h1,
		}

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.NoError(err)
		req.Equal(h1, defaultHandler)
	})

	t.Run("a slice with multiple non-defaulting entries returns the last entry", func(t *testing.T) {
		h1 := &mockHandler{isDefault: false}
		h2 := &mockHandler{isDefault: false}
		h3 := &mockHandler{isDefault: false}

		handlers := []ApiHandler{
			h1,
			h2,
			h3,
		}

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.NoError(err)
		req.Equal(h3, defaultHandler)
	})

	t.Run("a slice with multiple defaulting entries returns an error", func(t *testing.T) {
		h1 := &mockHandler{isDefault: false}
		h2 := &mockHandler{isDefault: true}
		h3 := &mockHandler{isDefault: true}

		handlers := []ApiHandler{
			h1,
			h2,
			h3,
		}

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.Error(err)
		req.Nil(defaultHandler)
	})

	t.Run("a slice with multiple entries and one defaulting entry returns the defaulting entry", func(t *testing.T) {
		h1 := &mockHandler{isDefault: false}
		h2 := &mockHandler{isDefault: true}
		h3 := &mockHandler{isDefault: false}

		handlers := []ApiHandler{
			h1,
			h2,
			h3,
		}

		defaultHandler, err := getDefault(handlers)

		req := require.New(t)
		req.NoError(err)
		req.Equal(h2, defaultHandler)
	})
}
