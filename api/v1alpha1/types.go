package v1alpha1

type Label struct {
	Key   string   `json:"key"`
	Value []string `json:"value"`
}

const (
	// ConditionReady represents that a given resource is in ready state.
	ConditionReady = "Ready"

	// ConditionFailed represents information about a final failure that should not be retried.
	ConditionFailed = "Failed"
)
