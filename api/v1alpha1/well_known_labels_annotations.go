package v1alpha1

import (
	"strings"
)

const (
	LabelActiveEndpoint    = "raven.openyurt.io/is-active-endpoint"
	LabelTopologyKeyPrefix = "topology.raven.openyurt.io/"
)

// TopologyLabelName return the full label name of the given topology name.
func TopologyLabelName(topologyName string) string {
	return LabelTopologyKeyPrefix + topologyName
}

// IsTopologyLabel returns if the given label name is topology label.
func IsTopologyLabel(labelName string) bool {
	return strings.HasPrefix(labelName, LabelTopologyKeyPrefix)
}
