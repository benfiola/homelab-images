package reconciler

// +kubebuilder:rbac:groups=cilium.io,resources=ciliumclusterwidenetworkpolicies,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=router-policy-sync.homelab-images.benfiola.com,resources=routerpolicies,verbs=create;delete;get;list;patch;update;watch

import (
	"context"
	"time"

	"github.com/benfiola/homelab-images/router-policy-sync/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	ciliumapis "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AnnotationSyncWithRouter = "router-policy-sync.homelab-images.benfiola.com/sync-with-router"
)

type CiliumPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *CiliumPolicyReconciler) Register(manager controllerruntime.Manager) error {
	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&ciliumapis.CiliumClusterwideNetworkPolicy{}).
		Complete(r)
}

func (r *CiliumPolicyReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	ciliumPolicy := &ciliumapis.CiliumClusterwideNetworkPolicy{}
	err := r.Get(ctx, request.NamespacedName, ciliumPolicy)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch cilium policy", "error", err)
		return controllerruntime.Result{}, err
	}

	if ciliumPolicy.DeletionTimestamp != nil {
		if !controllerutil.ContainsFinalizer(ciliumPolicy, Finalizer) {
			return controllerruntime.Result{}, nil
		}

		controllerutil.RemoveFinalizer(ciliumPolicy, Finalizer)
		err := r.Client.Update(ctx, ciliumPolicy)
		if err != nil {
			logger.Error("failed to remove finalizer", "error", err)
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}

	annotations := ciliumPolicy.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	_, hasAnnotation := annotations[AnnotationSyncWithRouter]

	policy := &api.RouterPolicy{
		ObjectMeta: v1.ObjectMeta{Name: ciliumPolicy.GetName()},
	}

	if !hasAnnotation {
		if !controllerutil.ContainsFinalizer(ciliumPolicy, Finalizer) {
			return controllerruntime.Result{}, nil
		}

		err = r.Client.Delete(ctx, policy)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("failed to delete router policy", "router-policy", policy.Name, "error", err)
			return controllerruntime.Result{}, err
		}

		controllerutil.RemoveFinalizer(ciliumPolicy, Finalizer)
		err = r.Client.Update(ctx, ciliumPolicy)
		if err != nil {
			logger.Error("failed to remove finalizer", "error", err)
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(ciliumPolicy, Finalizer) {
		controllerutil.AddFinalizer(ciliumPolicy, Finalizer)
		err = r.Client.Update(ctx, ciliumPolicy)
		if err != nil {
			logger.Error("failed to add finalizer", "error", err)
			return controllerruntime.Result{}, err
		}
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Annotations = map[string]string{
			AnnotationRefresh: time.Now().UTC().Format(time.RFC3339),
		}
		return controllerutil.SetOwnerReference(ciliumPolicy, policy, r.Scheme)
	})
	if err != nil {
		logger.Error("failed to create or update router policy", "router-policy", policy.Name, "error", err)
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{}, nil
}
