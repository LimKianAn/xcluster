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
	Driver *metalgo.Driver
}

// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls/status,verbs=get;update;patch

func (r *XClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("xcluster", req.NamespacedName)

	// Fetch XCluster instance
	cl := &clusterv1.XCluster{}
	if err := r.Get(ctx, req.NamespacedName, cl); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cl.IsBeingDeleted() {
		return r.FreeMetalStackNetwork(ctx, cl, log)
	}

	// Add finalizer if none.
	if !cl.HasFinalizer(clusterv1.XFirewallFinalizer) {
		cl.AddFinalizer(clusterv1.XFirewallFinalizer)
		if err := r.Update(ctx, cl); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update XFirewall finalizer: %w", err)
		}
		r.Log.Info("finalizer added")
	}

	if cl.Spec.PrivateNetworkID == "" {
		resp, err := r.Driver.NetworkAllocate(&metalgo.NetworkAllocateRequest{
			Name:        cl.Spec.Partition,
			PartitionID: cl.Spec.Partition,
			ProjectID:   cl.Spec.ProjectID,
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to allocating network: %w", err)
		}
		log.Info("private network-ID allocated")

		cl.Spec.PrivateNetworkID = *resp.Network.ID
		if err := r.Update(ctx, cl); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update the privateNetworkID of the XCluster: %v", err)
		}
	}

	fw := &clusterv1.XFirewall{}
	if err := r.Get(ctx, req.NamespacedName, fw); err != nil {
		// errors other than `NotFound`
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to fetch XFirewall instance: %w", err)
		}

		// Create XFirewall instance
		fw = cl.ToXFirewall()

		// cl is the owner of fw. Once cl is deleted, so is fw automatically.
		if err := controllerutil.SetControllerReference(cl, fw, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set the owner reference of the XFirewall: %w", err)
		}

		if err := r.Create(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create the firewall: %w", err)
		}
	}
	if !fw.Status.Ready {
		return ctrl.Result{Requeue: true}, nil
	}

	cl.Status.Ready = true
	if err := r.Update(ctx, cl); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update the readiness of the XCluster: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *XClusterReconciler) FreeMetalStackNetwork(ctx context.Context, cl *clusterv1.XCluster, log logr.Logger) (ctrl.Result, error) {
	if _, err := r.Driver.NetworkFree(cl.Spec.PrivateNetworkID); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to free metal-stack network: %w", err)
	}
	log.Info("metal-stack network freed")

	cl.RemoveFinalizer(clusterv1.XFirewallFinalizer)
	if err := r.Update(ctx, cl); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove XCluster finalizer: %w", err)
	}
	r.Log.Info("finalizer removed")

	return ctrl.Result{}, nil
}

func (r *XClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.XCluster{}).
		Owns(&clusterv1.XFirewall{}).
		Complete(r)
}
