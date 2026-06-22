package scheme

import (
	"github.com/benfiola/homelab-images/router-policy-sync/internal/api"
	ciliumapis "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Build() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	err := ciliumapis.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = corev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = api.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	return scheme, nil
}
