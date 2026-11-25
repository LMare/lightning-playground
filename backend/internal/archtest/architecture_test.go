// Package archtest enforces the hexagonal dependency rule: business code never depends on infrastructure.
package archtest

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	module   = "github.com/Lmare/lightning-playground"
	internal = module + "/backend/internal"
)

// importsOf returns, for each package matching the pattern, the packages it imports (tests excluded).
func importsOf(t *testing.T, pattern string) map[string][]string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", pattern).CombinedOutput()
	require.NoError(t, err, string(out))

	result := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result[fields[0]] = fields[1:]
		}
	}
	require.NotEmpty(t, result, "no package found for %s", pattern)
	return result
}

func TestBusinessDoesNotDependOnInfrastructure(t *testing.T) {
	forbidden := []string{
		internal + "/infra",
		internal + "/bootstrap",
		internal + "/testsupport",
		module + "/backend/gRPC", // generated LND protobuf types
		module,                   // root config package (exact match only, see below)
	}

	for pkg, imports := range importsOf(t, "./../business/...") {
		for _, imp := range imports {
			for _, bad := range forbidden {
				if imp == bad || (bad != module && strings.HasPrefix(imp, bad+"/")) {
					assert.Failf(t, "forbidden dependency", "%s imports %s", pkg, imp)
				}
			}
		}
	}
}

// Pages live in infra/http/web; resource packages (lightning, user) must not depend on it,
// and web must not depend on them (the menu coupling stays inside web).
func TestHTTPPagesAreSeparatedFromResources(t *testing.T) {
	web := internal + "/infra/http/web"
	resources := []string{internal + "/infra/http/lightning", internal + "/infra/http/user"}

	for _, res := range resources {
		for pkg, imports := range importsOf(t, "./../infra/http/"+res[strings.LastIndex(res, "/")+1:]) {
			for _, imp := range imports {
				assert.NotEqual(t, web, imp, "%s imports %s", pkg, imp)
			}
		}
	}

	for pkg, imports := range importsOf(t, "./../infra/http/web") {
		for _, imp := range imports {
			for _, res := range resources {
				assert.False(t, imp == res || strings.HasPrefix(imp, res+"/"), "%s imports %s", pkg, imp)
			}
		}
	}
}

func TestBusinessContextsAreIndependent(t *testing.T) {
	for _, ctx := range []struct{ name, other string }{{"lightning", "user"}, {"user", "lightning"}} {
		for pkg, imports := range importsOf(t, "./../business/"+ctx.name+"/...") {
			for _, imp := range imports {
				assert.False(t, strings.HasPrefix(imp, internal+"/business/"+ctx.other), "%s imports %s", pkg, imp)
			}
		}
	}
}
