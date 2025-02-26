/*
Copyright 2022 The Crossplane Authors.

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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ComputeParameters are the configurable fields of a Compute.
type ComputeParameters struct {
	ConfigurableField string         `json:"configurableField"`
	AWSConfig         AWSConfig      `json:"awsConfig"`
	InstanceConfig    InstanceConfig `json:"instanceConfig"`
}

type AWSConfig struct {
	Region string `json:"region"`
}

type InstanceConfig struct {
	InstanceName           string            `json:"instanceName"`
	InstanceType           string            `json:"instanceType"`
	InstanceAMI            string            `json:"instanceAMI"`
	InstanceDisk           string            `json:"instanceDisk"`
	InstanceSecurityGroups []string          `json:"instanceSecurityGroups"`
	InstanceTags           map[string]string `json:"instanceTags"`
}

// ComputeObservation are the observable fields of a Compute.
type ComputeObservation struct {
	ObservableField string `json:"observableField,omitempty"`
	State           string `json:"state"`
}

// A ComputeSpec defines the desired state of a Compute.
type ComputeSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ComputeParameters `json:"forProvider"`
}

// A ComputeStatus represents the observed state of a Compute.
type ComputeStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ComputeObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Compute is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,customcomputeprovider}
type Compute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComputeSpec   `json:"spec"`
	Status ComputeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComputeList contains a list of Compute
type ComputeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Compute `json:"items"`
}

// Compute type metadata.
var (
	ComputeKind             = reflect.TypeOf(Compute{}).Name()
	ComputeGroupKind        = schema.GroupKind{Group: Group, Kind: ComputeKind}.String()
	ComputeKindAPIVersion   = ComputeKind + "." + SchemeGroupVersion.String()
	ComputeGroupVersionKind = SchemeGroupVersion.WithKind(ComputeKind)
)

func init() {
	SchemeBuilder.Register(&Compute{}, &ComputeList{})
}
