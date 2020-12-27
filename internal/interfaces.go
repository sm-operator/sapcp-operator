package internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type SAPCPResource interface {
	SetConditions([]metav1.Condition)
	GetConditions() []metav1.Condition
	GetControllerName() string
	GetParameters() *runtime.RawExtension
}
