package reconciler

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/bucket-sync/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	regexpValidBucketChars = regexp.MustCompile(`[^a-z0-9.-]+`)
)

type BucketLock struct {
	client.Client
	Namespace string
	Bucket    string
}

func NewBucketLock(c client.Client, namespace string, bucket string) *BucketLock {
	lock := &BucketLock{
		Client:    c,
		Namespace: namespace,
		Bucket:    bucket,
	}
	return lock
}

func (l *BucketLock) configMapName() string {
	name := regexpValidBucketChars.ReplaceAllString(strings.ToLower(l.Bucket), ".")
	name = strings.Trim(name, ".")
	hash := md5.Sum([]byte(l.Bucket))
	hashStr := fmt.Sprintf("%x", hash)[:8]
	return fmt.Sprintf("bucket-sync-lock-%s-%s", name, hashStr)
}

func (l *BucketLock) buildConfigMap(sync *api.BucketSync) *corev1.ConfigMap {
	accessedAt := time.Now().UTC().Format(time.RFC3339)

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Namespace: l.Namespace,
			Name:      l.configMapName(),
			Annotations: map[string]string{
				AnnotationLock: "true",
			},
		},
		Data: map[string]string{
			KeyLockAccessedAt:    accessedAt,
			KeyLockSyncName:      sync.Name,
			KeyLockSyncNamespace: sync.Namespace,
			KeyLockBucket:        l.Bucket,
		},
	}

	return cm
}

func (l *BucketLock) Release(ctx context.Context, sync *api.BucketSync) error {
	cm := &corev1.ConfigMap{}
	err := l.Get(ctx, GetNSNameFrom(l.buildConfigMap(sync)), cm)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}

	if cm.Data[KeyLockSyncName] != sync.Name || cm.Data[KeyLockSyncNamespace] != sync.Namespace {
		return nil
	}

	err = l.Delete(ctx, cm)
	if err != nil {
		return err
	}
	return nil
}

func (l *BucketLock) TryAcquire(ctx context.Context, sync *api.BucketSync) (bool, string, error) {
	cm := l.buildConfigMap(sync)

	logger := logging.FromContext(ctx).With("resource", GetNSNameFrom(cm))

	err := l.Create(ctx, cm)
	if err == nil {
		return true, sync.Name, nil
	}

	if client.IgnoreAlreadyExists(err) != nil {
		return false, "", err
	}

	existing := &corev1.ConfigMap{}
	err = l.Get(ctx, GetNSNameFrom(cm), existing)
	if err != nil {
		return false, "", err
	}

	ownerNs := existing.Data[KeyLockSyncNamespace]
	owner := existing.Data[KeyLockSyncName]
	acquired := owner == sync.Name && ownerNs == sync.Namespace

	if acquired {
		existing.Data[KeyLockAccessedAt] = time.Now().UTC().Format(time.RFC3339)
		err := l.Update(ctx, existing)
		if err != nil {
			logger.Warn("failed to update lock 'accessed-at' timestamp", "error", err)
		}
	}

	return acquired, owner, nil
}
