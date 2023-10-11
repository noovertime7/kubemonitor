/*
Copyright 2023 The Kubernetes Authors.

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

package cache

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDefaultOpts(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{}
	testCases := []struct {
		name string
		in   Options

		verification func(Options) string
	}{
		{
			name: "ByObject.Namespaces gets defaulted from ByObject",
			in: Options{
				ByObject: map[client.Object]ByObject{pod: {
					Namespaces: map[string]Config{
						"default": {},
					},
					Label: labels.SelectorFromSet(map[string]string{"from": "by-object"}),
				}},
				DefaultNamespaces: map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-namespaces"})},
				},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "by-object"})},
				}
				return cmp.Diff(expected, o.ByObject[pod].Namespaces)
			},
		},
		{
			name: "ByObject.Namespaces gets defaulted from DefaultNamespaces",
			in: Options{
				ByObject: map[client.Object]ByObject{pod: {
					Namespaces: map[string]Config{
						"default": {},
					},
				}},
				DefaultNamespaces: map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-namespaces"})},
				},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-namespaces"})},
				}
				return cmp.Diff(expected, o.ByObject[pod].Namespaces)
			},
		},
		{
			name: "ByObject.Namespaces gets defaulted from DefaultLabelSelector",
			in: Options{
				ByObject: map[client.Object]ByObject{pod: {
					Namespaces: map[string]Config{
						"default": {},
					},
				}},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"})},
				}
				return cmp.Diff(expected, o.ByObject[pod].Namespaces)
			},
		},
		{
			name: "ByObject.Namespaces gets defaulted from DefaultNamespaces",
			in: Options{
				ByObject: map[client.Object]ByObject{pod: {}},
				DefaultNamespaces: map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-namespaces"})},
				},
			},

			verification: func(o Options) string {
				expected := map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-namespaces"})},
				}
				return cmp.Diff(expected, o.ByObject[pod].Namespaces)
			},
		},
		{
			name: "ByObject.Namespaces doesn't get defaulted when its empty",
			in: Options{
				ByObject: map[client.Object]ByObject{pod: {Namespaces: map[string]Config{}}},
				DefaultNamespaces: map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-namespaces"})},
				},
			},

			verification: func(o Options) string {
				expected := map[string]Config{}
				return cmp.Diff(expected, o.ByObject[pod].Namespaces)
			},
		},
		{
			name: "ByObject.Labels gets defaulted from DefautLabelSelector",
			in: Options{
				ByObject:             map[client.Object]ByObject{pod: {}},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := labels.SelectorFromSet(map[string]string{"from": "default-label-selector"})
				return cmp.Diff(expected, o.ByObject[pod].Label)
			},
		},
		{
			name: "ByObject.Labels doesn't get defaulted when set",
			in: Options{
				ByObject:             map[client.Object]ByObject{pod: {Label: labels.SelectorFromSet(map[string]string{"from": "by-object"})}},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := labels.SelectorFromSet(map[string]string{"from": "by-object"})
				return cmp.Diff(expected, o.ByObject[pod].Label)
			},
		},
		{
			name: "ByObject.Fields gets defaulted from DefaultFieldSelector",
			in: Options{
				ByObject:             map[client.Object]ByObject{pod: {}},
				DefaultFieldSelector: fields.SelectorFromSet(map[string]string{"from": "default-field-selector"}),
			},

			verification: func(o Options) string {
				expected := fields.SelectorFromSet(map[string]string{"from": "default-field-selector"})
				return cmp.Diff(expected, o.ByObject[pod].Field, cmp.Exporter(func(reflect.Type) bool { return true }))
			},
		},
		{
			name: "ByObject.Fields doesn't get defaulted when set",
			in: Options{
				ByObject:             map[client.Object]ByObject{pod: {Field: fields.SelectorFromSet(map[string]string{"from": "by-object"})}},
				DefaultFieldSelector: fields.SelectorFromSet(map[string]string{"from": "default-field-selector"}),
			},

			verification: func(o Options) string {
				expected := fields.SelectorFromSet(map[string]string{"from": "by-object"})
				return cmp.Diff(expected, o.ByObject[pod].Field, cmp.Exporter(func(reflect.Type) bool { return true }))
			},
		},
		{
			name: "ByObject.UnsafeDisableDeepCopy gets defaulted from DefaultUnsafeDisableDeepCopy",
			in: Options{
				ByObject:                     map[client.Object]ByObject{pod: {}},
				DefaultUnsafeDisableDeepCopy: pointer.Bool(true),
			},

			verification: func(o Options) string {
				expected := pointer.Bool(true)
				return cmp.Diff(expected, o.ByObject[pod].UnsafeDisableDeepCopy)
			},
		},
		{
			name: "ByObject.UnsafeDisableDeepCopy doesn't get defaulted when set",
			in: Options{
				ByObject:                     map[client.Object]ByObject{pod: {UnsafeDisableDeepCopy: pointer.Bool(false)}},
				DefaultUnsafeDisableDeepCopy: pointer.Bool(true),
			},

			verification: func(o Options) string {
				expected := pointer.Bool(false)
				return cmp.Diff(expected, o.ByObject[pod].UnsafeDisableDeepCopy)
			},
		},
		{
			name: "DefaultNamespace label selector gets defaulted from DefaultLabelSelector",
			in: Options{
				DefaultNamespaces:    map[string]Config{"default": {}},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := map[string]Config{
					"default": {LabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"})},
				}
				return cmp.Diff(expected, o.DefaultNamespaces)
			},
		},
		{
			name: "DefaultNamespace label selector doesn't get defaulted when set",
			in: Options{
				DefaultNamespaces:    map[string]Config{"default": {LabelSelector: labels.Everything()}},
				DefaultLabelSelector: labels.SelectorFromSet(map[string]string{"from": "default-label-selector"}),
			},

			verification: func(o Options) string {
				expected := map[string]Config{
					"default": {LabelSelector: labels.Everything()},
				}
				return cmp.Diff(expected, o.DefaultNamespaces)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Mapper = &fakeRESTMapper{}

			defaulted, err := defaultOpts(&rest.Config{}, tc.in)
			if err != nil {
				t.Fatal(err)
			}

			if diff := tc.verification(defaulted); diff != "" {
				t.Errorf("expected config differs from actual: %s", diff)
			}
		})
	}
}

type fakeRESTMapper struct {
	meta.RESTMapper
}

func (f *fakeRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	return &meta.RESTMapping{Scope: meta.RESTScopeNamespace}, nil
}

func TestDefaultConfigConsidersAllFields(t *testing.T) {
	t.Parallel()
	seed := time.Now().UnixNano()
	t.Logf("Seed is %d", seed)
	f := fuzz.NewWithSeed(seed).Funcs(
		func(ls *labels.Selector, _ fuzz.Continue) {
			*ls = labels.SelectorFromSet(map[string]string{"foo": "bar"})
		},
		func(fs *fields.Selector, _ fuzz.Continue) {
			*fs = fields.SelectorFromSet(map[string]string{"foo": "bar"})
		},
		func(tf *cache.TransformFunc, _ fuzz.Continue) {
			// never default this, as functions can not be compared so we fail down the line
		},
	)

	for i := 0; i < 100; i++ {
		fuzzed := Config{}
		f.Fuzz(&fuzzed)

		defaulted := defaultConfig(Config{}, fuzzed)

		if diff := cmp.Diff(fuzzed, defaulted, cmp.Exporter(func(reflect.Type) bool { return true })); diff != "" {
			t.Errorf("Defaulted config doesn't match fuzzed one: %s", diff)
		}
	}
}
