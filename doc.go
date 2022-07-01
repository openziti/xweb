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

/*
Package xweb provides facilities to creating composable xweb.ApiHandler instances and http.Server's from configuration files.

Basics

xweb provides customizable and extendable components to stand up multiple http.Server's listening on one or more
network interfaces and ports.

Each Instance is responsible for defining configuration sections to be parsed, parsing the configuration, starting
servers, and shutting down relevant server. An example implementation is included in the package: InstanceImpl. This
implementation should cover most use cases. In addition, InstanceImpl makes use of InstanceConfig which is reusable
component for parsing InstanceImpl configuration sections. Both Instance and InstanceConfig assume that configuration
will be acquired from some source and be presented as a map of interface{}-to-interface{} values.

InstanceConfig configuration sections allow the definition of an array of ServerConfig. In turn each ServerConfig
can listen on many interface/port combinations specified by an array of BindPointConfig's and host many http.Handler's
by defining an array of ApiConfig's that are converted into ApiHandler's. ApiHandler's are http.Handler's with
metadata and can be as complex or as simple as necessary - using other libraries or only the standard http Go
capabilities.

To deal with a single ServerConfig hosting multiple APIs as web.ServerConfig's, incoming requests must be forwarded
to the correct ApiHandler. The responsibility is handled by another configurable http.Handler called an
"xweb demux handler". This handler's responsibility is to inspect incoming requests and forward them to the correct
ApiHandler. It is specified by an DemuxFactory and a reference implementation, PathPrefixDemuxFactory
has been provided.

Another way to say it: each Instance defines a configuration section (default `web`) to define ServerConfig's and their
hosted APIs. Each ServerConfig maps to one Server/http.Server per BindPointConfig. No two Server instances can have
colliding BindPointConfig's due to port conflicts.

*/
package xweb
