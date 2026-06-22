package reconciler

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=delete;get;list;watch
// +kubebuilder:rbac:groups=bucket-sync.homelab-images.benfiola.com,resources=bucketsyncs,verbs=get;

import (
	"context"
	"time"

	"github.com/benfiola/homelab-images/bucket-sync/internal/api"
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
	syncNamespace := cm.Data[KeyLockSyncNamespace]
	syncName := cm.Data[KeyLockSyncName]
	bucket := cm.Data[KeyLockBucket]

	if accessedAtStr == "" || syncNamespace == "" || syncName == "" || bucket == "" {
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

	sync := &api.BucketSync{}
	err = r.Client.Get(ctx, GetNSName(syncNamespace, syncName), sync)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return r.deleteLock(ctx, cm, "missing bucket sync")
		}

		logger.Error("failed to get bucket sync resource", "sync", GetNSNameFrom(sync), "error", err)
		return controllerruntime.Result{}, err
	}

	requeueAfter := TimeoutLock - age
	return controllerruntime.Result{RequeueAfter: requeueAfter}, nil
}
