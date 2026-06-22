package internal

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/mikrotik"
	"github.com/benfiola/homelab-images/router-policy-sync/internal/reconciler"
	"github.com/benfiola/homelab-images/router-policy-sync/internal/scheme"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/clientcmd"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type Opts struct {
	HealthAddress    string
	Kubeconfig       string
	LeaderElection   bool
	MetricsAddress   string
	MikrotikBaseURL  string
	MikrotikUsername string
	MikrotikPassword string
	ReservedCIDRs    []string
	SyncInterval     time.Duration
}

type Controller struct {
	Manager controllerruntime.Manager
}

func New(opts *Opts) (*Controller, error) {
	if opts.MikrotikBaseURL == "" {
		return nil, fmt.Errorf("mikrotik base URL required")
	}
	if opts.MikrotikUsername == "" {
		return nil, fmt.Errorf("mikrotik username required")
	}
	if opts.MikrotikPassword == "" {
		return nil, fmt.Errorf("mikrotik password required")
	}

	reservedCIDRStrs := []string{
		"0.0.0.0/8",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"224.0.0.0/4",
		"255.255.255.255/32",
	}
	reservedCIDRStrs = append(reservedCIDRStrs, opts.ReservedCIDRs...)
	reservedCIDRs := []*net.IPNet{}
	for _, reservedCIDRStr := range reservedCIDRStrs {
		_, cidr, err := net.ParseCIDR(reservedCIDRStr)
		if err != nil {
			return nil, fmt.Errorf("invalid reserved CIDR %q: %w", reservedCIDRStr, err)
		}
		reservedCIDRs = append(reservedCIDRs, cidr)
	}

	healthAddress := opts.HealthAddress
	if healthAddress == "" {
		healthAddress = ":8081"
	}
	metricsAddress := opts.MetricsAddress
	if metricsAddress == "" {
		metricsAddress = ":8080"
	}
	syncInterval := opts.SyncInterval
	if syncInterval == 0 {
		syncInterval = 5 * time.Minute
	}

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", opts.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	mikrotikClient, err := mikrotik.New(&mikrotik.Opts{
		BaseURL:  opts.MikrotikBaseURL,
		Password: opts.MikrotikPassword,
		Username: opts.MikrotikUsername,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build mikrotik client: %w", err)
	}

	builtScheme, err := scheme.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build scheme: %w", err)
	}

	webhookServer := webhook.NewServer(webhook.Options{Port: 0})
	manager, err := controllerruntime.NewManager(kubeConfig, controllerruntime.Options{
		HealthProbeBindAddress: healthAddress,
		LeaderElection:         opts.LeaderElection,
		LeaderElectionID:       "router-policy-sync.homelab-images.benfiola.com",
		Metrics:                server.Options{BindAddress: metricsAddress},
		WebhookServer:          webhookServer,
		Scheme:                 builtScheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager: %w", err)
	}

	client := manager.GetClient()
	reconcilers := []reconciler.Reconciler{
		&reconciler.CiliumPolicyReconciler{
			Client: client,
			Scheme: builtScheme,
		},
		&reconciler.RouterPolicyReconciler{
			Client:        client,
			Mikrotik:      mikrotikClient,
			Scheme:        builtScheme,
			ReservedCIDRs: reservedCIDRs,
			SyncInterval:  syncInterval,
		},
	}
	for index, rec := range reconcilers {
		err = rec.Register(manager)
		if err != nil {
			return nil, fmt.Errorf("failed to register reconciler %d: %w", index, err)
		}
	}

	return &Controller{
		Manager: manager,
	}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Info("setting controller-runtime logger")
	crLogger := logr.FromSlogHandler(logger.Handler()).WithName("controller-runtime")
	controllerruntime.SetLogger(crLogger)

	logger.Info("adding probes")
	err := c.Manager.AddHealthzCheck("ping", healthz.Ping)
	if err != nil {
		logger.Error("failed to setup liveness probe", "error", err)
	}

	readyz := func(req *http.Request) error {
		val := c.Manager.GetCache().WaitForCacheSync(req.Context())
		if !val {
			logger.Warn("readyz cache sync check failed")
			return fmt.Errorf("readyz cache sync check failed")
		}
		return nil
	}
	err = c.Manager.AddReadyzCheck("caches", readyz)
	if err != nil {
		logger.Error("failed to setup readiness probe", "error", err)
	}

	logger.Info("starting controller")
	err = c.Manager.Start(ctx)
	if err != nil {
		logger.Error("controller manager exited with error", "error", err)
		return err
	}

	return nil
}
