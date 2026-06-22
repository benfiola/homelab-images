package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/pvc-restore/internal/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PVCLock struct {
	client.Client
	Namespace string
	PVC       string
}

func NewPVCLock(c client.Client, namespace string, pvc string) *PVCLock {
	lock := &PVCLock{
		Client:    c,
		Namespace: namespace,
		PVC:       pvc,
	}
	return lock
}

func (l *PVCLock) buildConfigMap(restore *api.PVCRestore) *corev1.ConfigMap {
	name := fmt.Sprintf("pvc-restore-lock-%s", l.PVC)
	accessedAt := time.Now().UTC().Format(time.RFC3339)

	gvk := restore.GetObjectKind().GroupVersionKind()

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Namespace: l.Namespace,
			Name:      name,
			Annotations: map[string]string{
				AnnotationLock: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					UID:        restore.GetUID(),
					Name:       restore.GetName(),
				},
			},
		},
		Data: map[string]string{
			KeyLockAccessedAt: accessedAt,
			KeyLockRestore:    restore.Name,
			KeyLockPVC:        l.PVC,
		},
	}

	return cm
}

func (l *PVCLock) Release(ctx context.Context, restore *api.PVCRestore) error {
	cm := &corev1.ConfigMap{}
	err := l.Get(ctx, GetNSNameFrom(l.buildConfigMap(restore)), cm)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}

	if cm.Data[KeyLockRestore] != restore.Name {
		return nil
	}

	err = l.Delete(ctx, cm)
	if err != nil {
		return err
	}
	return nil
}

func (l *PVCLock) TryAcquire(ctx context.Context, restore *api.PVCRestore) (bool, string, error) {
	cm := l.buildConfigMap(restore)

	logger := logging.FromContext(ctx).With("resource", GetNSNameFrom(cm))

	err := l.Create(ctx, cm)
	if err == nil {
		return true, restore.Name, nil
	}

	if client.IgnoreAlreadyExists(err) != nil {
		return false, "", err
	}

	existing := &corev1.ConfigMap{}
	err = l.Get(ctx, GetNSNameFrom(cm), existing)
	if err != nil {
		return false, "", err
	}

	owner := existing.Data[KeyLockRestore]
	acquired := owner == restore.Name

	if acquired {
		existing.Data[KeyLockAccessedAt] = time.Now().UTC().Format(time.RFC3339)
		err := l.Update(ctx, existing)
		if err != nil {
			logger.Warn("failed to update lock 'accessed-at' timestamp", "error", err)
		}
	}

	return acquired, owner, nil
}
