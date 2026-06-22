package reconciler

// +kubebuilder:rbac:groups=pvc-restore.homelab-images.benfiola.com,resources=pvcrestores,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=pvc-restore.homelab-images.benfiola.com,resources=pvcrestores/status,verbs=get;patch;update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;delete;get;update
// +kubebuilder:rbac:groups="",resources=pods,verbs=list;watch
// +kubebuilder:rbac:groups=volsync.backube,resources=replicationsources,verbs=get;list;watch
// +kubebuilder:rbac:groups=volsync.backube,resources=replicationdestinations,verbs=create;delete;get;list;watch
// +kubebuilder:rbac:groups="apps",resources="*",verbs=get
// +kubebuilder:rbac:groups="apps",resources="*/scale",verbs=get;update

import (
	"context"
	"fmt"
	"maps"
	"time"

	volsyncv1alpha1 "github.com/backube/volsync/api/v1alpha1"
	"github.com/benfiola/homelab-images/pvc-restore/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/ptr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/scale"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	IntervalWaitForLock    = 15 * time.Second
	IntervalWaitForRestore = 15 * time.Second
	PhaseInitialize        = "Initialize"
	PhaseScaleDown         = "ScaleDown"
	PhaseRestore           = "Restore"
	PhaseFinalize          = "Finalize"
	PhaseFinished          = "Finished"

	TimeoutRestore = 5 * time.Minute
)

type PVCRestoreReconciler struct {
	client.Client
	CacheStorageClass string
	DiscoveryClient   discovery.CachedDiscoveryInterface
	RESTMapper        *restmapper.DeferredDiscoveryRESTMapper
	Scaler            scale.ScalesGetter
	Scheme            *runtime.Scheme
}

type PhaseResult struct {
	NextPhase string
	Result    controllerruntime.Result
	Err       error
}

type PhaseHandler func(context.Context, *api.PVCRestore) PhaseResult

func (r *PVCRestoreReconciler) Register(manager controllerruntime.Manager) error {
	restConfig := manager.GetConfig()

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return err
	}

	cachedDiscoveryClient := memory.NewMemCacheClient(discoveryClient)

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)

	scaler, err := scale.NewForConfig(
		manager.GetConfig(),
		restMapper,
		dynamic.LegacyAPIPathResolverFunc,
		scale.NewDiscoveryScaleKindResolver(cachedDiscoveryClient),
	)
	if err != nil {
		return err
	}

	r.DiscoveryClient = cachedDiscoveryClient
	r.RESTMapper = restMapper
	r.Scaler = scaler

	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&api.PVCRestore{}).
		Complete(r)
}

func (r *PVCRestoreReconciler) failRestore(ctx context.Context, restore *api.PVCRestore, reason string) PhaseResult {
	logger := logging.FromContext(ctx)

	restore.Status.Error = &reason
	err := r.Status().Update(ctx, restore)
	if err != nil {
		logger.Error("failed to set restore error", "error", err)
		return PhaseResult{Err: err}
	}

	return PhaseResult{NextPhase: PhaseFinalize}
}

func (r *PVCRestoreReconciler) ensureLocked(ctx context.Context, restore *api.PVCRestore) (bool, error) {
	namespace, err := r.resolveObjectNamespace(restore)
	if err != nil {
		return false, err
	}

	lock := NewPVCLock(r.Client, namespace, restore.Spec.PVC)
	acquired, _, err := lock.TryAcquire(ctx, restore)
	if err != nil {
		return false, err
	}

	return acquired, nil
}

func (r *PVCRestoreReconciler) isTimedOut(restore *api.PVCRestore) bool {
	now := time.Now().UTC()
	timedOut := now.Sub(restore.CreationTimestamp.Time) > TimeoutRestore
	return timedOut
}

