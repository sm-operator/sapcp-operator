package internal

import "github.com/sm-operator/sapcp-operator/api/v1alpha1"

type SAPCPResource interface {
	SetConditions([]*v1alpha1.Condition)
	GetConditions() []*v1alpha1.Condition
	GetControllerName() string
}
