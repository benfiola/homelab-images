package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
type PVCRestoreSpec struct {
	// +kubebuilder:validation:Required
	PVC string `json:"pvc"`
	//+optional
	Previous *int32 `json:"previous,omitempty"`
	// +kubebuilder:validation:Format="date-time"
	//+optional
	RestoreAsOf *string `json:"restoreAsOf,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Error",type=string,JSONPath=`.status.error`
type PVCRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PVCRestoreSpec   `json:"spec"`
	Status            PVCRestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type PVCOwner struct {
	ResourceRef `json:",inline"`
	Replicas    int32 `json:"replicas"`
}

// +kubebuilder:object:generate=true
type PVCRestoreStatus struct {
	//+optional
	Phase string `json:"phase,omitempty"`
	//+optional
	PVCOwners []PVCOwner `json:"pvcOwners,omitempty"`
	//+optional
	ReplicationSource *string `json:"replicationSource,omitempty"`
	//+optional
	ReplicationDestination *string `json:"replicationDestination,omitempty"`
	//+optional
	Error *string `json:"error,omitempty"`
	//+optional
	LastReconciledTime *metav1.Time `json:"lastReconciledTime,omitempty"`
	//+optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
type PVCRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []PVCRestore `json:"items"`
}
