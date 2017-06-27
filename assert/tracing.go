/*
Copyright 2017 Turbine Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package assert

import (
	"os"
	"testing"

	"github.com/turbinelabs/test/stack"
)

// A TracingTB embeds a testing.TB, overriding the Errorf and Fatalf methods to
// append stack traces.
type TracingTB struct {
	testing.TB
}

// Tracing wraps a testing.T or testing.TB so that stack traces are appended to
// all Errorf and Fatalf calls. If a TracingTB or G is supplied, it is returned
// unmodified
func Tracing(t testing.TB) testing.TB {
	switch obj := t.(type) {
	case *G:
		return obj
	case *TracingTB:
		return obj
	default:
		return &TracingTB{t}
	}
}

func stackTrace() string {
	tbnPath := os.Getenv(tbnHomePath) + "/"

	trace := stack.New()
	if tbnPath != "/" {
		trace.TrimPaths(tbnPath, goPath)
	}

	trace.PopFrames("test/")

	return "\n" + trace.Format(true)
}

// Errorf invokes the underlying testing.TB's Errorf function with the
// given error message and arguments.
func (tr *TracingTB) Errorf(format string, args ...interface{}) {
	tr.TB.Errorf(format+" in %s", append(args, stackTrace())...)
}

// Error invokes the underlying testing.TB's Error function with the
// given arguments.
func (tr *TracingTB) Error(args ...interface{}) {
	args = append(args, "in", stackTrace())
	tr.TB.Error(args...)
}

// Fatalf invokes the underlying testing.TB's Fatalf function with the
// given error message and arguments.
func (tr *TracingTB) Fatalf(format string, args ...interface{}) {
	tr.TB.Fatalf(format+" in %s", append(args, stackTrace())...)
}

// Fatal invokes the underlying testing.TB's Fatal function with the
// given arguments.
func (tr *TracingTB) Fatal(args ...interface{}) {
	args = append(args, "in", stackTrace())
	tr.TB.Fatal(args...)
}
