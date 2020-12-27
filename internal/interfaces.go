package internal

import (
	"github.com/sm-operator/sapcp-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SAPCPResource interface {
	client.Object
	SetConditions([]metav1.Condition)
	GetConditions() []metav1.Condition
	GetControllerName() v1alpha1.ControllerName
	GetParameters() *runtime.RawExtension
	GetStatus() interface{}
	SetStatus(status interface{})
}
