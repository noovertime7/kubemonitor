/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusPushSpec defines the desired state of PrometheusPush
type PrometheusPushSpec struct {
	Url           string   `json:"url"`
	BasicAuthUser string   `json:"basic_auth_user,omitempty"`
	BasicAuthPass string   `json:"basic_auth_pass,omitempty"`
	Headers       []string `json:"headers,omitempty"`

	Timeout             int64 `json:"timeout"`
	DialTimeout         int64 `json:"dial_timeout"`
	MaxIdleConnsPerHost int   `json:"max_idle_conns_per_host"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Writer Writer `json:"writer"`
}

type Writer struct {
	Batch    int `json:"batch"`
	ChanSize int `json:"chan_size"`
}

// PrometheusPushStatus defines the observed state of PrometheusPush
type PrometheusPushStatus struct {
	LastPush metav1.Time `json:"lastPush"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PrometheusPush is the Schema for the prometheuspushes API
type PrometheusPush struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrometheusPushSpec   `json:"spec,omitempty"`
	Status PrometheusPushStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PrometheusPushList contains a list of PrometheusPush
type PrometheusPushList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrometheusPush `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrometheusPush{}, &PrometheusPushList{})
}
