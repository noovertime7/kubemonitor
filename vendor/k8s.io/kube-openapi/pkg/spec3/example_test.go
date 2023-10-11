/*
Copyright 2021 The Kubernetes Authors.

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

package spec3_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"k8s.io/kube-openapi/pkg/spec3"
	jsontesting "k8s.io/kube-openapi/pkg/util/jsontesting"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func TestExampleRoundtrip(t *testing.T) {
	cases := []jsontesting.RoundTripTestCase{
		{
			Name: "Basic Roundtrip",
			Object: &spec3.Example{
				spec.Refable{Ref: spec.MustCreateRef("Dog")},
				spec3.ExampleProps{
					Description: "foo",
				},
				spec.VendorExtensible{Extensions: spec.Extensions{
					"x-framework": "go-swagger",
				}},
			},
		},
	}

	for _, tcase := range cases {
		t.Run(tcase.Name, func(t *testing.T) {
			require.NoError(t, tcase.RoundTripTest(&spec3.Example{}))
		})
	}
}

func TestExampleJSONSerialization(t *testing.T) {
	cases := []struct {
		name           string
		target         *spec3.Example
		expectedOutput string
	}{
		{
			name: "basic",
			target: &spec3.Example{
				ExampleProps: spec3.ExampleProps{
					Summary: "An example",
					Value:   map[string]string{"foo": "bar"},
				},
			},
			expectedOutput: `{"summary":"An example","value":{"foo":"bar"}}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawTarget, err := json.Marshal(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			serializedTarget := string(rawTarget)
			if !cmp.Equal(serializedTarget, tc.expectedOutput) {
				t.Fatalf("diff %s", cmp.Diff(serializedTarget, tc.expectedOutput))
			}
		})
	}
}
