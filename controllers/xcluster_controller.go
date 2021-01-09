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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	metalgo "github.com/metal-stack/metal-go"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clusterv1 "github.com/LimKianAn/xcluster/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// XClusterReconciler reconciles a XCluster object
type XClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// blog: why do we need the driver?
	*metalgo.Driver
}

// blog: Explain the lines about xfirewalls
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls/status,verbs=get

func (r *XClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("xcluster", req.NamespacedName)

	// Fetch the XCluster in the Request.
	cl := &clusterv1.XCluster{}
	if err := r.Get(ctx, req.NamespacedName, cl); err != nil {
		// blog: Why IgnoreNotFound? How Kubebuilder handles error?
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// No other resources are managed by our XCluster, so we don't have to consider the finalizer. // blog

	if cl.Spec.PrivateNetworkID == "" {
		resp, err := r.NetworkAllocate(&metalgo.NetworkAllocateRequest{
			Name:        cl.Spec.Partition,
			PartitionID: cl.Spec.Partition,
			ProjectID:   cl.Spec.ProjectID,
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		cl.Spec.PrivateNetworkID = *resp.Network.ID
		if err := r.Update(ctx, cl); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while updating the privateNetworkID of the XCluster: %v", err)
		}
	}

	fw := &clusterv1.XFirewall{}
	if err := r.Get(ctx, req.NamespacedName, fw); err != nil {
		// blog: Explain the reason.
		cl.Status.Ready = false
		if err := r.Update(ctx, cl); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while updating the readiness of the XCluster: %v", err)
		}

		// errors other than `NotFound`
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error while fetching the firewall: %w", err)
		}

		fw.Name = cl.Name
		fw.Namespace = cl.Namespace
		fw.Spec.DefaultNetworkID = cl.Spec.XFirewallTemplate.Spec.DefaultNetworkID
		fw.Spec.Image = cl.Spec.XFirewallTemplate.Spec.Image
		fw.Spec.Size = cl.Spec.XFirewallTemplate.Spec.Size

		// cl is the owner of fw. Once cl is deleted, so is fw automatically. // blog
		if err := controllerutil.SetControllerReference(cl, fw, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while setting the owner reference of the XFirewall: %w", err)
		}

		if err := r.Create(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while creating the firewall: %w", err)
		}
	}

	// Skip update. blog

	cl.Status.Ready = fw.Status.Ready
	if err := r.Update(ctx, cl); err != nil {
		return ctrl.Result{}, fmt.Errorf("error while updating the readiness of the XCluster: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *XClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.XCluster{}).
		Owns(&clusterv1.XFirewall{}). // blog: Explain
		Complete(r)
}
