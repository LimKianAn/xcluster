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
	Driver *metalgo.Driver
}

// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls/status,verbs=get;update;patch

func (r *XFirewallReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("xfirewall", req.NamespacedName)

	// Fetch XFirewall instance
	fw := &clusterv1.XFirewall{}
	if err := r.Get(ctx, req.NamespacedName, fw); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if fw.IsBeingDeleted() {
		// Resetting the states of the underlying raw machine before XFirewall is deleted on API-server.
		return r.DeleteMetalStackFirewall(ctx, fw, log)
	}

	// Add finalizer if none.
	if !fw.HasFinalizer(clusterv1.XFirewallFinalizer) {
		fw.AddFinalizer(clusterv1.XFirewallFinalizer)
		if err := r.Update(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update XFirewall finalizer: %w", err)
		}
		r.Log.Info("finalizer added")
	}

	if fw.Spec.MachineID == "" {
		if err := r.CreateMetalStackFirewall(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create metal-stack firewall: %w", err)
		}
		r.Log.Info("metal-stack firewall created")
	}

	return r.UpdateAndCheckFirewallStatus(ctx, fw, log)
}

func (r *XFirewallReconciler) CreateMetalStackFirewall(ctx context.Context, fw *clusterv1.XFirewall) error {
	cl := &clusterv1.XCluster{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: fw.Namespace,
		Name:      fw.Name,
	}, cl); err != nil {
		return fmt.Errorf("failed to fetch owner XCluster instance: %w", err)
	}

	resp, err := r.Driver.FirewallCreate(&metalgo.FirewallCreateRequest{
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
		return fmt.Errorf("failed to create metal-stack firewall: %w", err)
	}

	fw.Spec.MachineID = *resp.Firewall.ID
	if err := r.Update(ctx, fw); err != nil {
		return fmt.Errorf("failed to update XFirewall machine-ID: %w", err)
	}

	return nil
}

func (r *XFirewallReconciler) DeleteMetalStackFirewall(ctx context.Context, fw *clusterv1.XFirewall, log logr.Logger) (ctrl.Result, error) {
	if _, err := r.Driver.MachineDelete(fw.Spec.MachineID); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete firewall: %w", err)
	}
	log.Info("states of the machine managed by XFirewall reset")

	fw.RemoveFinalizer(clusterv1.XFirewallFinalizer)
	if err := r.Update(ctx, fw); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove XFirewall finalizer: %w", err)
	}
	r.Log.Info("finalizer removed")

	return ctrl.Result{}, nil
}

func (r *XFirewallReconciler) UpdateAndCheckFirewallStatus(ctx context.Context, fw *clusterv1.XFirewall, log logr.Logger) (ctrl.Result, error) {
	resp, err := r.Driver.FirewallGet(fw.Spec.MachineID)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while fetching the underlying raw firewall: %w", err)
	}
	isReady := *resp.Firewall.Allocation.Succeeded
	if fw.Status.Ready != isReady {
		fw.Status.Ready = isReady
		if err := r.Update(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update the status of the firewall: %w", err)
		}
	}

	if !fw.Status.Ready {
		return ctrl.Result{Requeue: true}, nil
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
