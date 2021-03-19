
# xcluster

On top of [*metal-stack*](https://github.com/metal-stack) and [*kubebuilder*](https://github.com/kubernetes-sigs/kubebuilder), we built a minimal computer cluster which contains *metal-stack* resources. We would like to walk you through this process to show you *metal-stack* and share what we learnt about *kubebuilder* with you. We will assume you already went through [*kubebuiler book*](https://book.kubebuilder.io) and are looking for more hands-on examples.

## Architecture

We created two *CustomResourceDefinition* (CRD), `XCluster` and `XFirewall`, as shown in the following figure. `XCluster` represents the computer cluster which contains *metal-stack network* and `XFirewall`. `XFirewall` corresponds to *metal-stack firewall*. The circular arrows imply the nature of recociliation.

![architecture](hack/xcluster.drawio.svg)

## Demo

Clone the repo of [*mini-lab*](https://github.com/metal-stack/mini-lab) and *xcluster* in the same folder.

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

Then, in another terminal yet still in folder *mini-lab* (must!), do

``` bash
eval $(make dev-env) # for talking to metal-api in this shell
cd ../xcluster
```

Now you should be in folder *xcluster*. Then,

```bash
make
```

Then, check out your *xcluster-controller-manager* running alongside other *metal-stack* deployments.

```bash
kubectl get deployment -A
```

Then, deploy your *xcluster*.

```bash
kubectl apply -f config/samples/xcluster.yaml
```

Check out your brand new *custom resources*.

```bash
kubectl get xcluster,xfirewall -A
```

Then go back to the previous terminal where you did

```bash
docker-compose run metalctl machine ls
```

Repeat the command and you should see a *metal-stack* firewall running.

## kubebuilder markers for CRD

*kubebuilder* provides lots of handful markers. Here are some examples:

1. API Resource Type

   ``` go
   // +kubebuilder:object:root=true
   ```

   The *go* `struct` under this marker will be an *API resource type* in the url. For example, the url path to `XCluster` instance *myxcluster* would be

   ```url
   /apis/cluster.www.x-cellent.com/v1/namespaces/myns/xclusters/myxcluster
   ```

1. API Subresource

   ```go
   // +kubebuilder:subresource:status
   ```

   The *go* `sturct` under this marker contains *API subresource* status. For the last example, the url path to the status of the instance would be:

   ```url
   /apis/cluster.www.x-cellent.com/v1/namespaces/myns/xclusters/myxcluster/status
   ```

1. Terminal Output

   ```go
   // +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.ready`
   ```

   This specifies an extra column of output on terminal when you do `kubetctl get`.

## metal-api

[*metal-api*](https://github.com/metal-stack/metal-api) manages all *metal-stack* resources, including machine, firewall, switch, OS image, IP, network and more. They are constructs which enable you to build a data center. You can try it out on *mini-lab*, where we built this demo project. In this project, *metal-api* does the real job. It allocates the network and creates the firewall, fulfiliing what you wish in the [**xcluster.yaml**](https://github.com/LimKianAn/xcluster/blob/main/config/samples/xcluster.yaml).

## Wire up metal-api client metalgo.Driver

`metalgo.Driver` is the client in *go* code for talking to *metal-api*. To enable both controllers of `XCluster` and `XFirewall` to do that, we created a `metalgo.Driver` named `metalClient` and set field `Driver` of both controllers as shown in the following snippet from [**main.go**](https://github.com/LimKianAn/xcluster/blob/main/main.go), .

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

With the following lines in [**xcluster_controller.go**](https://github.com/LimKianAn/xcluster/blob/main/controllers/xcluster_controller.go) and the euivalent lines in [**xfirewall_controller.go**](https://github.com/LimKianAn/xcluster/blob/main/controllers/xfirewall_controller.go) (in our case overlapped), *kubebuiler* generates [**role.yaml**](https://github.com/LimKianAn/xcluster/blob/main/config/rbac/role.yaml) and wire up everything for your *xcluster-controller-manager* pod when you do `make deploy`. The `verbs` are the actions your pod is allowed to perform on the `resources`, which are `xclusters` and `xfirewalls` in our case.
```go
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.www.x-cellent.com,resources=xfirewalls/status,verbs=get;update;patch
```

## Finalizer

When you want to do some clean-up before *api-server* deletes your resource in no time upon `kubectl delete`, *finalizer* comes in handy. *Finalizer* is a string. For example, the *finalizer* of `XCluster` in [**xcluster_types.go**](https://github.com/LimKianAn/xcluster/blob/main/api/v1/xcluster_types.go):

`const XClusterFinalizer = "xcluster.finalizers.cluster.www.x-cellent.com"`

The *api-server* will not delete the instance before its *finalizer*s are all removed from the instance. For example, in [**xcluster_controller.go**](https://github.com/LimKianAn/xcluster/blob/main/controllers/xcluster_controller.go) we add the above finalier to the `XCluster` instance, so later when the instance is about to be deleted, the *api-server* can't delete the instance before we've freed the *metal-stack* network and then removed the finalizer from the instance:

```go
	resp, err := r.Driver.NetworkFind(&metalgo.NetworkFindRequest{
		ID:        &cl.Spec.PrivateNetworkID,
		Name:      &cl.Spec.Partition,
		ProjectID: &cl.Spec.ProjectID,
	})

	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list metal-stack networks: %w", err)
	}

	if len := len(resp.Networks); len > 1 {
		return ctrl.Result{}, fmt.Errorf("more than one network listed: %w", err)
	} else if len == 1 {
		if _, err := r.Driver.NetworkFree(cl.Spec.PrivateNetworkID); err != nil {
			return ctrl.Result{Requeue: true}, nil
		}
	}
	log.Info("metal-stack network freed")

	cl.RemoveFinalizer(clusterv1.XFirewallFinalizer)
	if err := r.Update(ctx, cl); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove xcluster finalizer: %w", err)
	}
	r.Log.Info("finalizer removed")
