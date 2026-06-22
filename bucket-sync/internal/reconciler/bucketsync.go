package reconciler

// +kubebuilder:rbac:groups=bucket-sync.homelab-images.benfiola.com,resources=bucketsyncs,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=bucket-sync.homelab-images.benfiola.com,resources=bucketsyncs/status,verbs=get;patch;update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;delete;get;update
// +kubebuilder:rbac:groups="batch",resources=jobs,verbs=create;delete;get;list;patch;update;watch

import (
	"context"
	"fmt"
	"time"

	"github.com/benfiola/homelab-images/bucket-sync/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/ptr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	IntervalWaitForLock    = 15 * time.Second
	IntervalWaitForRestore = 15 * time.Second

	TimeoutSync = 5 * time.Minute
)

type BucketSyncReconciler struct {
	client.Client
	Namespace string
	Scheme    *runtime.Scheme
}

type PhaseResult struct {
	NextPhase api.BucketSyncPhase
	Result    controllerruntime.Result
	Err       error
}

type PhaseHandler func(context.Context, *api.BucketSync) PhaseResult

func (r *BucketSyncReconciler) Register(manager controllerruntime.Manager) error {
	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&api.BucketSync{}).
		Complete(r)
}

func (r *BucketSyncReconciler) failRestore(ctx context.Context, sync *api.BucketSync, reason string) PhaseResult {
	logger := logging.FromContext(ctx)

	sync.Status.Error = &reason
	err := r.Status().Update(ctx, sync)
	if err != nil {
		logger.Error("failed to set sync error", "error", err)
		return PhaseResult{Err: err}
	}

	return PhaseResult{NextPhase: api.BucketSyncPhaseFinalize}
}

func (r *BucketSyncReconciler) ensureLocked(ctx context.Context, sync *api.BucketSync) (bool, error) {
	acquired := false

	lock := NewBucketLock(r.Client, r.Namespace, sync.Spec.Source)
	sourceAcquired, _, err := lock.TryAcquire(ctx, sync)
	if err != nil {
		return acquired, err
	}
	defer func() {
		if !acquired {
			lock.Release(ctx, sync)
		}
	}()

	lock = NewBucketLock(r.Client, r.Namespace, sync.Spec.Destination)
	destinationAcquired, _, err := lock.TryAcquire(ctx, sync)
	if err != nil {
		return acquired, err
	}
	defer func() {
		if !acquired {
			lock.Release(ctx, sync)
		}
	}()

	acquired = sourceAcquired && destinationAcquired

	return acquired, nil
}

func (r *BucketSyncReconciler) isTimedOut(sync *api.BucketSync) bool {
	now := time.Now().UTC()
	timedOut := now.Sub(sync.CreationTimestamp.Time) > TimeoutSync
	return timedOut
}

func (r *BucketSyncReconciler) releaseLock(ctx context.Context, sync *api.BucketSync) error {
	var err error

	lock := NewBucketLock(r.Client, r.Namespace, sync.Spec.Source)
	if err1 := lock.Release(ctx, sync); err1 != nil {
		err = err1
	}

	lock = NewBucketLock(r.Client, r.Namespace, sync.Spec.Destination)
	if err2 := lock.Release(ctx, sync); err2 != nil {
		err = err2
	}

	return err
}

func (r *BucketSyncReconciler) prefixEnv(vars []corev1.EnvVar, prefix string) []corev1.EnvVar {
	prefixed := make([]corev1.EnvVar, len(vars))
	for i, env := range vars {
		prefixed[i] = env
		prefixed[i].Name = prefix + env.Name
	}
	return prefixed
}

