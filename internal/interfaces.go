package internal

import "github.wdf.sap.corp/i042428/sapcp-operator/api/v1alpha1"

type SapCPController interface {
	SetConditions([]*v1alpha1.Condition)
	GetConditions() []*v1alpha1.Condition
	GetControllerName() string
}
