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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1 "github.com/LimKianAn/xcluster/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// XClusterReconciler reconciles a XCluster object
type XClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters/status,verbs=get;update;patch

func (r *XClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("xcluster", req.NamespacedName)

	// Fetch the XCluster in the Request.
	cl := &clusterv1.XCluster{}
	if err := r.Get(ctx, req.NamespacedName, cl); err != nil {
		log.Error(err, "err while fetching the XCluster")
		// blog: Why IgnoreNotFound? How Kubebuilder handles error?
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// No other resources are managed by our XCluster, so we don't have to consider the finalizer. // blog

	fw := &clusterv1.XFirewall{}
	if err := r.Get(ctx, req.NamespacedName, fw); err != nil {
		// errors other than `NotFound`
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("error while fetching the firewall: %w", err)
		}

		fw.ObjectMeta = cl.Spec.XFirewallTemplate.ObjectMeta
		fw.Spec = cl.Spec.XFirewallTemplate.Spec
		if err := r.Create(ctx, fw); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while creating the firewall: %w", err)
		}
	}

	// Skip update. blog

	cl.Status.Ready = fw.Status.Ready
	if err := r.Update(ctx, cl); err != nil {
		return ctrl.Result{}, fmt.Errorf("error while updating the status of the XCluster: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *XClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.XCluster{}).
		Complete(r)
}
