package reconciler

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetNSName(namespace string, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}

func GetNSNameFrom(object client.Object) types.NamespacedName {
	return GetNSName(object.GetNamespace(), object.GetName())
}
