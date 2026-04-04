package dash0

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoYAMLDependency verifies that the root package does not transitively
// depend on any YAML library. YAML support is isolated in the yaml/ subpackage
// so that consumers who only import the root package do not pull it in.
func TestNoYAMLDependency(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "github.com/dash0hq/dash0-api-client-go")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps failed: %v", err)
	}

	forbidden := []string{
		"gopkg.in/yaml",
		"sigs.k8s.io/yaml",
		"go.yaml.in/yaml",
	}

	for _, dep := range strings.Split(string(out), "\n") {
		dep = strings.TrimSpace(dep)
		for _, banned := range forbidden {
			if strings.Contains(dep, banned) {
				t.Errorf("root package depends on %q — YAML dependencies must stay in the yaml/ subpackage", dep)
			}
		}
	}
}