func (r *BucketSyncReconciler) buildJob(sync *api.BucketSync) *batchv1.Job {
	sourceEnv := r.prefixEnv(sync.Spec.SourceEnv, "RCLONE_CONFIG_SOURCE_")
	destEnv := r.prefixEnv(sync.Spec.DestinationEnv, "RCLONE_CONFIG_DESTINATION_")
	env := append(sourceEnv, destEnv...)

	source := fmt.Sprintf("source:%s", sync.Spec.Source)
	destination := fmt.Sprintf("destination:%s", sync.Spec.Destination)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sync.Name,
			Namespace: sync.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.Get(int32(0)),
			Completions:  ptr.Get(int32(1)),
			Parallelism:  ptr.Get(int32(1)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: sync.Spec.JobLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sync",
							Image: "rclone/rclone:1.73.1",
							Args:  []string{"sync", source, destination},
							Env:   env,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.Get(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								ReadOnlyRootFilesystem: ptr.Get(true),
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:    ptr.Get(int64(65534)),
						RunAsGroup:   ptr.Get(int64(65534)),
						RunAsNonRoot: ptr.Get(true),
						FSGroup:      ptr.Get(int64(65534)),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			},
			TTLSecondsAfterFinished: ptr.Get(int32(3600)),
		},
	}
}

func (r *BucketSyncReconciler) getJobStatus(job *batchv1.Job) (bool, bool) {
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			return true, true
		}
		if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
			return true, false
		}
	}
	return false, false
}

func (r *BucketSyncReconciler) initialize(ctx context.Context, sync *api.BucketSync) PhaseResult {
	logger := logging.FromContext(ctx)

	acquired, err := r.ensureLocked(ctx, sync)
	if err != nil {
		logger.Error("failed to ensure lock", "error", err)
		return PhaseResult{Err: err}
	}

	if !acquired {
		logger.Info("waiting for lock")
		return PhaseResult{Result: controllerruntime.Result{RequeueAfter: IntervalWaitForLock}}
	}

	if !controllerutil.ContainsFinalizer(sync, Finalizer) {
		controllerutil.AddFinalizer(sync, Finalizer)
		err = r.Client.Update(ctx, sync)
		if err != nil {
			logger.Error("failed to add finalizer to sync", "error", err)
			return PhaseResult{Err: err}
		}
	}

	sync.Status.StartTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, sync); err != nil {
		logger.Error("failed to set start time on sync", "error", err)
		return PhaseResult{Err: err}
	}

	return PhaseResult{NextPhase: api.BucketSyncPhaseSync}
}

func (r *BucketSyncReconciler) sync(ctx context.Context, sync *api.BucketSync) PhaseResult {
	logger := logging.FromContext(ctx)

	acquired, err := r.ensureLocked(ctx, sync)
	if err != nil {
		logger.Error("failed to ensure lock", "error", err)
		return PhaseResult{Err: err}
	}
	if !acquired {
		return r.failRestore(ctx, sync, "lock is not acquired")
	}

	if r.isTimedOut(sync) {
		return r.failRestore(ctx, sync, "sync timed out")
	}

	if sync.Status.Job == nil {
		job := r.buildJob(sync)

		err = r.Client.Create(ctx, job)
		if client.IgnoreAlreadyExists(err) != nil {
			logger.Error("failed to create job", "error", err)
			return PhaseResult{Err: err}
		}

		sync.Status.Job = ptr.Get(job.GetName())
		err = r.Status().Update(ctx, sync)
		if err != nil {
			logger.Error("failed to set job", "error", err)
			return PhaseResult{Err: err}
		}
	}

	job := &batchv1.Job{}
	if err := r.Client.Get(ctx, GetNSName(sync.Namespace, *sync.Status.Job), job); err != nil {
		logger.Error("failed to get job", "error", err)
		return PhaseResult{Err: err}
	}
	finished, successful := r.getJobStatus(job)
	if !finished {
		now := time.Now().UTC()
		if now.Sub(job.CreationTimestamp.Time) > TimeoutSync {
			return r.failRestore(ctx, sync, "sync timed out")
		}
		return PhaseResult{Result: controllerruntime.Result{RequeueAfter: IntervalWaitForRestore}}
	}
	if !successful {
		sync.Status.Error = ptr.Get("sync job failed")
		err = r.Status().Update(ctx, sync)
		if err != nil {
			logger.Error("failed to set sync error", "error", err)
			return PhaseResult{Err: err}
		}
	}

	return PhaseResult{NextPhase: api.BucketSyncPhaseFinalize}
}

