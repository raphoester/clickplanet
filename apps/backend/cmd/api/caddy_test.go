package main

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Caddy answers 404 for a path it does not list, and the client reads a 404 as a service the server does not offer.
func TestCaddyRoutesEveryPublicService(t *testing.T) {
	caddyfile, err := os.ReadFile("../../../../deploy/vps/caddy/Caddyfile")
	require.NoError(t, err)

	pkg := regexp.MustCompile(`(?m)^package ([a-z0-9.]+);`)
	service := regexp.MustCompile(`(?m)^service ([A-Za-z]+) \{`)
	var public []string
	protos := os.DirFS("../../../../proto")
	require.NoError(t, fs.WalkDir(protos, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".proto") {
			return err
		}
		source, err := fs.ReadFile(protos, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		named := pkg.FindSubmatch(source)
		for _, found := range service.FindAllSubmatch(source, -1) {
			name := string(found[1])
			if name == "AdminService" || name == "InternalService" {
				continue
			}
			public = append(public, string(named[1])+"."+name)
		}
		return nil
	}))
	require.NotEmpty(t, public)

	for _, name := range public {
		assert.Contains(t, string(caddyfile), "handle /"+name+"/* {", "%s is public and Caddy does not route it", name)
	}
}
