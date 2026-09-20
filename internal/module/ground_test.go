// Package module holds the guards over the module's own shape.
package module

import (
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// root is where the module sits, from the directory these tests run in.
const root = "../.."
