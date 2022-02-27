package controllers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ravenv1alpha1 "github.com/openyurtio/raven-controller-manager/pkg/ravencontroller/apis/raven/v1alpha1"
)

// NodeChangedPredicates filters certain notable change events for nodes
// before enqueuing the keys.
type NodeChangedPredicates struct {
	predicate.Funcs
	log logr.Logger
}

// Update implements default UpdateEvent filter for validating notable change.
// Notable change including:
// * NodeReady condition change;
// * Gateway label change;
func (n NodeChangedPredicates) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		n.log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		n.log.Error(nil, "Update event has no new object to update", "event", e)
		return false
	}
	oldObj := e.ObjectOld.(*corev1.Node)
	newObj := e.ObjectNew.(*corev1.Node)
	// check if the gateway label changed
	gatewayChanged := func(oldObj, newObj *corev1.Node) bool {
		oldLabel := e.ObjectOld.GetLabels()
		newLabel := e.ObjectNew.GetLabels()
		if oldLabel == nil {
			oldLabel = make(map[string]string)
		}
		if newLabel == nil {
			newLabel = make(map[string]string)
		}
		return oldLabel[ravenv1alpha1.LabelCurrentGateway] != newLabel[ravenv1alpha1.LabelCurrentGateway]
	}
	// check if NodeReady condition changed
	statusChanged := func(oldObj, newObj *corev1.Node) bool {
		return isNodeReady(*oldObj) != isNodeReady(*newObj)
	}
	return gatewayChanged(oldObj, newObj) || statusChanged(oldObj, newObj)
}
