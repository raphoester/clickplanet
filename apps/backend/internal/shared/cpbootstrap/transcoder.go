package cpbootstrap

import (
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/vanguard"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func (r *rpcRoutes) transcodeOn(router *http.ServeMux, middlewares func(http.Handler) http.Handler) error {
	var services []*vanguard.Service
	for _, path := range r.paths {
		if declaresHTTPRoutes(path) {
			services = append(services, vanguard.NewService(path, r.byPath[path]))
		}
	}
	if len(services) == 0 {
		return nil
	}

	transcoder, err := vanguard.NewTranscoder(services)
	if err != nil {
		return fmt.Errorf("failed to serve the routes the services declare: %w", err)
	}

	// The fallback: every service path above is more specific, so its Connect calls never come here.
	router.Handle("/", middlewares(transcoder))

	return nil
}

func declaresHTTPRoutes(path string) bool {
	found, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(strings.Trim(path, "/")))
	if err != nil {
		return false
	}
	service, ok := found.(protoreflect.ServiceDescriptor)
	if !ok {
		return false
	}

	methods := service.Methods()
	for i := range methods.Len() {
		if proto.HasExtension(methods.Get(i).Options(), annotations.E_Http) {
			return true
		}
	}

	return false
}
