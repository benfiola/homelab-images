package reconciler

import controllerruntime "sigs.k8s.io/controller-runtime"

type Reconciler interface {
	Register(mgr controllerruntime.Manager) error
}


