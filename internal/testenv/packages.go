package testenv

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Module is the module every package of this repository sits under.
const Module = "github.com/JHK/work-cli"

// machineHome is the home directory the process started under, read before init
// takes it away. The go tool keeps its caches there, and [Listed] is the one
// thing a test runs that needs them.
var machineHome = os.Getenv("HOME")

// Listed runs go list over the module rooted at root and hands back the line it
// wrote for each package. A guard on what the packages import reads them through
// this rather than walking the module itself.
func Listed(t *testing.T, root string, args ...string) []string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir, cmd.Env = root, append(os.Environ(), "HOME="+machineHome)
	var said strings.Builder
	cmd.Stderr = &said
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s: %v\n%s", strings.Join(args, " "), err, said.String())
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