func (r *PVCRestoreReconciler) releaseLock(ctx context.Context, restore *api.PVCRestore) error {
	namespace, err := r.resolveObjectNamespace(restore)
	if err != nil {
		return err
	}

	lock := NewPVCLock(r.Client, namespace, restore.Spec.PVC)
	return lock.Release(ctx, restore)
}

func (r *PVCRestoreReconciler) normalizeNamespace(mapping *meta.RESTMapping, namespace string) string {
	if mapping.Scope.Name() != meta.RESTScopeNameNamespace {
		return ""
	} else if namespace == "" {
		return "default"
	}
	return namespace
}

func (r *PVCRestoreReconciler) resolveRefNamespace(ref api.ResourceRef) (string, error) {
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return "", err
	}

	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ref.Kind,
	}

	mapping, err := r.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	return r.normalizeNamespace(mapping, ref.Namespace), nil
}

func (r *PVCRestoreReconciler) resolveObjectNamespace(obj client.Object) (string, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()

	mapping, err := r.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	return r.normalizeNamespace(mapping, obj.GetNamespace()), nil
}

func (r *PVCRestoreReconciler) getResource(ctx context.Context, ref api.ResourceRef) (client.Object, error) {
	namespace, err := r.resolveRefNamespace(ref)
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ref.APIVersion)
	obj.SetKind(ref.Kind)

	err = r.Get(ctx, GetNSName(namespace, ref.Name), obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (r *PVCRestoreReconciler) objectToRef(o client.Object) api.ResourceRef {
	gvk := o.GetObjectKind().GroupVersionKind()
	return api.ResourceRef{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Namespace:  o.GetNamespace(),
		Name:       o.GetName(),
	}
}

func (r *PVCRestoreReconciler) refToRef(ownerRef v1.OwnerReference, namespace string) api.ResourceRef {
	return api.ResourceRef{
		APIVersion: ownerRef.APIVersion,
		Kind:       ownerRef.Kind,
		Namespace:  namespace,
		Name:       ownerRef.Name,
	}
}

func (r *PVCRestoreReconciler) findResourceRoot(ctx context.Context, resource client.Object) (map[string]client.Object, error) {
	ownerReferences := resource.GetOwnerReferences()
	if len(ownerReferences) == 0 {
		return map[string]client.Object{}, nil
	}

	allRoots := map[string]client.Object{}
	for _, ownerReference := range ownerReferences {
		resourceRef := r.refToRef(ownerReference, resource.GetNamespace())
		resource, err := r.getResource(ctx, resourceRef)
		if err != nil {
			return nil, err
		}
		resourceRoots, err := r.findResourceRoot(ctx, resource)
		if err != nil {
			return nil, err
		}
		if len(resourceRoots) == 0 {
			ref := r.objectToRef(resource)
			allRoots[ref.Key()] = resource
		} else {
			maps.Copy(allRoots, resourceRoots)
		}
	}

	return allRoots, nil
}

func (r *PVCRestoreReconciler) findPods(ctx context.Context, restore *api.PVCRestore) ([]*corev1.Pod, error) {
	namespace, err := r.resolveObjectNamespace(restore)
	if err != nil {
		return nil, err
	}

	podList := &corev1.PodList{}
	err = r.Client.List(ctx, podList, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	pods := []*corev1.Pod{}
	for _, pod := range podList.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == restore.Spec.PVC {
				pods = append(pods, &pod)
			}
		}
	}

	return pods, nil
}

func (r *PVCRestoreReconciler) getScaleInterface(o client.Object) (scale.ScaleInterface, schema.GroupResource, error) {
	namespace, err := r.resolveObjectNamespace(o)
	if err != nil {
		return nil, schema.GroupResource{}, err
	}

	gvk := o.GetObjectKind().GroupVersionKind()

	mapping, err := r.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, schema.GroupResource{}, err
	}

	resource := mapping.Resource.GroupResource()

	return r.Scaler.Scales(namespace), resource, nil
}

