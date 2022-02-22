package controllers

import (
	"github.com/go-logr/logr"
	"github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NodePoolChangedPredicates filters node pool change events for nodes
// before enqueuing the keys.
type NodePoolChangedPredicates struct {
	predicate.Funcs
	log logr.Logger
}

// Update implements default UpdateEvent filter for validating node pool change.
func (n NodePoolChangedPredicates) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		n.log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		n.log.Error(nil, "Update event has no new object to update", "event", e)
		return false
	}
	// process update event if node pool changed
	oldLabel := e.ObjectOld.GetLabels()
	newLabel := e.ObjectNew.GetLabels()
	return oldLabel[v1alpha1.LabelCurrentNodePool] != newLabel[v1alpha1.LabelCurrentNodePool]
}
