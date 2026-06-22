package api

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BucketSyncPhase string

const (
	BucketSyncPhaseInitialize BucketSyncPhase = "Initialize"
	BucketSyncPhaseSync       BucketSyncPhase = "Sync"
	BucketSyncPhaseFinalize   BucketSyncPhase = "Finalize"
	BucketSyncPhaseFinished   BucketSyncPhase = "Finished"
)

// +kubebuilder:object:generate=true
type BucketSyncSpec struct {
	// +optional
	JobLabels map[string]string `json:"jobLabels,omitempty"`
	// +kubebuilder:validation:Required
	Source string `json:"source"`
	// +kubebuilder:validation:Required
	Destination string `json:"destination"`
	// +optional
	SourceEnv []corev1.EnvVar `json:"sourceEnv,omitempty"`
	// +optional
	DestinationEnv []corev1.EnvVar `json:"destinationEnv,omitempty"`
	// +optional
	Policy *string `json:"policy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Start",type=date,JSONPath=`.status.startTime`
// +kubebuilder:printcolumn:name="Finish",type=date,JSONPath=`.status.finishTime`
// +kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.error`
type BucketSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              BucketSyncSpec   `json:"spec"`
	Status            BucketSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type BucketSyncStatus struct {
	//+optional
	Job *string `json:"job,omitempty"`
	//+optional
	Phase BucketSyncPhase `json:"phase,omitempty"`
	//+optional
	Error *string `json:"error,omitempty"`
	//+optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	//+optional
	FinishTime *metav1.Time `json:"finishTime,omitempty"`
	//+optional
	LastReconciledTime *metav1.Time `json:"lastReconciledTime,omitempty"`
	//+optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
type BucketSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BucketSync `json:"items"`
}
