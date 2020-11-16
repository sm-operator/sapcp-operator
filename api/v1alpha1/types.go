package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Label struct {
	Key   string   `json:"key"`
	Value []string `json:"value"`
}

type ConditionType string

const (
	// ConditionReady represents that a given resource is in ready state.
	ConditionReady ConditionType = "Ready"

	// ConditionFailed represents information about a final failure that should not be retried.
	ConditionFailed ConditionType = "Failed"
)

type Condition struct {
	// Type of the condition, currently ('Ready').
	Type ConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown').
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status change of this condition.
	LastTransitionTime v1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last transition, complementing reason.
	Message string `json:"message,omitempty"`
}

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)
