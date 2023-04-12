/*
Copyright 2021 The Dapr Authors
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
	"strconv"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+genclient:noStatus
//+kubebuilder:object:root=true

// ExternalHTTPEndpoint describes an Dapr ExternalHTTPEndpoint type to allow for external service invocation.
type ExternalHTTPEndpoint struct {
	metav1.TypeMeta `json:",inline"`
	//+optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	//+optional
	Spec ExternalHTTPEndpointSpec `json:"spec,omitempty"`
	//+optional
	Auth `json:"auth,omitempty"`
	//+optional
	Scopes []string `json:"scopes,omitempty"`
}

// Kind returns the component kind.
func (ExternalHTTPEndpoint) Kind() string {
	return "ExternalHTTPEndpoint"
}

// ExternalHTTPEndpointSpec describes an access specification for allowing external service invocations.
type ExternalHTTPEndpointSpec struct {
	Allowed  []ExternalAPISpec `json:"allowed,omitempty"`
	Metadata []MetadataItem    `json:"metadata"`
}

// ExternalAPISpec describes the configuration for Dapr API communication with external services.
type ExternalAPISpec struct {
	BaseURL string `json:"baseUrl" validate:"required"`
	//+optional
	Headers map[string]string `json:"headers"`
	Name    string            `json:"name" validate:"required"`
	//+optional
	Protocol string `json:"protocol"`
}

// MetadataItem is a name/value pair for a metadata.
type MetadataItem struct {
	Name string `json:"name"`
	//+optional
	Value DynamicValue `json:"value,omitempty"`
	//+optional
	SecretKeyRef SecretKeyRef `json:"secretKeyRef,omitempty"`
}

// SecretKeyRef is a reference to a secret holding the value for the metadata item. Name is the secret name, and key is the field in the secret.
type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// Auth represents authentication details for the component.
type Auth struct {
	SecretStore string `json:"secretStore"`
}

//+kubebuilder:object:root=true

// ExternalHTTPEndpointList is a list of Dapr ExternalHTTPEndpoints.
type ExternalHTTPEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ExternalHTTPEndpoint `json:"items"`
}

// DynamicValue is a dynamic value struct for the component.metadata pair value.
type DynamicValue struct {
	v1.JSON `json:",inline"`
}

// String returns the string representation of the raw value.
// If the value is a string, it will be unquoted as the string is guaranteed to be a JSON serialized string.
func (d *DynamicValue) String() string {
	s := string(d.Raw)
	c, err := strconv.Unquote(s)
	if err == nil {
		s = c
	}
	return s
}