```

Likewise, in [**xfirewall_controller.go**](https://github.com/LimKianAn/xcluster/blob/main/controllers/xfirewall_controller.go) we add the finalizer to `XFirewall` instance. Likewise, the *api-server* can't delete the instance before we clean up the underlying *metal-stack* firewall and then remove the finalizer from the instance:

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

## func errors.IsNotFound and client.IgnoreNotFound

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

If we can do nothing against the error **the instance not found**, we might simply stop the reconciliation without requeueing the request as follows:

```go
	cl := &clusterv1.XCluster{}
	if err := r.Get(ctx, req.NamespacedName, cl); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
```

## Exponential Back-Off

As far as requeue is concerned, returning `ctrl.Result{}, err` and `ctrl.Result{Requeue: true}, nil` are the same as shown in this [`if`](https://github.com/kubernetes-sigs/controller-runtime/blob/0fcf28efebc9a977c954f00d40af966d6a4aeae3/pkg/internal/controller/controller.go#L256) clause and this [`else if`](https://github.com/kubernetes-sigs/controller-runtime/blob/0fcf28efebc9a977c954f00d40af966d6a4aeae3/pkg/internal/controller/controller.go#L271) clause in the source code. Moreover, exponential back-off can be observed in the source code where dependencies of [controller](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.5.0/pkg/controller/controller.go#L90) are set and where [`func workqueue.DefaultControllerRateLimiter`](https://github.com/kubernetes/client-go/blob/0b19784585bd0a0ee5509855829ead81feaa2bdc/util/workqueue/default_rate_limiters.go#L39) is defined.

## ControllerReference

ControllerReference is a kind of `OwnerReference` that enables the garbage collection of the owned instance (`XFirewall`) when the owner instance (`XCluster`) is deleted. We demonstrate that in **xcluster_controller.go** by using the function `SetControllerReference`.

```go
		if err := controllerutil.SetControllerReference(cl, fw, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set the owner reference of the XFirewall: %w", err)
		}
```

Since `XCluster` owns `XFirewall` instance, we have to inform the manager that it should reconciling `XCluster` upon any change of an `XFirewall` instance:

```go
func (r *XClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.XCluster{}).
		Owns(&clusterv1.XFirewall{}).
		Complete(r)
}
```

## Wrap-up

Check out the code in this project for more details. If you want a fully-fledged implementation, stay tuned! Our *cluster-api-provider-metalstack* is on the way. If you want more blog posts about *metal-stack* and *kubebuilder*, let us know! Special thanks go to [*Grigoriy Mikhalkin*](https://github.com/GrigoriyMikhalkin).