func (r *PVCRestoreReconciler) getScale(ctx context.Context, o client.Object, scale *autoscalingv1.Scale) error {
	scaler, resource, err := r.getScaleInterface(o)
	if err != nil {
		return err
	}

	fetched, err := scaler.Get(ctx, resource, o.GetName(), v1.GetOptions{})
	if err != nil {
		return err
	}

	*scale = *fetched

	return nil
}

func (r *PVCRestoreReconciler) setScale(ctx context.Context, o client.Object, scale *autoscalingv1.Scale) error {
	scaler, resource, err := r.getScaleInterface(o)
	if err != nil {
		return err
	}

	updated, err := scaler.Update(ctx, resource, scale, v1.UpdateOptions{})
	if err != nil {
		return err
	}

	*scale = *updated

	return nil
}

func (r *PVCRestoreReconciler) findReplicationSource(ctx context.Context, restore *api.PVCRestore) (*volsyncv1alpha1.ReplicationSource, error) {
	namespace, err := r.resolveObjectNamespace(restore)
	if err != nil {
		return nil, err
	}

	repSrcList := &volsyncv1alpha1.ReplicationSourceList{}
	err = r.List(ctx, repSrcList, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	for _, repSrc := range repSrcList.Items {
		if repSrc.Spec.SourcePVC == restore.Spec.PVC {
			return &repSrc, nil
		}
	}

	return nil, nil
}

func (r *PVCRestoreReconciler) buildReplicationDestination(restore *api.PVCRestore, repSrc *volsyncv1alpha1.ReplicationSource) (*volsyncv1alpha1.ReplicationDestination, error) {
	namespace, err := r.resolveObjectNamespace(restore)
	if err != nil {
		return nil, err
	}

	var cacheStorageClass *string
	if r.CacheStorageClass != "" {
		cacheStorageClass = ptr.Get(r.CacheStorageClass)
	}
	name := fmt.Sprintf("pvc-restore-%s", restore.Name)
	gvk := restore.GroupVersionKind()
	repDst := &volsyncv1alpha1.ReplicationDestination{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Name:       restore.GetName(),
					UID:        restore.GetUID(),
				},
			},
		},
		Spec: volsyncv1alpha1.ReplicationDestinationSpec{
			Restic: &volsyncv1alpha1.ReplicationDestinationResticSpec{
				CacheStorageClassName: cacheStorageClass,
				EnableFileDeletion:    true,
				MoverConfig:           repSrc.Spec.Restic.MoverConfig,
				Previous:              restore.Spec.Previous,
				Repository:            repSrc.Spec.Restic.Repository,
				ReplicationDestinationVolumeOptions: volsyncv1alpha1.ReplicationDestinationVolumeOptions{
					CopyMethod:     volsyncv1alpha1.CopyMethodDirect,
					DestinationPVC: ptr.Get(restore.Spec.PVC),
				},
				RestoreAsOf: restore.Spec.RestoreAsOf,
			},
		},
	}

	return repDst, nil
}

func (r *PVCRestoreReconciler) initialize(ctx context.Context, restore *api.PVCRestore) PhaseResult {
	logger := logging.FromContext(ctx)

	acquired, err := r.ensureLocked(ctx, restore)
	if err != nil {
		logger.Error("failed to ensure lock", "error", err)
		return PhaseResult{Err: err}
	}

	if !acquired {
		logger.Info("waiting for lock")
		return PhaseResult{Result: controllerruntime.Result{RequeueAfter: IntervalWaitForLock}}
	}

	if !controllerutil.ContainsFinalizer(restore, Finalizer) {
		controllerutil.AddFinalizer(restore, Finalizer)
		err = r.Client.Update(ctx, restore)
		if err != nil {
			logger.Error("failed to add finalizer to restore", "error", err)
			return PhaseResult{Err: err}
		}
	}

	return PhaseResult{NextPhase: PhaseScaleDown}
}

