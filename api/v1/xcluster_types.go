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

// XClusterSpec defines the desired state of XCluster
type XClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Partition is the physical location where the cluster will be created
	Partition string `json:"partition"`

	// ProjectID is the projectID of the project in which K8s cluster should be deployed
	ProjectID string `json:"projectID"`

	XFirewallTemplate XFirewallTemplate `json:"xFirewallTemplate,omitempty"`
}

type XFirewallTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              XFirewallSpec `json:"spec,omitempty"`
}

// XClusterStatus defines the observed state of XCluster
type XClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true

// XCluster is the Schema for the xclusters API
type XCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XClusterSpec   `json:"spec,omitempty"`
	Status XClusterStatus `json:"status,omitempty"`
}

func (fw *XCluster) IsBeingDeleted() bool {
	return !fw.ObjectMeta.DeletionTimestamp.IsZero()
}

// XClusterFinalizer is for the resources managed by XCluster. // blog
const XClusterFinalizer = "xcluster.finalizers.cluster.www.x-cellent.com"

func (fw *XCluster) AddFinalizer(finalizer string) {
	fw.ObjectMeta.Finalizers = append(fw.ObjectMeta.Finalizers, finalizer)
}
func (fw *XCluster) HasFinalizer(finalizer string) bool {
	return containsElem(fw.ObjectMeta.Finalizers, finalizer)
}
func (fw *XCluster) RemoveFinalizer(finalizer string) {
	fw.ObjectMeta.Finalizers = removeElem(fw.ObjectMeta.Finalizers, finalizer)
}

// +kubebuilder:object:root=true

// XClusterList contains a list of XCluster
type XClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []XCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&XCluster{}, &XClusterList{})
}
