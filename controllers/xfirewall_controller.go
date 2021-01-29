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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1 "github.com/LimKianAn/xcluster/api/v1"
)

// XFirewallReconciler reconciles a XFirewall object
type XFirewallReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// blog: Add the client to the reconciler to interact with `metal-api`.
	Driver *metalgo.Driver
}

// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls/status,verbs=get;update;patch

func (r *XFirewallReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("xfirewall", req.NamespacedName)

	// Fetch the XFirewall in the Request.
	fw := &clusterv1.XFirewall{}
	if err := r.Get(ctx, req.NamespacedName, fw); err != nil {
		// blog: why?
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Resetting the states of the underlying raw machine before XFirewall is deleted on API-server. // blog
	// TODO: Move to seperate function
	if fw.IsBeingDeleted() {
		log.Info("resetting the states of the machine managed by XFirewall")

		if _, err := r.MachineDelete(fw.Spec.MachineID); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while deleting the underlying raw firewall: %w", err)
		}

		fw.RemoveFinalizer(clusterv1.XFirewallFinalizer)
		if err := r.Update(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while removing the finalizers of the XFirewall: %w", err)
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if none.
	if !fw.HasFinalizer(clusterv1.XFirewallFinalizer) {
		fw.AddFinalizer(clusterv1.XFirewallFinalizer)
		if err := r.Update(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while updating the finalizer of the XFirewall: %w", err)
		}
		r.Log.Info("finalizer added")
	}

	if fw.Spec.MachineID == "" {
		cl := &clusterv1.XCluster{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: fw.Namespace,
			Name:      fw.Name,
		}, cl); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to fetch the corresponding cluster: %w", err)
		}

		resp, err := r.FirewallCreate(&metalgo.FirewallCreateRequest{
			MachineCreateRequest: metalgo.MachineCreateRequest{
				Description:   "",
				Name:          fw.Name,
				Hostname:      fw.Name + "-firewall",
				Size:          fw.Spec.Size,
				Project:       cl.Spec.ProjectID,
				Partition:     cl.Spec.Partition,
				Image:         fw.Spec.Image,
				SSHPublicKeys: []string{},
				Networks:      toNetworks(fw.Spec.DefaultNetworkID, cl.Spec.PrivateNetworkID),
				UserData:      "",
				Tags:          []string{},
			},
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create the firewall: %w", err)
		}

		fw.Spec.MachineID = *resp.Firewall.ID
		if err := r.Update(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while updating the machine-ID of the firewall: %w", err)
		}
	}

	fw.Status.Ready = true
	if err := r.Update(ctx, fw); err != nil {
		return ctrl.Result{}, fmt.Errorf("error while updating the status of the firewall: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *XFirewallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.XFirewall{}).
		Complete(r)
}

func toNetworks(ss ...string) (networks []metalgo.MachineAllocationNetwork) {
	for _, s := range ss {
		networks = append(networks, metalgo.MachineAllocationNetwork{
			NetworkID:   s,
			Autoacquire: true,
		})
	}
	return
}
