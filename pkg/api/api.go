// Package api defines the interface to a build service
package api

import (
	"bytes"
	"fmt"

	"github.com/grafana/k6build"
)

// BuildRequest defines a request to the build service
type BuildRequest struct {
	K6Constrains string               `json:"k6,omitempty"`
	Dependencies []k6build.Dependency `json:"dependencies,omitempty"`
	Platform     string               `json:"platform,omitempty"`
}

// String returns a text serialization of the BuildRequest
func (r BuildRequest) String() string {
	buffer := &bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("platform: %s", r.Platform))
	buffer.WriteString(fmt.Sprintf("k6: %s", r.K6Constrains))
	for _, d := range r.Dependencies {
		buffer.WriteString(fmt.Sprintf("%s:%q", d.Name, d.Constraints))
	}
	return buffer.String()
}

// BuildResponse defines the response for a BuildRequest
type BuildResponse struct {
	Error    string           `json:"error,omitempty"`
	Artifact k6build.Artifact `json:"artifact,omitempty"`
}
