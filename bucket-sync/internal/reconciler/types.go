package reconciler

import (
	controllerruntime "sigs.k8s.io/controller-runtime"
)

type Reconciler interface {
	Register(mgr controllerruntime.Manager) error
}

const (
	Finalizer = "bucket-sync.homelab-images.benfiola.com/finalizer"

	AnnotationLock       = "bucket-sync.homelab-images.benfiola.com/lock"
	KeyLockAccessedAt    = "accessed-at"
	KeyLockSyncNamespace = "sync-namespace"
	KeyLockSyncName      = "sync"
	KeyLockBucket        = "bucket"
)
