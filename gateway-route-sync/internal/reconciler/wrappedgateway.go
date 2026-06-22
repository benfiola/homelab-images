package reconciler

// +kubebuilder:rbac:groups=gateway-route-sync.homelab-images.benfiola.com,resources=wrappedgateways,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=gateway-route-sync.homelab-images.benfiola.com,resources=wrappedgateways/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=grpcroutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tlsroutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes,verbs=get;list;watch

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/gateway-route-sync/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/ptr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapis "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	AnnotationPreviousParentRefs = "gateway-route-sync.homelab-images.benfiola.com/previous-parent-refs"

	ConditionTypeReady = "Ready"

	Finalizer = "gateway-route-sync.homelab-images.benfiola.com/finalizer"

	ReasonFinalizerFailed         = "FinalizerFailed"
	ReasonRoutesFetchFailed       = "RoutesFetchFailed"
	ReasonGatewayStatusFailed     = "GatewayStatusFailed"
	ReasonGatewaySyncFailed       = "GatewaySyncFailed"
	ReasonReconciliationSucceeded = "ReconciliationSucceeded"
)

type WrappedGatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *WrappedGatewayReconciler) routeToWrappedGatewayRequests(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := logging.FromContext(ctx)
	parentRefs, err := getParentRefs(obj)
	if err != nil {
		logger.Warn("could not get parent refs", "error", err)
		return nil
	}

	if ann := obj.GetAnnotations()[AnnotationPreviousParentRefs]; ann != "" {
		data := RouteParentRefs{}
		if err := json.Unmarshal([]byte(ann), &data); err == nil {
			parentRefs = append(parentRefs, data.ParentRefs...)
		}
	}

	seen := map[types.NamespacedName]struct{}{}
	var requests []reconcile.Request
	for _, ref := range parentRefs {
		group := "gateway.networking.k8s.io"
		if ref.Group != nil {
			group = string(*ref.Group)
		}
		kind := "Gateway"
		if ref.Kind != nil {
			kind = string(*ref.Kind)
		}
		if group != "gateway.networking.k8s.io" || kind != "Gateway" {
			continue
		}
		namespace := obj.GetNamespace()
		if ref.Namespace != nil {
			namespace = string(*ref.Namespace)
		}
		nn := types.NamespacedName{Namespace: namespace, Name: string(ref.Name)}
		if _, ok := seen[nn]; ok {
			continue
		}
		seen[nn] = struct{}{}
		requests = append(requests, reconcile.Request{NamespacedName: nn})
	}
	return requests
}

func (r *WrappedGatewayReconciler) Register(manager controllerruntime.Manager) error {
	mapper := handler.EnqueueRequestsFromMapFunc(r.routeToWrappedGatewayRequests)
	b := controllerruntime.
		NewControllerManagedBy(manager).
		For(&api.WrappedGateway{}).
		Owns(&gatewayapis.Gateway{})
	return watchRoutes(b, mapper).Complete(r)
}

func (r *WrappedGatewayReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	wgateway := api.WrappedGateway{}
	err := r.Get(ctx, request.NamespacedName, &wgateway)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch wrapped gateway", "error", err)
		return controllerruntime.Result{}, err
	}

	if wgateway.DeletionTimestamp != nil {
		controllerutil.RemoveFinalizer(&wgateway, Finalizer)
		err = r.Update(ctx, &wgateway)
		if err != nil {
			logger.Error("failed to remove finalizer during deletion", "error", err)
			r.setCondition(&wgateway, ReasonFinalizerFailed, err.Error())
			r.Status().Update(ctx, &wgateway)
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&wgateway, Finalizer) {
		controllerutil.AddFinalizer(&wgateway, Finalizer)
		err = r.Update(ctx, &wgateway)
		if err != nil {
			logger.Error("failed to add finalizer", "error", err)
			r.setCondition(&wgateway, ReasonFinalizerFailed, err.Error())
			r.Status().Update(ctx, &wgateway)
			return controllerruntime.Result{}, err
		}
	}

	gateway := gatewayapis.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wgateway.Name,
			Namespace: wgateway.Namespace,
		},
	}

	listeners := []gatewayapis.Listener{}
	routes, err := r.GetRouteData(ctx, &wgateway)
	if err != nil {
		logger.Error("failed to get route data", "error", err)
		r.setCondition(&wgateway, ReasonRoutesFetchFailed, err.Error())
		r.Status().Update(ctx, &wgateway)
		return controllerruntime.Result{}, err
	}

	for _, route := range routes {
		allowedRoutes := &gatewayapis.AllowedRoutes{
			Namespaces: &gatewayapis.RouteNamespaces{
				From: ptr.Get(gatewayapis.NamespacesFromSelector),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": string(route.Namespace),
					},
				},
			},
			Kinds: []gatewayapis.RouteGroupKind{
				{
					Group: ptr.Get(route.Group),
					Kind:  route.Kind,
				},
			},
		}

		name := route.SectionName
		protocol := route.Protocol

		var tls *gatewayapis.ListenerTLSConfig
		if protocol == gatewayapis.HTTPSProtocolType || protocol == gatewayapis.TLSProtocolType {
			certName := fmt.Sprintf("gateway-%s", *route.Hostname)
			certName = strings.ReplaceAll(certName, ".", "-")
			tls = &gatewayapis.ListenerTLSConfig{
				Mode: ptr.Get(gatewayapis.TLSModeTerminate),
				CertificateRefs: []gatewayapis.SecretObjectReference{{
					Name: gatewayapis.ObjectName(certName),
				}},
			}
		}

		listener := gatewayapis.Listener{
			AllowedRoutes: allowedRoutes,
			Name:          name,
			Hostname:      route.Hostname,
			Port:          route.Port,
			Protocol:      protocol,
			TLS:           tls,
		}
		listeners = append(listeners, listener)
	}

	if len(listeners) == 0 {
		logger.Warn("gateway has no listeners - deleting")
		err = r.Delete(ctx, &gateway)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("failed to delete gateway", "error", err)
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &gateway, func() error {
		gateway.Annotations = wgateway.Annotations
		gateway.Labels = wgateway.Labels
		gateway.Spec = gatewayapis.GatewaySpec{
			Addresses:        wgateway.Spec.Addresses,
			GatewayClassName: wgateway.Spec.GatewayClassName,
			Infrastructure:   wgateway.Spec.Infrastructure,
			Listeners:        listeners,
		}
		return controllerutil.SetControllerReference(&wgateway, &gateway, r.Scheme)
	})
	if err != nil {
		logger.Error("failed to create or update gateway", "error", err, "routes", len(routes))
		r.setCondition(&wgateway, ReasonGatewaySyncFailed, err.Error())
		r.Status().Update(ctx, &wgateway)
		return controllerruntime.Result{}, err
	}

	logger.Info("synced gateway", "routes", len(routes))
	wgateway.Status.ObservedGeneration = wgateway.Generation
	wgateway.Status.LastReconciledTime = &metav1.Time{Time: time.Now()}
	r.setCondition(&wgateway, ReasonReconciliationSucceeded, "")
	err = r.Status().Update(ctx, &wgateway)
	if err != nil {
		logger.Error("failed to update wrapped gateway status on success", "error", err)
		r.setCondition(&wgateway, ReasonGatewayStatusFailed, err.Error())
		r.Status().Update(ctx, &wgateway)
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{}, nil
}

