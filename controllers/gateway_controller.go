/*
Copyright 2022.

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
	"github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ravenv1alpha1 "github.com/openyurtio/raven-controller-manager/api/v1alpha1"
)

const (
	// nodePoolField is the index field name for both Gateway and Endpoint Object.
	nodePoolField = ".spec.nodePool"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=raven.openyurt.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=raven.openyurt.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=raven.openyurt.io,resources=gateways/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps.openyurt.io,resources=nodepools,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.V(4).Info("started reconciling Gateway", "name", req.Name)
	defer func() {
		log.V(4).Info("finished reconciling Gateway", "name", req.Name)
	}()
	var gw ravenv1alpha1.Gateway
	if err := r.Get(ctx, req.NamespacedName, &gw); err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Error(err, "unable to fetch Gateway")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// ensure topology labels is consist with.spec.Topology
	if gw.Labels == nil {
		gw.Labels = make(map[string]string)
	}
	ensureTopologyLabels(gw.Labels, gw.Spec.Topologies)

	// reconcile endpoint and get the active endpoint if exists.
	activeEndpoint, err := r.reconcileEndpoint(ctx, &gw)
	if err != nil {
		log.Error(err, "unable to reconcile Endpoint")
		return ctrl.Result{}, err
	}
	// 2. get subnet list of all nodes managed by the Gateway
	subnets, err := r.getSubnets(ctx, &gw)
	if err != nil {
		log.Error(err, "unable to get subnets")
		return ctrl.Result{}, err
	}
	log.V(2).Info("get subnet list", "subnets", subnets)
	gw.Status.Subnets = subnets
	if activeEndpoint != nil {
		log.V(2).Info("active endpoint selected", "activeEndpoint", activeEndpoint.Name)
		gw.Status.ActiveEndpoint = &activeEndpoint.Spec
	} else {
		gw.Status.ActiveEndpoint = nil
	}

	err = r.Status().Update(ctx, &gw)
	if err != nil {
		log.Error(err, "unable to Update Gateway.status")
		return ctrl.Result{}, err
	}

	err = r.Update(ctx, &gw)
	if err != nil {
		log.Error(err, "unable to Update Gateway")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// ensureTopologyLabels ensures the label is consist with the given topology names.
func ensureTopologyLabels(labels map[string]string, topologies []ravenv1alpha1.Topology) {
	for k := range labels {
		if ravenv1alpha1.IsTopologyLabel(k) {
			delete(labels, k)
		}
	}
	for _, v := range topologies {
		labels[ravenv1alpha1.TopologyLabelName(v.Name)] = ""
	}
}

// getSubnets gets subnet list of nodes within the same nodepool as the Gateway
func (r *GatewayReconciler) getSubnets(ctx context.Context, gw *ravenv1alpha1.Gateway) (subnets []string, err error) {
	var nodes corev1.NodeList
	nodeSelector, err := labels.Parse(fmt.Sprintf(v1alpha1.LabelCurrentNodePool+"=%s", gw.Spec.NodePool))
	if err != nil {
		return
	}
	err = r.List(ctx, &nodes, &client.ListOptions{
		LabelSelector: nodeSelector,
	})
	if err != nil {
		err = fmt.Errorf("unable to list nodes: %s", err)
		return
	}
	for _, v := range nodes.Items {
		subnets = append(subnets, v.Spec.PodCIDR)
	}
	return
}

// reconcileEndpoint
func (r *GatewayReconciler) reconcileEndpoint(ctx context.Context, gw *ravenv1alpha1.Gateway) (activeEndpoint *ravenv1alpha1.Endpoint, err error) {
	// fetch endpoints belongs to the gateway
	var endpoints ravenv1alpha1.EndpointList
	err = r.List(context.TODO(), &endpoints, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(nodePoolField, gw.Spec.NodePool),
	})
	if err != nil {
		err = fmt.Errorf("unable to list Endpoint: %s", err)
		return
	}
	// construct active endpoint
	for k := range endpoints.Items {
		if endpoints.Items[k].Labels == nil {
			endpoints.Items[k].Labels = make(map[string]string)
		}
		lb := endpoints.Items[k].Labels
		ensureTopologyLabels(lb, gw.Spec.Topologies)
		isActive, ok := lb[ravenv1alpha1.LabelActiveEndpoint]
		if ok && isActive == "true" {
			// guarantee there is only one active endpoint
			if activeEndpoint == nil {
				activeEndpoint = endpoints.Items[k].DeepCopy()
			} else {
				delete(lb, ravenv1alpha1.LabelActiveEndpoint)
			}
		}
	}
	// if there is no active endpoint, try to select one
	if activeEndpoint == nil && len(endpoints.Items) != 0 {
		activeEndpoint = endpoints.Items[0].DeepCopy()
		activeEndpoint.Labels[ravenv1alpha1.LabelActiveEndpoint] = "true"
	}
	for _, v := range endpoints.Items {
		err = r.Update(ctx, &v)
		if err != nil {
			err = fmt.Errorf("unable to update Endpoint: %s", err)
			return
		}
	}
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set nodepool index for gateway
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &ravenv1alpha1.Gateway{}, nodePoolField, func(rawObj client.Object) []string {
		obj := rawObj.(*ravenv1alpha1.Gateway)
		if obj.Spec.NodePool == "" {
			return nil
		}
		return []string{obj.Spec.NodePool}
	}); err != nil {
		return err
	}
	// set nodepool index for endpoint
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &ravenv1alpha1.Endpoint{}, nodePoolField, func(rawObj client.Object) []string {
		obj := rawObj.(*ravenv1alpha1.Endpoint)
		if obj.Spec.NodePool == "" {
			return nil
		}
		return []string{obj.Spec.NodePool}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).For(&ravenv1alpha1.Gateway{}).
		Watches(
			&source.Kind{Type: &corev1.Node{}},
			handler.EnqueueRequestsFromMapFunc(r.mapNodeToRequest),
			builder.WithPredicates(NodePoolChangedPredicates{log: r.Log}),
		).Watches(
		&source.Kind{Type: &ravenv1alpha1.Endpoint{}},
		handler.EnqueueRequestsFromMapFunc(r.mapEndpointToRequest),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
	).Complete(r)
}

// mapNodeToRequest maps the given Node object to reconcile.Request.
func (r *GatewayReconciler) mapNodeToRequest(object client.Object) []reconcile.Request {
	node := object.(*corev1.Node)
	nodePool, ok := node.Labels[v1alpha1.LabelCurrentNodePool]
	if !ok || nodePool == "" {
		return []reconcile.Request{}
	}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(nodePoolField, nodePool),
	}
	// fetch related Gateways
	var gatewayList ravenv1alpha1.GatewayList
	err := r.List(context.TODO(), &gatewayList, listOps)
	if err != nil {
		// TODO log context
		r.Log.Error(err, "unable to fetch gateway")
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(gatewayList.Items))
	for i, item := range gatewayList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// mapEndpointToRequest maps the given Endpoint object to reconcile.Request.
func (r *GatewayReconciler) mapEndpointToRequest(object client.Object) []reconcile.Request {
	ep := object.(*ravenv1alpha1.Endpoint)
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(nodePoolField, ep.Spec.NodePool),
	}
	// fetch related Gateways
	var gatewayList ravenv1alpha1.GatewayList
	err := r.List(context.TODO(), &gatewayList, listOps)
	if err != nil {
		r.Log.Error(err, "unable to fetch gateway")
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(gatewayList.Items))
	for i, item := range gatewayList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
