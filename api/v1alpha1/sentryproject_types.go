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

// +kubebuilder:validation:Enum=Created;Pending;Failed;Deleted;Conflict
type SentryProjectCrStatus string

const (
	// Created means the project has been created in Sentry
	Created SentryProjectCrStatus = "Created"

	// Pending means the project is pending creation in Sentry
	Pending SentryProjectCrStatus = "Pending"

	// Failed means the project creation has failed in Sentry
	Failed SentryProjectCrStatus = "Failed"

	// Deleted means the project has been deleted in Sentry
	Deleted SentryProjectCrStatus = "Deleted"

	// Conflict means the project creation has failed due to a conflict in Sentry
	Conflict SentryProjectCrStatus = "Conflict"
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

	// DSN is the Data Source Name for the project in Sentry
	// +optional
	DSN string `json:"dsn,omitempty"`
}

// SentryProjectStatus defines the observed state of SentryProject
type SentryProjectStatus struct {
	State   SentryProjectCrStatus `json:"state,omitempty"`
	Message string                `json:"message,omitempty"`
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
