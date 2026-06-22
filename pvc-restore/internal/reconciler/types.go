package reconciler

import (
	controllerruntime "sigs.k8s.io/controller-runtime"
)

type Reconciler interface {
	Register(mgr controllerruntime.Manager) error
}

const (
	Finalizer = "pvc-restore.homelab-images.benfiola.com/finalizer"

	AnnotationLock    = "pvc-restore.homelab-images.benfiola.com/lock"
	KeyLockAccessedAt = "accessed-at"
	KeyLockRestore    = "restore"
	KeyLockPVC        = "pvc"
)