func (r *WrappedGatewayReconciler) GetNamespace(o client.Object) string {
	namespace := o.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}
	return namespace
}

type RouteData struct {
	Hostname    *gatewayapis.Hostname
	Kind        gatewayapis.Kind
	Group       gatewayapis.Group
	Namespace   gatewayapis.Namespace
	Port        gatewayapis.PortNumber
	Protocol    gatewayapis.ProtocolType
	SectionName gatewayapis.SectionName
}

func (d RouteData) String() string {
	hostname := ""
	if d.Hostname != nil {
		hostname = string(*d.Hostname)
	}
	vals := []string{
		string(d.SectionName),
		string(d.Namespace),
		string(d.Group),
		string(d.Kind),
		hostname,
		strconv.Itoa(int(d.Port)),
	}
	val := strings.Join(vals, "/")
	return val
}

func (r *WrappedGatewayReconciler) GetRouteData(ctx context.Context, gateway *api.WrappedGateway) ([]RouteData, error) {
	logger := logging.FromContext(ctx)

	allRoutes, err := listRoutes(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	dataMap := map[string]RouteData{}
	addRoute := func(route client.Object) {
		refs, _ := getParentRefs(route)
		gvk := route.GetObjectKind().GroupVersionKind()
		group := gatewayapis.Group(gvk.Group)
		kind := gatewayapis.Kind(gvk.Kind)
		namespace := gatewayapis.Namespace(r.GetNamespace(route))
		protocol, err := getProtocol(route)
		if err != nil {
			logger.Error("failed to determine protocol", "route-kind", kind, "route", GetNSNameFrom(route), "error", err)
			return
		}
		hostnames := getHostnames(route)
		for _, ref := range refs {
			refGroup := "gateway.networking.k8s.io"
			if ref.Group != nil {
				refGroup = string(*ref.Group)
			}
			refKind := "Gateway"
			if ref.Kind != nil {
				refKind = string(*ref.Kind)
			}
			refNamespace := r.GetNamespace(route)
			if ref.Namespace != nil {
				refNamespace = string(*ref.Namespace)
			}
			if refGroup != "gateway.networking.k8s.io" || refKind != "Gateway" || refNamespace != gateway.Namespace || string(ref.Name) != gateway.Name {
				continue
			}
			if ref.SectionName == nil {
				continue
			}
			sectionName := *ref.SectionName
			ports := []int{}
			if ref.Port != nil {
				ports = []int{int(*ref.Port)}
			} else {
				ports = getDefaultPorts(route)
			}
			var hostnamePtr *gatewayapis.Hostname
			for _, hostname := range hostnames {
				if hostname != "" {
					hostnamePtr = ptr.Get(hostname)
					break
				}
			}
			for _, port := range ports {
				item := RouteData{
					Group:       group,
					Hostname:    hostnamePtr,
					Kind:        kind,
					Namespace:   namespace,
					Port:        gatewayapis.PortNumber(port),
					Protocol:    protocol,
					SectionName: sectionName,
				}
				dataMap[string(sectionName)] = item
			}
		}
	}

	for _, route := range allRoutes {
		addRoute(route)
	}

	keys := make([]string, 0, len(dataMap))
	for key := range dataMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	data := make([]RouteData, 0, len(keys))
	for _, key := range keys {
		data = append(data, dataMap[key])
	}

	return data, nil
}

func (r *WrappedGatewayReconciler) setCondition(wg *api.WrappedGateway, reason string, message string) {
	cstatus := metav1.ConditionFalse
	if reason == ReasonReconciliationSucceeded {
		cstatus = metav1.ConditionTrue
	}

	meta.SetStatusCondition(&wg.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             cstatus,
		ObservedGeneration: wg.Generation,
		Reason:             reason,
		Message:            message,
	})
}
