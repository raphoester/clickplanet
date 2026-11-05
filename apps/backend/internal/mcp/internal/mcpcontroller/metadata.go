package mcpcontroller

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// RFC 9728: the metadata of /mcp is at /.well-known/oauth-protected-resource/mcp; the bare path is served too.
const metadataPrefix = "/.well-known/oauth-protected-resource"

type metadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	BearerMethods        []string `json:"bearer_methods_supported"`
	ResourceName         string   `json:"resource_name"`
}

func MetadataPaths(resource *url.URL) (inserted, bare string) {
	return metadataPrefix + resource.EscapedPath(), metadataPrefix
}

func MetadataURL(resource *url.URL) string {
	inserted, _ := MetadataPaths(resource)
	return resource.Scheme + "://" + resource.Host + inserted
}

func NewMetadata(resource *url.URL) http.Handler {
	body, err := json.Marshal(metadata{
		Resource:             resource.String(),
		AuthorizationServers: []string{resource.Scheme + "://" + resource.Host},
		BearerMethods:        []string{"header"},
		ResourceName:         "ClickPlanet",
	})
	if err != nil {
		panic(err) // a struct of strings always encodes.
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(body)
	})
}
