package reconciler

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=delete;get;list;watch
// +kubebuilder:rbac:groups=pvc-restore.homelab-images.benfiola.com,resources=pvcrestores,verbs=get;

import (
	"context"
	"time"

	"github.com/benfiola/homelab-images/pvc-restore/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TimeoutLock = 10 * time.Minute
)

type LockCleanupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *LockCleanupReconciler) Register(manager controllerruntime.Manager) error {
	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *LockCleanupReconciler) deleteLock(ctx context.Context, cm *corev1.ConfigMap, reason string) (controllerruntime.Result, error) {
	logger := logging.FromContext(ctx)
	logger.Info("deleting stale lock", "reason", reason)

	err := r.Client.Delete(ctx, cm)
	if err != nil {
		logger.Error("failed to delete stale lock", "reason", reason, "error", err)
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{}, nil
}

func (r *LockCleanupReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	cm := &corev1.ConfigMap{}

	err := r.Client.Get(ctx, request.NamespacedName, cm)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch config map", "error", err)
		return controllerruntime.Result{}, err
	}

	if cm.DeletionTimestamp != nil {
		return controllerruntime.Result{}, nil
	}

	if cm.Annotations == nil || cm.Annotations[AnnotationLock] != "true" {
		return controllerruntime.Result{}, nil
	}

	accessedAtStr := cm.Data[KeyLockAccessedAt]
	restoreName := cm.Data[KeyLockRestore]
	pvc := cm.Data[KeyLockPVC]

	if accessedAtStr == "" || restoreName == "" || pvc == "" {
		return r.deleteLock(ctx, cm, "malformed data")
	}

	accessedAt, err := time.Parse(time.RFC3339, accessedAtStr)
	if err != nil {
		return r.deleteLock(ctx, cm, "unparseable time")
	}

	age := time.Since(accessedAt)
	if age > TimeoutLock {
		return r.deleteLock(ctx, cm, "lock timeout")
	}

	restore := &api.PVCRestore{}
	err = r.Client.Get(ctx, GetNSName(cm.Namespace, restoreName), restore)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return r.deleteLock(ctx, cm, "missing pvc restore")
		}

		logger.Error("failed to get pvc restore resource", "restore", GetNSNameFrom(restore), "error", err)
		return controllerruntime.Result{}, err
	}

	requeueAfter := TimeoutLock - age
	return controllerruntime.Result{RequeueAfter: requeueAfter}, nil
}