func (r *PVCRestoreReconciler) scaleDown(ctx context.Context, restore *api.PVCRestore) PhaseResult {
	logger := logging.FromContext(ctx)

	acquired, err := r.ensureLocked(ctx, restore)
	if err != nil {
		logger.Error("failed to ensure lock", "error", err)
		return PhaseResult{Err: err}
	}
	if !acquired {
		return r.failRestore(ctx, restore, "lock is not acquired")
	}

	if r.isTimedOut(restore) {
		return r.failRestore(ctx, restore, "restore timed out")
	}

	if restore.Status.PVCOwners == nil {
		pods, err := r.findPods(ctx, restore)
		if err != nil {
			logger.Error("failed to find pvc pods", "error", err)
			return PhaseResult{Err: err}
		}

		ownersMap := map[string]client.Object{}
		for _, pod := range pods {
			podOwners, err := r.findResourceRoot(ctx, pod)
			if err != nil {
				logger.Error("failed to find pvc pod owners", "pod", GetNSNameFrom(pod), "error", err)
				return PhaseResult{Err: err}
			}
			maps.Copy(ownersMap, podOwners)
		}

		owners := []api.PVCOwner{}
		for _, ownerObj := range ownersMap {
			scale := &autoscalingv1.Scale{}
			err = r.getScale(ctx, ownerObj, scale)
			if err != nil {
				logger.Error("failed to get scale subresource", "owner", GetNSNameFrom(ownerObj), "error", err)
				return PhaseResult{Err: err}
			}
			owner := api.PVCOwner{
				ResourceRef: r.objectToRef(ownerObj),
				Replicas:    scale.Spec.Replicas,
			}
			owners = append(owners, owner)
		}

		restore.Status.PVCOwners = owners
		err = r.Status().Update(ctx, restore)
		if err != nil {
			logger.Error("failed to set pvc owners", "error", err)
			return PhaseResult{Err: err}
		}
	}

	for _, restoreOwner := range restore.Status.PVCOwners {
		ownerRef := restoreOwner.ResourceRef
		owner, err := r.getResource(ctx, ownerRef)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				logger.Warn("owner not found", "owner", GetNSName(ownerRef.Namespace, ownerRef.Name))
				continue
			}
			logger.Error("failed to fetch owner", "owner", GetNSName(ownerRef.Namespace, ownerRef.Name), "error", err)
			return PhaseResult{Err: err}
		}
		scale := &autoscalingv1.Scale{}
		err = r.getScale(ctx, owner, scale)
		if err != nil {
			if errors.IsMethodNotSupported(err) {
				return r.failRestore(ctx, restore, fmt.Sprintf("resource does not support scale: %s", ownerRef.Key()))
			}
			logger.Error("failed to get scale subresource", "owner", GetNSNameFrom(owner), "error", err)
			return PhaseResult{Err: err}
		}
		scale.Spec.Replicas = 0
		err = r.setScale(ctx, owner, scale)
		if err != nil {
			logger.Error("failed to scale down owner", "owner", GetNSNameFrom(owner), "error", err)
			return PhaseResult{Err: err}
		}
	}

	return PhaseResult{NextPhase: PhaseRestore}
}

