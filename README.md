
# xcluster, a minimal computer cluster

## Goal

On top of [*metal-stack*](https://github.com/metal-stack) and [*kubebuilder*](https://github.com/kubernetes-sigs/kubebuilder), we built a minimal computer cluster which contains a *metal-stack* network and a *metal-stack* firewall. We would like to walk you through this process to show you *metal-stack* and share what we learnt about *kubebuilder* with you.

## CustomResource XCluster and XFirewall

In this post, we will assume you already went through [*kubebuiler book*](https://book.kubebuilder.io) and are looking for more hands-on examples. We created two *CustomResource* (*CR*): `XCluster` and `XFirewall`. `XCluster` represents the computer cluster and `XCluster` corresponds to *metal-stack* resource *firewall*. We would like to keep it simple. If you want a fully-fledged implementation, stay tuned! Our *cluster-api-provider-metalstack* is on the way.

## Demo

Clone the repo of *mini-lab* and *xcluster* in the same folder.

```console
├── mini-lab
└── xcluster
```

Download the prerequisite of [*mini-lab*](https://github.com/metal-stack/mini-lab#requirements). Then,

```bash
cd mini-lab
make
```

It's going to take some time to finish. From time to tiem, do

```bash
docker-compose run metalctl machine ls
```

Till you see **Waiting** under **LAST EVENT** as follows:

```console
ID                                          LAST EVENT   WHEN     AGE  HOSTNAME  PROJECT  SIZE          IMAGE  PARTITION
e0ab02d2-27cd-5a5e-8efc-080ba80cf258        Waiting      8s                               v1-small-x86         vagrant
2294c949-88f6-5390-8154-fa53d93a3313        Waiting      8s                               v1-small-x86         vagrant
```

Then, in another terminal yet still in `mini-lab`, do

``` bash
eval $(make dev-env) # for talking to metal-api in this shell
cd ../xcluster
```

Now you should be in the folder of *xcluster*. Then,

```bash
make install
kubectl apply -f config config/samples/cluster_v1_xcluster.yaml
make run
```

Go back to the previous terminal where you did

```bash
docker-compose run metalctl machine ls
```

You should see a *metal-stack* firewall running.

## kubebuilder markers for CustomerResourceDefinition (CRD)

*kubebuilder* provides lots of handful markers. Here are some examples:

``` go
// +kubebuilder:object:root=true
```

This denotes the following *go* `struct` will be the *API resource* in the url: `/apis/cluster.www.x-cellent.com/v1/namespaces/myns/xclusters/mycluster`

```go
// +kubebuilder:subresource:status
```

This denotes the following *go* `sturct` contains a *API subresource*: `/apis/cluster.www.x-cellent.com/v1/namespaces/myns/xclusters/mycluster/status`

```go
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.ready`
```

This specifies the columns of the results on the terminal when you do `kubetctl get`.

## metal-api

[*metal-api*](https://github.com/metal-stack/metal-api) manages all *metal-stack* resources, including machine, firewall, switch, OS image, IP, network and more. They are constructs which enable you to build a data center. You can try it out on *mini-lab*. It's also where we built this demo project.

## Wire up metal-api client metalgo.Driver

`metalgo.Driver` is the client in *go* code land for talking to *metal-api*. To enable both controllers of `XCluster` and `XFirewall` to do that, we created a `metalgo.Driver` named `metalClient` and set the `Driver` of the controllers as shown in the following snippet from **main.go**, .

```go
	if err = (&controllers.XClusterReconciler{
		Client: mgr.GetClient(),
		Driver: metalClient,
		Log:    ctrl.Log.WithName("controllers").WithName("XCluster"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "XCluster")
		os.Exit(1)
	}
```

## Role-based access control (RBAC)

With the following lines in **xcluster_controller.go** and the euivalent lines in **xfirewall_controller.go** (in our case overlapped), *kubebuiler* generates **config/rbac/role.yaml** and wire up everything for your manager pod when you do `make deploy`.

```go
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls/status,verbs=get;update;patch
```

## Finalizer

When you want to do some clean-up before *api-server* deletes your resource upon `kubectl delete` in no time, *finalizer* comes in handy. It's just a string. For example, the finalizer of `XCluster` in **xcluster_types.go**:

`const XClusterFinalizer = "xcluster.finalizers.cluster.www.x-cellent.com"`

The *api-server* will not delete the very instance before its *finalizer*s are all removed from the instance. For example, in **xcluster_controller.go** we add the above finalier to the `XCluster` instance, so later when the instance is about to be deleted, the *api-server* can't delete the instance before we've freed the *metal-stack* network and then removed the finalizer from the instance:

```go
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
```

Likewise, in **xfirewall_controller.go** we add an finalizer to the `XFirewall` instance. Likewise, the *api-server* can't delete the instance before we clean up the underlying *metal-stack* firewall and then remove the finalizer from the instance:

```go
func (r *XFirewallReconciler) DeleteFirewall(ctx context.Context, fw *clusterv1.XFirewall, log logr.Logger) (ctrl.Result, error) {
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
```

## Functions errors.IsNotFound and client.IgnoreNotFound

When you have different handlers depending on whether the error is **the instance not found**, you can consider using `errors.IsNotFound(err)` as follows from **xcluster_controller.go**:

```go
	fw := &clusterv1.XFirewall{}
	if err := r.Get(ctx, req.NamespacedName, fw); err != nil {
		// errors other than `NotFound`
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to fetch XFirewall instance: %w", err)
		}

		// Create XFirewall instance
		fw = cl.ToXFirewall()
```

Sometimes if we can do nothing against the error **the instance not found**, we might simply stop the reconciliation without requeueing the request as follows:

```go
	cl := &clusterv1.XCluster{}
	if err := r.Get(ctx, req.NamespacedName, cl); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
```

## Exponential Back-Off

As far as requeue is concerned, returning `ctrl.Result{}, err` and `ctrl.Result{Requeue: true}, nil` are the same as shown in the first `if` clause and the second `else if` clause in the [source](https://github.com/kubernetes-sigs/controller-runtime/blob/0fcf28efebc9a977c954f00d40af966d6a4aeae3/pkg/internal/controller/controller.go#L256). Moreover, exponential back-off can be observed where dependencies of a [controller](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.5.0/pkg/controller/controller.go#L90) are set and in the definition of [`workqueue.DefaultControllerRateLimiter`](https://github.com/kubernetes/client-go/blob/0b19784585bd0a0ee5509855829ead81feaa2bdc/util/workqueue/default_rate_limiters.go#L39).

## ControllerReference

ControllerReference is a kind of `OwnerReference` that enables the garbage collection of the owned instance (`XFirewall`) when the owner instance (`XCluster`) is deleted. We demonstrate that in **xcluster_controller.go** by using the function `SetControllerReference`.

```go
		if err := controllerutil.SetControllerReference(cl, fw, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set the owner reference of the XFirewall: %w", err)
		}
```

Since `XClusterController` owns some `XFirewall` instances, we have to inform the manager to invoke the function `Reconcile` upon any change of an `XFirewall` instance:

```go
func (r *XClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.XCluster{}).
		Owns(&clusterv1.XFirewall{}).
		Complete(r)
}
```

## Wrap-up

Check out the code in this project for more details. Let us know if you want more of *metal-stack* and *kubebuilder*. Special thanks go to [*Grigoriy Mikhalkin*](https://github.com/GrigoriyMikhalkin).
