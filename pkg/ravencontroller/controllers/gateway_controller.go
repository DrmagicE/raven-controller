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
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	ravenv1alpha1 "github.com/openyurtio/raven-controller-manager/pkg/ravencontroller/apis/raven/v1alpha1"
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
		if !apierrs.IsNotFound(err) {
			log.Error(err, "unable to fetch Gateway")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// get all managed nodes
	var nodeList corev1.NodeList
	nodeSelector, err := labels.Parse(fmt.Sprintf(ravenv1alpha1.LabelCurrentGateway+"=%s", gw.Name))
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.List(ctx, &nodeList, &client.ListOptions{
		LabelSelector: nodeSelector,
	})
	if err != nil {
		err = fmt.Errorf("unable to list nodes: %s", err)
		return ctrl.Result{}, err
	}

	// 1. try to select an active endpoint if possible
	err = r.setActiveEndpoint(nodeList, &gw)
	if err != nil {
		log.Error(err, "unable to set active endpoint")
		return ctrl.Result{}, err
	}
	if log.V(4).Enabled() {
		if gw.Status.ActiveEndpoint != nil {
			log.V(4).Info("active endpoint selected", "activeEndpoint", gw.Status.ActiveEndpoint.NodeName)
		} else {
			log.V(4).Info("unable to select active endpoint")
		}
	}

	// 2. get subnet list of all nodes managed by the Gateway
	var subnets []string
	for _, v := range nodeList.Items {
		subnets = append(subnets, v.Spec.PodCIDR)
	}
	if err != nil {
		log.Error(err, "unable to get subnets")
		return ctrl.Result{}, err
	}
	log.V(4).Info("get subnet list", "subnets", subnets)
	gw.Status.Subnets = subnets

	// 3. add region labels to relevant nodes.
	err = r.ensureRegionLabelForNodes(ctx, nodeList, &gw)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Status().Update(ctx, &gw)
	if err != nil {
		log.Error(err, "unable to Update Gateway.status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// ensureTopologyLabels ensures the topology label in the labels is consist with the given topologies.
func ensureTopologyLabels(labels map[string]string, topologies map[string]string) {
	for k := range labels {
		if ravenv1alpha1.IsTopologyLabel(k) {
			delete(labels, k)
		}
	}
	for k, v := range topologies {
		labels[k] = v
	}
}

// setActiveEndpoint will trys to select an active Endpoint and set it into gw.Status.ActiveEndpoint.
// If the current active endpoint remains competent, then we don't change it.
// Otherwise, try to select a new one.
func (r *GatewayReconciler) setActiveEndpoint(nodeList corev1.NodeList, gw *ravenv1alpha1.Gateway) (err error) {
	// get all ready nodes referenced by endpoints
	readyNodes := make(map[string]corev1.Node)
	for _, v := range nodeList.Items {
		if isNodeReady(v) {
			readyNodes[v.Name] = v
		}
	}
	// checkActive return if the given ep can be elected as an active endpoint.
	checkActive := func(ep *ravenv1alpha1.Endpoint) bool {
		if ep == nil {
			return false
		}
		// check if the node status is ready
		if _, ok := readyNodes[ep.NodeName]; ok {
			var inList bool
			// check if ep is in the Endpoint list
			for _, v := range gw.Spec.Endpoints {
				if reflect.DeepEqual(v, *ep) {
					inList = true
					break
				}
			}
			return inList
		}
		return false
	}
	if checkActive(gw.Status.ActiveEndpoint) {
		return nil
	}
	gw.Status.ActiveEndpoint = nil
	// try to select an active endpoint.
	for _, v := range gw.Spec.Endpoints {
		if checkActive(&v) {
			gw.Status.ActiveEndpoint = &ravenv1alpha1.Endpoint{
				NodeName:   v.NodeName,
				PrivateIP:  v.PrivateIP,
				PublicIP:   v.PublicIP,
				NATEnabled: v.NATEnabled,
				Config:     map[string]string{},
			}
			for ck, cv := range v.Config {
				gw.Status.ActiveEndpoint.Config[ck] = cv
			}
			return
		}
	}
	return
}

func (r *GatewayReconciler) ensureRegionLabelForNodes(ctx context.Context, nodeList corev1.NodeList, gw *ravenv1alpha1.Gateway) error {
	topologies := make(map[string]string)
	for k, v := range gw.Labels {
		if ravenv1alpha1.IsTopologyLabel(k) {
			topologies[k] = gw.Labels[v]
		}
	}
	for k := range nodeList.Items {
		n := &nodeList.Items[k]
		if n.Labels == nil {
			n.Labels = make(map[string]string)
		}
		lb := n.Labels
		ensureTopologyLabels(lb, topologies)
		err := r.Update(ctx, n)
		if err != nil {
			return fmt.Errorf("unable to update node: %s", err)
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&ravenv1alpha1.Gateway{}).
		Watches(
			&source.Kind{Type: &corev1.Node{}},
			handler.EnqueueRequestsFromMapFunc(r.mapNodeToRequest),
			builder.WithPredicates(NodeChangedPredicates{log: r.Log}),
		).Complete(r)
}

// mapNodeToRequest maps the given Node object to reconcile.Request.
func (r *GatewayReconciler) mapNodeToRequest(object client.Object) []reconcile.Request {
	node := object.(*corev1.Node)
	gwName, ok := node.Labels[ravenv1alpha1.LabelCurrentGateway]
	if !ok || gwName == "" {
		return []reconcile.Request{}
	}
	var gw ravenv1alpha1.Gateway
	err := r.Get(context.TODO(), types.NamespacedName{
		Name: gwName,
	}, &gw)
	if apierrs.IsNotFound(err) {
		r.Log.Info("gateway not found", "name", gwName)
		return []reconcile.Request{}
	}
	if err != nil {
		r.Log.Error(err, "unable to get Gateway")
	}
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: "",
				Name:      gwName,
			},
		},
	}
}

// isNodeReady checks if the `node` is `corev1.NodeReady`
func isNodeReady(node corev1.Node) bool {
	_, nc := getNodeCondition(&node.Status, corev1.NodeReady)
	// GetNodeCondition will return nil and -1 if the condition is not present
	return nc != nil && nc.Status == corev1.ConditionTrue
}

// getNodeCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getNodeCondition(status *corev1.NodeStatus, conditionType corev1.NodeConditionType) (int, *corev1.NodeCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}