func (r *PVCRestoreReconciler) restore(ctx context.Context, restore *api.PVCRestore) PhaseResult {
	logger := logging.FromContext(ctx)

	acquired, err := r.ensureLocked(ctx, restore)
	if err != nil {
		logger.Error("failed to ensure lock", "error", err)
		return PhaseResult{Err: err}
	}
	if !acquired {
		return r.failRestore(ctx, restore, "lock is not acquired")
	}

	if r.isTimedOut(restore) {
		return r.failRestore(ctx, restore, "restore timed out")
	}

	namespace, err := r.resolveObjectNamespace(restore)
	if err != nil {
		logger.Error("failed to get namespace for restore", "error", err)
		return PhaseResult{Err: err}
	}

	if restore.Status.ReplicationSource == nil {
		repSrc, err := r.findReplicationSource(ctx, restore)
		if err != nil {
			logger.Error("failed to find replication source", "error", err)
			return PhaseResult{Err: err}
		}
		if repSrc == nil {
			return r.failRestore(ctx, restore, "no replication source found")
		}

		restore.Status.ReplicationSource = &repSrc.Name
		err = r.Status().Update(ctx, restore)
		if err != nil {
			logger.Error("failed to set replication source", "error", err)
			return PhaseResult{Err: err}
		}
	}

	if restore.Status.ReplicationDestination == nil {
		repSrcNSN := GetNSName(namespace, *restore.Status.ReplicationSource)
		repSrc := &volsyncv1alpha1.ReplicationSource{}
		err = r.Client.Get(ctx, repSrcNSN, repSrc)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				return r.failRestore(ctx, restore, fmt.Sprintf("replication source %s no longer exists", repSrcNSN.String()))
			}
			logger.Error("failed to get replication source", "replication-source", repSrcNSN, "error", err)
			return PhaseResult{Err: err}
		}

		repDst, err := r.buildReplicationDestination(restore, repSrc)
		if err != nil {
			logger.Error("failed to build replication destination", "error", err)
			return PhaseResult{Err: err}
		}
		err = r.Client.Create(ctx, repDst)
		if client.IgnoreAlreadyExists(err) != nil {
			logger.Error("failed to create replication destination", "error", err)
			return PhaseResult{Err: err}
		}

		restore.Status.ReplicationDestination = ptr.Get(repDst.GetName())
		err = r.Status().Update(ctx, restore)
		if err != nil {
			logger.Error("failed to set replication destination", "error", err)
			return PhaseResult{Err: err}
		}
	}

	repDstNSN := GetNSName(namespace, *restore.Status.ReplicationDestination)
	repDst := &volsyncv1alpha1.ReplicationDestination{}
	err = r.Client.Get(ctx, repDstNSN, repDst)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return r.failRestore(ctx, restore, fmt.Sprintf("replication destination %s no longer exists", repDstNSN.String()))
		}
		logger.Error("failed to get replication destination", "replication-destination", repDstNSN, "error", err)
		return PhaseResult{Err: err}
	}

	if repDst.Status.LatestMoverStatus == nil || repDst.Status.LatestMoverStatus.Result == "" {
		now := time.Now().UTC()
		if now.Sub(repDst.CreationTimestamp.Time) > TimeoutRestore {
			return r.failRestore(ctx, restore, "restore timed out")
		}
		logger.Info("restore not finished")
		return PhaseResult{Result: controllerruntime.Result{RequeueAfter: IntervalWaitForRestore}}
	}

	if repDst.Status.LatestMoverStatus.Result == volsyncv1alpha1.MoverResultFailed {
		restore.Status.Error = ptr.Get(repDst.Status.LatestMoverStatus.Logs)
		err = r.Status().Update(ctx, restore)
		if err != nil {
			logger.Error("failed to set restore error", "error", err)
			return PhaseResult{Err: err}
		}
	}

	return PhaseResult{NextPhase: PhaseFinalize}
}

