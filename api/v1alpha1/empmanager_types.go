/*
Copyright 2024.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PENDING_STATE  = "PENDING"
	CREATING_STATE = "CREATING"
	CREATED_STATE  = "CREATED"
	ERROR_STATE    = "ERROR"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EmpManagerSpec defines the desired state of EmpManager
type EmpManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name is the name of Employee Manager instance we are going to create
	Name string `json:"name"`

	// Replicas needed for the server instances
	Replicas int32 `json:"replicas"`

	// Image needed for the server instances
	Image string `json:"image"`

	// Spec contains the namespace/deployement/service as yaml
	Spec string `json:"spec"`
}

// EmpManagerStatus defines the observed state of EmpManager
type EmpManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State string `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// EmpManager is the Schema for the empmanagers API
type EmpManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmpManagerSpec   `json:"spec,omitempty"`
	Status EmpManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmpManagerList contains a list of EmpManager
type EmpManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmpManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmpManager{}, &EmpManagerList{})
}
