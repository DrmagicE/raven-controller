package v1alpha1

import (
	"strings"
)

const (
	// LabelActiveEndpoint indicates if the current endpoint is active.
	LabelActiveEndpoint = "raven.openyurt.io/is-active-endpoint"

	// LabelTopologyKeyPrefix is the prefix of the labels representing topologies.
	LabelTopologyKeyPrefix = "topology.raven.openyurt.io/"

	// LabelCurrentGateway indicates which nodepool the node is currently belonging to
	LabelCurrentGateway = "raven.openyurt.io/gateway"
)

// IsTopologyLabel returns if the given label name is topology label.
func IsTopologyLabel(labelName string) bool {
	return strings.HasPrefix(labelName, LabelTopologyKeyPrefix)
}
