package reconciler

// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups=bucket-sync.homelab-images.benfiola.com,resources=bucketsyncpolicies,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=bucket-sync.homelab-images.benfiola.com,resources=bucketsyncpolicies/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=bucket-sync.homelab-images.benfiola.com,resources=bucketsyncs,verbs=create;delete;list

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/benfiola/homelab-images/bucket-sync/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AnnotationSyncNow = "bucket-sync.homelab-images.benfiola.com/sync-now"
)

type BucketSyncPolicyReconciler struct {
	client.Client
	CronParser cron.Parser
	Recorder   record.EventRecorder
	Scheme     *runtime.Scheme
}

func (r *BucketSyncPolicyReconciler) Register(manager controllerruntime.Manager) error {
	r.CronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	r.Recorder = manager.GetEventRecorderFor("bucket-sync")

	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&api.BucketSyncPolicy{}).
		Complete(r)
}

func (r *BucketSyncPolicyReconciler) UpdateStatus(ctx context.Context, policy *api.BucketSyncPolicy, now time.Time, errMsg string) {
	logger := logging.FromContext(ctx)

	policy.Status.Error = nil
	if errMsg != "" {
		policy.Status.Error = &errMsg
	}
	policy.Status.LastReconciledTime = &metav1.Time{Time: now}
	policy.Status.ObservedGeneration = policy.Generation

	if err := r.Status().Update(ctx, policy); err != nil {
		logger.Error("failed to update status", "error", err.Error())
	}
}

func (r *BucketSyncPolicyReconciler) sortBucketSyncsByCreation(syncs []api.BucketSync) {
	sort.Slice(syncs, func(i, j int) bool {
		return syncs[i].CreationTimestamp.After(syncs[j].CreationTimestamp.Time)
	})
}

func (r *BucketSyncPolicyReconciler) cleanupOldBucketSyncs(ctx context.Context, policy *api.BucketSyncPolicy) error {
	logger := logging.FromContext(ctx)

	if policy.Spec.SyncHistoryLimit == nil || *policy.Spec.SyncHistoryLimit <= 0 {
		return nil
	}

	syncs := &api.BucketSyncList{}
	err := r.Client.List(ctx, syncs, client.InNamespace(policy.Namespace))
	if err != nil {
		logger.Error("failed to list bucket syncs", "error", err)
		return err
	}

	var policySyncs []api.BucketSync
	for _, sync := range syncs.Items {
		if sync.Spec.Policy != nil && *sync.Spec.Policy == policy.Name {
			policySyncs = append(policySyncs, sync)
		}
	}

	if len(policySyncs) > int(*policy.Spec.SyncHistoryLimit) {
		itemsToDelete := len(policySyncs) - int(*policy.Spec.SyncHistoryLimit)

		r.sortBucketSyncsByCreation(policySyncs)

		for i := range itemsToDelete {
			sync := policySyncs[len(policySyncs)-1-i]
			if err := r.Client.Delete(ctx, &sync); err != nil {
				logger.Error("failed to clean up old bucket sync", "sync", GetNSNameFrom(&sync), "error", err)
				return err
			}
		}
	}

	return nil
}

func (r *BucketSyncPolicyReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	now := time.Now()
	policy := &api.BucketSyncPolicy{}

	err := r.Client.Get(ctx, request.NamespacedName, policy)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch bucket sync policy", "error", err)
		return controllerruntime.Result{}, err
	}

	if policy.DeletionTimestamp != nil {
		controllerutil.RemoveFinalizer(policy, Finalizer)
		err = r.Update(ctx, policy)
		if err != nil {
			logger.Error("failed to remove finalizer during deletion", "error", err)
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(policy, Finalizer) {
		controllerutil.AddFinalizer(policy, Finalizer)
		err = r.Update(ctx, policy)
		if err != nil {
			logger.Error("failed to add finalizer", "error", err)
			return controllerruntime.Result{}, err
		}
	}

	schedule, err := r.CronParser.Parse(policy.Spec.Schedule)
	if err != nil {
		logger.Error("failed to parse schedule", "error", err)
		r.UpdateStatus(ctx, policy, now, err.Error())
		return controllerruntime.Result{}, nil
	}

	shouldTrigger := false
	_, manualTrigger := policy.Annotations[AnnotationSyncNow]
	lastSync := policy.Status.LastSyncTime
	if manualTrigger {
		shouldTrigger = true
	} else if lastSync == nil {
		shouldTrigger = true
	} else {
		nextSync := schedule.Next(lastSync.Time)
		shouldTrigger = now.After(nextSync)
	}

	if !shouldTrigger {
		var nextSyncTime time.Time
		if lastSync == nil {
			nextSyncTime = schedule.Next(time.Now())
		} else {
			nextSyncTime = schedule.Next(lastSync.Time)
		}
		requeueAfter := nextSyncTime.Sub(now)
		return controllerruntime.Result{RequeueAfter: requeueAfter}, nil
	}

	if manualTrigger {
		delete(policy.Annotations, AnnotationSyncNow)
		if err := r.Update(ctx, policy); err != nil {
			logger.Error("failed to update policy annotations", "error", err)
			return controllerruntime.Result{}, err
		}
	}

	name := fmt.Sprintf("%s-%d", policy.Name, now.Unix())
	sync := &api.BucketSync{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: policy.Namespace,
			Name:      name,
		},
		Spec: api.BucketSyncSpec{
			JobLabels:      policy.Spec.JobLabels,
			SourceEnv:      policy.Spec.SourceEnv,
			DestinationEnv: policy.Spec.DestinationEnv,
			Source:         policy.Spec.Source,
			Destination:    policy.Spec.Destination,
			Policy:         &policy.Name,
		},
	}
	err = r.Client.Create(ctx, sync)
	if err != nil {
		logger.Error("failed to create restore resource", "restore", GetNSNameFrom(sync), "error", err)
		if manualTrigger {
			r.Recorder.Event(policy, corev1.EventTypeWarning, "BucketSyncCreationFailed", fmt.Sprintf("failed to create bucket sync resource: %s", err.Error()))
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, err
	}

	if err := r.cleanupOldBucketSyncs(ctx, policy); err != nil {
		logger.Error("failed to cleanup old bucket syncs", "error", err)
	}

	policy.Status.LastSyncTime = &metav1.Time{Time: now}
	policy.Status.NextSyncTime = &metav1.Time{Time: schedule.Next(now)}
	if err := r.Status().Update(ctx, policy); err != nil {
		logger.Error("failed to update last/next sync time in status", "error", err)
		return controllerruntime.Result{}, err
	}

	r.UpdateStatus(ctx, policy, now, "")
	return controllerruntime.Result{}, nil
}
