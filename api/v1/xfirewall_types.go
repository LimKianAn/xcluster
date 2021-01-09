/*


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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// XFirewallSpec defines the desired state of XFirewall
type XFirewallSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DefaultNetworkID string `json:"defaultNetworkID,omitempty"`
	Image            string `json:"image,omitempty"`
	MachineID        string `json:"machineID,omitempty"`
	Size             string `json:"size,omitempty"`
}

// XFirewallStatus defines the observed state of XFirewall
type XFirewallStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.ready`

// XFirewall is the Schema for the xfirewalls API
type XFirewall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XFirewallSpec   `json:"spec,omitempty"`
	Status XFirewallStatus `json:"status,omitempty"`
}

func (fw *XFirewall) IsBeingDeleted() bool {
	return !fw.ObjectMeta.DeletionTimestamp.IsZero()
}

// XFirewallFinalizer is for the resources managed by XFirewall. // blog
const XFirewallFinalizer = "xfirewall.finalizers.cluster.www.x-cellent.com"

func (fw *XFirewall) AddFinalizer(finalizer string) {
	fw.ObjectMeta.Finalizers = append(fw.ObjectMeta.Finalizers, finalizer)
}
func (fw *XFirewall) HasFinalizer(finalizer string) bool {
	return containsElem(fw.ObjectMeta.Finalizers, finalizer)
}
func (fw *XFirewall) RemoveFinalizer(finalizer string) {
	fw.ObjectMeta.Finalizers = removeElem(fw.ObjectMeta.Finalizers, finalizer)
}

func containsElem(ss []string, s string) bool {
	for _, elem := range ss {
		if elem == s {
			return true
		}
	}
	return false
}

func removeElem(ss []string, s string) (out []string) {
	for _, elem := range ss {
		if elem == s {
			continue
		}
		out = append(out, elem)
	}
	return
}

// +kubebuilder:object:root=true

// XFirewallList contains a list of XFirewall
type XFirewallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []XFirewall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&XFirewall{}, &XFirewallList{})
}
