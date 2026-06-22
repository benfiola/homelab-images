package reconciler

import controllerruntime "sigs.k8s.io/controller-runtime"

const (
	AnnotationRefresh = "router-policy-sync.homelab-images.benfiola.com/refresh"
	Finalizer         = "router-policy-sync.homelab-images.benfiola.com/finalizer"
)

type Reconciler interface {
	Register(mgr controllerruntime.Manager) error
}
