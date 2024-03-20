package v1alpha1

// ConditionType is a type of condition
type ConditionType string

const (
	// Ready means the resource is ready
	Ready ConditionType = "Ready"
)

type ConditionStatus string

const (
	True    ConditionStatus = "True"
	False   ConditionStatus = "False"
	Unknown ConditionStatus = "Unknown"
)

// Condition describes the state of a resource at a certain point.
type Condition struct {
	// Type of condition
	Type string `json:"type"`

	// Status of the condition, one of True, False, Unknown
	Status string `json:"status"`

	//Message is a human-readable message indicating details about the condition
	Message string `json:"message,omitempty"`

	// LastTransitionTime is the last time the condition transitioned from one status to another
	LastTransitionTime string `json:"lastTransitionTime"`
}
