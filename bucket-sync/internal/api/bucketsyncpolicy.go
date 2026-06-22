package api

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
type BucketSyncPolicySpec struct {
	// +optional
	JobLabels map[string]string `json:"jobLabels,omitempty"`
	// +kubebuilder:validation:Required
	Source string `json:"source"`
	// +kubebuilder:validation:Required
	Destination string `json:"destination"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^(@(annually|yearly|monthly|weekly|daily|hourly|reboot)|(((\\d+,)+\\d+|(\\d+([/\\-])\\d+)|\\d+|\\*)\\s?){5})$"
	Schedule string `json:"schedule"`
	// +optional
	// +kubebuilder:validation:Minimum=1
	SyncHistoryLimit *int32 `json:"syncHistoryLimit,omitempty"`
	// +optional
	SourceEnv []corev1.EnvVar `json:"sourceEnv,omitempty"`
	// +optional
	DestinationEnv []corev1.EnvVar `json:"destinationEnv,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Last Sync",type=date,JSONPath=`.status.lastSyncTime`
// +kubebuilder:printcolumn:name="Next Sync",type=date,JSONPath=`.status.nextSyncTime`
// +kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.error`
type BucketSyncPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              BucketSyncPolicySpec   `json:"spec"`
	Status            BucketSyncPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type BucketSyncPolicyStatus struct {
	//+optional
	Error *string `json:"error,omitempty"`
	//+optional
	LastReconciledTime *metav1.Time `json:"lastReconciledTime,omitempty"`
	//+optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	//+optional
	NextSyncTime *metav1.Time `json:"nextSyncTime,omitempty"`
	//+optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
type BucketSyncPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []BucketSyncPolicy `json:"items"`
}
