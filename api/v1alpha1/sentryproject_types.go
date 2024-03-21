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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=Ignore;Update;Fail
type ConflictPolicy string

const (
	// Ignore means the operator will ignore the conflict and continue
	Ignore ConflictPolicy = "Ignore"

	// Update means the operator will update the project in Sentry
	Update ConflictPolicy = "Update"

	// Fail means the operator will fail the project creation in Sentry
	Fail ConflictPolicy = "Fail"
)

// SentryProjectSpec defines the desired state of SentryProject
type SentryProjectSpec struct {
	// Name is the human-readable name of the project in Sentry
	// +optional
	Name string `json:"name,omitempty"`

	//+kubebuilder:validation:MinLength=4
	//+kubebuilder:validation:MaxLength=50
	//+kubebuilder:validation:Pattern=^[a-z0-9-]+$

	// Slug is the unique identifier for the project in Sentry
	Slug string `json:"slug"`

	//+kubebuilder:validation:MinLength=4
	//+kubebuilder:validation:MaxLength=50
	//+kubebuilder:validation:Pattern=^[a-z0-9-]+$

	// Team is the team that owns the project in Sentry
	Team string `json:"team"`

	// Platform is the platform of the project in Sentry
	Platform string `json:"platform"`

	//+kubebuilder:validation:MinLength=4
	//+kubebuilder:validation:MaxLength=50
	//+kubebuilder:validation:Pattern=^[a-z0-9-]+$

	// Organization is the organization that owns the project in Sentry
	Organization string `json:"organization"`

	// ConflictPolicy is the policy to apply when a conflict is detected
	// +optional
	ConflictPolicy ConflictPolicy `json:"conflictPolicy,omitempty"`
}

// SentryProjectStatus defines the observed state of SentryProject
type SentryProjectStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
	Slug       string      `json:"slug,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SentryProject is the Schema for the sentryprojects API
type SentryProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SentryProjectSpec   `json:"spec,omitempty"`
	Status SentryProjectStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SentryProjectList contains a list of SentryProject
type SentryProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SentryProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SentryProject{}, &SentryProjectList{})
}

func (sp *SentryProject) GetReadyCondition() *Condition {
	for _, c := range sp.Status.Conditions {
		if c.Type == Ready {
			return &c
		}
	}
	return nil
}

func (sp *SentryProject) IsProjectCreated() bool {
	cond := sp.GetReadyCondition()
	return cond != nil && sp.Status.Slug != ""
}
