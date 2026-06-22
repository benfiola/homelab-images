package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
type RouterPolicySpec struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type RouterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              RouterPolicySpec   `json:"spec"`
	Status            RouterPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type RouterPolicyStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	LastReconciledTime *metav1.Time       `json:"lastReconciledTime,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
type RouterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []RouterPolicy `json:"items"`
}
