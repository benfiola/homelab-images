package scheme

import (
	"github.com/benfiola/homelab-images/bucket-sync/internal/api"
	"k8s.io/apimachinery/pkg/runtime"
	coreapis "k8s.io/client-go/kubernetes/scheme"
)

func Build() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	err := coreapis.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = api.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	return scheme, nil
}
