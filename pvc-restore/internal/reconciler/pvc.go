package reconciler

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=pvc-restore.homelab-images.benfiola.com,resources=pvcrestores,verbs=create

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/benfiola/homelab-images/pvc-restore/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/ptr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationRestore = "pvc-restore.homelab-images.benfiola.com/restore"
)

type PersistentVolumeClaimReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

func (r *PersistentVolumeClaimReconciler) Register(manager controllerruntime.Manager) error {
	r.Recorder = manager.GetEventRecorderFor("pvc-restore")

	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}

func (r *PersistentVolumeClaimReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	pvc := &corev1.PersistentVolumeClaim{}

	err := r.Client.Get(ctx, request.NamespacedName, pvc)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch pvc", "error", err)
		return controllerruntime.Result{}, err
	}

	if pvc.GetAnnotations() == nil {
		pvc.SetAnnotations(map[string]string{})
	}
	restoreValue, ok := pvc.GetAnnotations()[AnnotationRestore]
	if !ok {
		return controllerruntime.Result{}, nil
	}

	var previous *int32
	var restoreAsOf *string

	if restoreValue != "" {
		if restoreValueInt, err := strconv.Atoi(restoreValue); err == nil {
			previous = ptr.Get(int32(restoreValueInt))
		} else if _, err := time.Parse(time.RFC3339, restoreValue); err == nil {
			restoreAsOf = ptr.Get(restoreValue)
		}
		if previous == nil && restoreAsOf == nil {
			logger.Error("invalid restore annotation value", "value", restoreValue)
			return controllerruntime.Result{}, nil
		}
	}

	annotations := pvc.GetAnnotations()
	delete(annotations, AnnotationRestore)
	pvc.SetAnnotations(annotations)
	err = r.Client.Update(ctx, pvc)
	if err != nil {
		logger.Error("failed to clear pvc restore annotation", "error", err)
		return controllerruntime.Result{}, err
	}

	now := time.Now()
	name := fmt.Sprintf("%s-%d", pvc.Name, now.Unix())
	restore := &api.PVCRestore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvc.Namespace,
			Name:      name,
		},
		Spec: api.PVCRestoreSpec{
			PVC:         pvc.Name,
			Previous:    previous,
			RestoreAsOf: restoreAsOf,
		},
	}
	err = r.Client.Create(ctx, restore)
	if err != nil {
		logger.Error("failed to create pvc restore resource", "pvc-restore", GetNSNameFrom(restore), "error", err)
		r.Recorder.Event(pvc, corev1.EventTypeWarning, "PVCRestoreCreationFailed", fmt.Sprintf("failed to create pvc restore resource from pvc: %s", err.Error()))
	}

	return controllerruntime.Result{}, nil
}
