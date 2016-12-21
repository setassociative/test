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

package http

import (
	"flag"
	"testing"

	"github.com/turbinelabs/test/assert"
)

type goodTestCase struct {
	args []string
	test func(TestServer) bool
}

func TestFlags(t *testing.T) {
	goodCases := []goodTestCase{
		{[]string{"--ports", "1234,1234,1235,1234,1236,1235"}, func(ts TestServer) bool {
			return len(ts.ports) == 3 && ts.ports[0] == "1234" && ts.ports[1] == "1235" && ts.ports[2] == "1236"
		}},
		{[]string{"--error-rate", "4.1"}, func(ts TestServer) bool { return ts.errorRate == 4.1 }},
		{[]string{"--latency-mean", "4.2"}, func(ts TestServer) bool { return ts.latencyMean == 4.2 }},
		{[]string{"--latency-stddev", "4.3"}, func(ts TestServer) bool { return ts.latencyStdDev == 4.3 }},
	}

	badCases := [][]string{
		{"--error-rate", "-1"},
		{"--error-rate", "100.1"},
		{"--latency-mean", "-1"},
		{"--latency-stddev", "-1"},
	}

	for _, tc := range goodCases {
		var f flag.FlagSet
		res, err := NewTestServerFromFlagSet(&f, tc.args)
		assert.Nil(t, err)
		if !tc.test(*res) {
			t.Errorf("Bad result for %v: %v", tc.args, res)
		}
	}

	for _, tc := range badCases {
		var f flag.FlagSet
		_, err := NewTestServerFromFlagSet(&f, tc)
		assert.NonNil(t, err)
	}
}