func (r *PVCRestoreReconciler) finalize(ctx context.Context, restore *api.PVCRestore) PhaseResult {
	logger := logging.FromContext(ctx)

	if restore.Status.PVCOwners != nil {
		for _, restoreOwner := range restore.Status.PVCOwners {
			ownerRef := restoreOwner.ResourceRef
			owner, err := r.getResource(ctx, ownerRef)
			if err != nil {
				if client.IgnoreNotFound(err) == nil {
					logger.Warn("owner not found", "owner", GetNSName(ownerRef.Namespace, ownerRef.Name))
					continue
				}
				logger.Error("failed to fetch owner", "owner", GetNSName(ownerRef.Namespace, ownerRef.Name), "error", err)
				return PhaseResult{Err: err}
			}
			scale := &autoscalingv1.Scale{}
			err = r.getScale(ctx, owner, scale)
			if err != nil {
				logger.Error("failed to get scale subresource", "owner", GetNSNameFrom(owner), "error", err)
				return PhaseResult{Err: err}
			}
			scale.Spec.Replicas = restoreOwner.Replicas
			err = r.setScale(ctx, owner, scale)
			if err != nil {
				logger.Error("failed to scale up owner", "owner", GetNSNameFrom(owner), "error", err)
				return PhaseResult{Err: err}
			}
		}
	}

	if restore.Status.ReplicationDestination != nil {
		namespace, err := r.resolveObjectNamespace(restore)
		if err != nil {
			logger.Error("failed to resolve namespace", "error", err)
			return PhaseResult{Err: err}
		}

		repDst := &volsyncv1alpha1.ReplicationDestination{ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      *restore.Status.ReplicationDestination,
		}}
		err = r.Client.Delete(ctx, repDst)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("failed to delete replication destination", "replication-destination", GetNSNameFrom(repDst), "error", err)
			return PhaseResult{Err: err}
		}
	}

	err := r.releaseLock(ctx, restore)
	if err != nil {
		logger.Error("failed to release lock", "error", err)
		return PhaseResult{Err: err}
	}

	if controllerutil.ContainsFinalizer(restore, Finalizer) {
		controllerutil.RemoveFinalizer(restore, Finalizer)
		err = r.Client.Update(ctx, restore)
		if err != nil {
			logger.Error("failed to remove finalizer from restore", "error", err)
			return PhaseResult{Err: err}
		}
	}

	return PhaseResult{NextPhase: PhaseFinished}
}

func (r *PVCRestoreReconciler) handlePhaseResult(ctx context.Context, restore *api.PVCRestore, result PhaseResult) (controllerruntime.Result, error) {
	logger := logging.FromContext(ctx)

	var err error

	if result.NextPhase != "" && restore.Status.Phase != result.NextPhase {
		logger.Info("transitioning phase", "from-phase", restore.Status.Phase, "to-phase", result.NextPhase)
		restore.Status.Phase = result.NextPhase
		err = r.Status().Update(ctx, restore)
		if err != nil {
			logger.Error("failed to transition phase", "from-phase", restore.Status.Phase, "to-phase", result.NextPhase)
			return controllerruntime.Result{}, err
		}
	}

	if result.Err == nil {
		restore.Status.LastReconciledTime = ptr.Get(v1.Now())
		restore.Status.ObservedGeneration = restore.Generation
		err = r.Status().Update(ctx, restore)
		if err != nil {
			logger.Warn("failed to update reconcile time and generation", "error", err)
		}
	}

	return result.Result, result.Err
}

func (r *PVCRestoreReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	restore := &api.PVCRestore{}

	err := r.Client.Get(ctx, request.NamespacedName, restore)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch restore", "error", err)
		return controllerruntime.Result{}, err
	}

	if restore.Status.Phase == "" {
		return r.handlePhaseResult(ctx, restore, PhaseResult{NextPhase: PhaseInitialize})
	}

	terminals := map[string]bool{
		PhaseFinished: true,
	}
	_, isTerminalPhase := terminals[restore.Status.Phase]
	if isTerminalPhase {
		return r.handlePhaseResult(ctx, restore, PhaseResult{})
	}

	phases := map[string]PhaseHandler{
		PhaseInitialize: r.initialize,
		PhaseScaleDown:  r.scaleDown,
		PhaseRestore:    r.restore,
		PhaseFinalize:   r.finalize,
	}

	handler, hasPhaseHandler := phases[restore.Status.Phase]
	if !hasPhaseHandler {
		logger.Error("invalid phase", "phase", restore.Status.Phase)
		return controllerruntime.Result{}, nil
	}

	result := handler(ctx, restore)
	return r.handlePhaseResult(ctx, restore, result)
}
