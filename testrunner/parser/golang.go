/*
Copyright 2018 Turbine Labs, Inc.

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

package parser

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/turbinelabs/test/testrunner/results"
)

var (
	resultRegex  = regexp.MustCompile(`^--- (PASS|FAIL|SKIP): (.+) \((\d+\.\d+)(?: seconds|s)\)$`)
	summaryRegex = regexp.MustCompile(`^(?:PASS|FAIL)$`)
)

// Convert verbose go test output into test results suitable for formatting.
func ParseTestOutput(
	pkgName string,
	duration time.Duration,
	output *bytes.Buffer,
) ([]*results.TestPackage, error) {
	eof := false
	var t *results.Test
	testPkg := results.TestPackage{
		Name:     pkgName,
		Result:   results.Skipped,
		Duration: duration.Seconds(),
		Tests:    make([]*results.Test, 0),
		Output:   string(output.Bytes()),
	}

	for !eof {
		lineBytes, err := output.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				eof = true
			} else {
				return nil, err
			}
		}

		line := strings.TrimRightFunc(string(lineBytes), unicode.IsSpace)

		if t == nil {
			if strings.HasPrefix(line, "=== RUN ") {
				// start of test
				t = new(results.Test)
				t.Name = strings.TrimSpace(line[8:])
			} else if m := summaryRegex.FindStringSubmatch(line); len(m) == 1 {
				// End of package
				if testPkg.Result != results.Skipped {
					return nil, fmt.Errorf("expected only a single package")
				}
				switch line {
				case "PASS":
					testPkg.Result = results.Passed
				default:
					testPkg.Result = results.Failed
				}
			} else if len(testPkg.Tests) > 0 && strings.HasPrefix(line, "\t") {
				// test failure output
				lastTest := testPkg.Tests[len(testPkg.Tests)-1]
				lastTest.Failure.Write(lineBytes)
			}
		} else {
			if m := resultRegex.FindStringSubmatch(line); len(m) == 4 {
				// end of test
				switch m[1] {
				case "PASS":
					t.Result = results.Passed
				case "SKIP":
					t.Result = results.Skipped
				default:
					t.Result = results.Failed
				}
				t.Duration, _ = strconv.ParseFloat(m[3], 64)

				testPkg.Tests = append(testPkg.Tests, t)
				t = nil
			} else {
				t.Output.Write(lineBytes)
			}
		}
	}

	if testPkg.Result == results.Skipped {
		testPkg.Result = results.Failed
		testPkg.Output += "\n[Did not find package result: marking package as failed.]\n"
	}

	return []*results.TestPackage{&testPkg}, nil
}

// Forces the presence of the go test verbose flag (-test.v=true).
func ForceVerboseFlag(args []string) []string {
	for i, arg := range args {
		if arg == "-test.v=true" {
			return args
		} else if arg == "-test.v=false" {
			args = append(args[0:i], args[i+1:]...)
			break
		}
	}

	return append(args, "-test.v=true")
}