func (r *BucketSyncReconciler) finalize(ctx context.Context, sync *api.BucketSync) PhaseResult {
	logger := logging.FromContext(ctx)

	if sync.Status.Job != nil {
		job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
			Namespace: sync.Namespace,
			Name:      *sync.Status.Job,
		}}
		err := r.Client.Delete(ctx, job, client.PropagationPolicy("Background"))
		if client.IgnoreNotFound(err) != nil {
			logger.Error("failed to delete job", "job", GetNSNameFrom(job), "error", err)
			return PhaseResult{Err: err}
		}
	}

	err := r.releaseLock(ctx, sync)
	if err != nil {
		logger.Error("failed to release lock", "error", err)
		return PhaseResult{Err: err}
	}

	sync.Status.FinishTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, sync); err != nil {
		logger.Error("failed to set finish time on sync resource", "error", err)
		return PhaseResult{Err: err}
	}

	if controllerutil.ContainsFinalizer(sync, Finalizer) {
		controllerutil.RemoveFinalizer(sync, Finalizer)
		err = r.Client.Update(ctx, sync)
		if err != nil {
			logger.Error("failed to remove finalizer from sync", "error", err)
			return PhaseResult{Err: err}
		}
	}

	return PhaseResult{NextPhase: api.BucketSyncPhaseFinished}
}

func (r *BucketSyncReconciler) handlePhaseResult(ctx context.Context, sync *api.BucketSync, result PhaseResult) (controllerruntime.Result, error) {
	logger := logging.FromContext(ctx)

	var err error

	if result.NextPhase != "" && sync.Status.Phase != result.NextPhase {
		logger.Info("transitioning phase", "from-phase", sync.Status.Phase, "to-phase", result.NextPhase)
		sync.Status.Phase = result.NextPhase
		err = r.Status().Update(ctx, sync)
		if err != nil {
			logger.Error("failed to transition phase", "from-phase", sync.Status.Phase, "to-phase", result.NextPhase)
			return controllerruntime.Result{}, err
		}
	}

	if result.Err == nil {
		sync.Status.LastReconciledTime = ptr.Get(metav1.Now())
		sync.Status.ObservedGeneration = sync.Generation
		err = r.Status().Update(ctx, sync)
		if err != nil {
			logger.Warn("failed to update reconcile time and generation", "error", err)
		}
	}

	return result.Result, result.Err
}

func (r *BucketSyncReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	sync := &api.BucketSync{}

	err := r.Client.Get(ctx, request.NamespacedName, sync)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch bucket sync", "error", err)
		return controllerruntime.Result{}, err
	}

	if sync.Status.Phase == "" {
		return r.handlePhaseResult(ctx, sync, PhaseResult{NextPhase: api.BucketSyncPhaseInitialize})
	}

	terminals := map[api.BucketSyncPhase]bool{
		api.BucketSyncPhaseFinished: true,
	}
	_, isTerminalPhase := terminals[sync.Status.Phase]
	if isTerminalPhase {
		return r.handlePhaseResult(ctx, sync, PhaseResult{})
	}

	phases := map[api.BucketSyncPhase]PhaseHandler{
		api.BucketSyncPhaseInitialize: r.initialize,
		api.BucketSyncPhaseSync:       r.sync,
		api.BucketSyncPhaseFinalize:   r.finalize,
	}

	handler, hasPhaseHandler := phases[sync.Status.Phase]
	if !hasPhaseHandler {
		logger.Error("invalid phase", "phase", sync.Status.Phase)
		return controllerruntime.Result{}, nil
	}

	result := handler(ctx, sync)
	return r.handlePhaseResult(ctx, sync, result)
}
