package reconciler

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=grpcroutes,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tlsroutes,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes,verbs=get;list;patch;update;watch

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayapis "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapisv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func getProtocol(o client.Object) (gatewayapis.ProtocolType, error) {
	switch o.(type) {
	case *gatewayapis.GRPCRoute:
		return gatewayapis.HTTPSProtocolType, nil
	case *gatewayapis.HTTPRoute:
		return gatewayapis.HTTPSProtocolType, nil
	case *gatewayapisv1a2.TCPRoute:
		return gatewayapis.TCPProtocolType, nil
	case *gatewayapisv1a2.TLSRoute:
		return gatewayapis.TLSProtocolType, nil
	case *gatewayapisv1a2.UDPRoute:
		return gatewayapis.UDPProtocolType, nil
	}
	return "", fmt.Errorf("unknown resource: %s", o.GetObjectKind())
}

func getParentRefs(o client.Object) ([]gatewayapis.ParentReference, error) {
	switch v := o.(type) {
	case *gatewayapis.GRPCRoute:
		return v.Spec.ParentRefs, nil
	case *gatewayapis.HTTPRoute:
		return v.Spec.ParentRefs, nil
	case *gatewayapisv1a2.TCPRoute:
		return v.Spec.ParentRefs, nil
	case *gatewayapisv1a2.TLSRoute:
		return v.Spec.ParentRefs, nil
	case *gatewayapisv1a2.UDPRoute:
		return v.Spec.ParentRefs, nil
	}
	return nil, fmt.Errorf("unknown resource: %s", o.GetObjectKind())
}

func getHostnames(o client.Object) []gatewayapis.Hostname {
	switch v := o.(type) {
	case *gatewayapis.GRPCRoute:
		return v.Spec.Hostnames
	case *gatewayapis.HTTPRoute:
		return v.Spec.Hostnames
	case *gatewayapisv1a2.TLSRoute:
		return v.Spec.Hostnames
	default:
		return []gatewayapis.Hostname{""}
	}
}

func getDefaultPorts(o client.Object) []int {
	switch o.(type) {
	case *gatewayapis.GRPCRoute:
		return []int{443}
	case *gatewayapis.HTTPRoute:
		return []int{443}
	case *gatewayapisv1a2.TLSRoute:
		return []int{443}
	}
	return []int{}
}

func listRoutes(ctx context.Context, c client.Client) ([]client.Object, error) {
	logger := logging.FromContext(ctx)
	var routes []client.Object

	grpcRoutes := gatewayapis.GRPCRouteList{}
	if err := c.List(ctx, &grpcRoutes); err != nil {
		return nil, fmt.Errorf("listing GRPCRoutes: %w", err)
	}
	for i := range grpcRoutes.Items {
		routes = append(routes, &grpcRoutes.Items[i])
	}

	httpRoutes := gatewayapis.HTTPRouteList{}
	if err := c.List(ctx, &httpRoutes); err != nil {
		return nil, fmt.Errorf("listing HTTPRoutes: %w", err)
	}
	for i := range httpRoutes.Items {
		routes = append(routes, &httpRoutes.Items[i])
	}

	tcpRoutes := gatewayapisv1a2.TCPRouteList{}
	if err := c.List(ctx, &tcpRoutes); err != nil {
		return nil, fmt.Errorf("listing TCPRoutes: %w", err)
	}
	for i := range tcpRoutes.Items {
		routes = append(routes, &tcpRoutes.Items[i])
	}

	tlsRoutes := gatewayapisv1a2.TLSRouteList{}
	if err := c.List(ctx, &tlsRoutes); err != nil {
		return nil, fmt.Errorf("listing TLSRoutes: %w", err)
	}
	for i := range tlsRoutes.Items {
		routes = append(routes, &tlsRoutes.Items[i])
	}

	udpRoutes := gatewayapisv1a2.UDPRouteList{}
	if err := c.List(ctx, &udpRoutes); err != nil {
		return nil, fmt.Errorf("listing UDPRoutes: %w", err)
	}
	for i := range udpRoutes.Items {
		routes = append(routes, &udpRoutes.Items[i])
	}

	logger.Debug("listed routes", "count", len(routes))
	return routes, nil
}

func watchRoutes(b *controllerruntime.Builder, h handler.EventHandler) *controllerruntime.Builder {
	for _, obj := range []client.Object{
		&gatewayapis.GRPCRoute{},
		&gatewayapis.HTTPRoute{},
		&gatewayapisv1a2.TCPRoute{},
		&gatewayapisv1a2.TLSRoute{},
		&gatewayapisv1a2.UDPRoute{},
	} {
		b = b.Watches(obj, h)
	}
	return b
}

type Adapter reconcile.TypedReconciler[controllerruntime.Request]

func getAdapters(r *RouteReconciler) map[client.Object]Adapter {
	return map[client.Object]Adapter{
		&gatewayapis.GRPCRoute{}:    &GRPCRouteReconcilerAdapter{RouteReconciler: r},
		&gatewayapis.HTTPRoute{}:    &HTTPRouteReconcilerAdapter{RouteReconciler: r},
		&gatewayapisv1a2.TCPRoute{}: &TCPRouteReconcilerAdapter{RouteReconciler: r},
		&gatewayapisv1a2.TLSRoute{}: &TLSRouteReconcilerAdapter{RouteReconciler: r},
		&gatewayapisv1a2.UDPRoute{}: &UDPRouteReconcilerAdapter{RouteReconciler: r},
	}
}

type RouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *RouteReconciler) Register(manager controllerruntime.Manager) error {
	for route, adapter := range getAdapters(r) {
		err := controllerruntime.NewControllerManagedBy(manager).For(route).Complete(adapter)
		if err != nil {
			return err
		}
	}

	return nil
}

type RouteParentRefs struct {
	ParentRefs []gatewayapis.ParentReference `json:",inline"`
}

func (r *RouteReconciler) setPreviousParentRefs(_ context.Context, route client.Object, refs []gatewayapis.ParentReference) {
	data := RouteParentRefs{ParentRefs: refs}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	if route.GetAnnotations() == nil {
		route.SetAnnotations(map[string]string{})
	}
	route.GetAnnotations()[AnnotationPreviousParentRefs] = string(dataBytes)
}

func (r *RouteReconciler) getPreviousParentRefs(_ context.Context, route client.Object) []gatewayapis.ParentReference {
	annotations := route.GetAnnotations()
	if annotations == nil {
		return []gatewayapis.ParentReference{}
	}

	dataStr, ok := annotations[AnnotationPreviousParentRefs]
	if !ok {
		return []gatewayapis.ParentReference{}
	}

	data := RouteParentRefs{}
	err := json.Unmarshal([]byte(dataStr), &data)
	if err != nil {
		return []gatewayapis.ParentReference{}
	}

	return data.ParentRefs
}

func (r *RouteReconciler) ReconcileRoute(ctx context.Context, route client.Object) (controllerruntime.Result, error) {
	logger := logging.FromContext(ctx)

	if route.GetDeletionTimestamp() != nil {
		controllerutil.RemoveFinalizer(route, Finalizer)
		err := r.Update(ctx, route)
		if err != nil {
			logger.Error("failed to remove finalizer during deletion", "error", err)
			return controllerruntime.Result{}, err
		}
		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(route, Finalizer) {
		controllerutil.AddFinalizer(route, Finalizer)
		err := r.Update(ctx, route)
		if err != nil {
			logger.Error("failed to add finalizer", "error", err)
			return controllerruntime.Result{}, err
		}
	}

	previous := r.getPreviousParentRefs(ctx, route)
	current, err := getParentRefs(route)
	if err != nil {
		logger.Error("failed to get parent refs", "error", err)
		return controllerruntime.Result{}, err
	}
	if slices.Equal(previous, current) {
		return controllerruntime.Result{}, nil
	}

	r.setPreviousParentRefs(ctx, route, current)
	if err := r.Update(ctx, route); err != nil {
		logger.Error("failed to update resource", "error", err)
		return controllerruntime.Result{}, err
	}

	return controllerruntime.Result{}, nil
}

type HTTPRouteReconcilerAdapter struct {
	*RouteReconciler
}

func (r *HTTPRouteReconcilerAdapter) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	route := gatewayapis.HTTPRoute{}
	err := r.Get(ctx, request.NamespacedName, &route)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch http route", "error", err)
		return controllerruntime.Result{}, err
	}

	return r.ReconcileRoute(ctx, &route)
}

type GRPCRouteReconcilerAdapter struct {
	*RouteReconciler
}

func (r *GRPCRouteReconcilerAdapter) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	route := gatewayapis.GRPCRoute{}
	err := r.Get(ctx, request.NamespacedName, &route)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch grpc route", "error", err)
		return controllerruntime.Result{}, err
	}

	return r.ReconcileRoute(ctx, &route)
}

type TCPRouteReconcilerAdapter struct {
	*RouteReconciler
}

func (r *TCPRouteReconcilerAdapter) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	route := gatewayapisv1a2.TCPRoute{}
	err := r.Get(ctx, request.NamespacedName, &route)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch tcp route", "error", err)
		return controllerruntime.Result{}, err
	}

	return r.ReconcileRoute(ctx, &route)
}

type TLSRouteReconcilerAdapter struct {
	*RouteReconciler
}

func (r *TLSRouteReconcilerAdapter) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	route := gatewayapisv1a2.TLSRoute{}
	err := r.Get(ctx, request.NamespacedName, &route)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch tls route", "error", err)
		return controllerruntime.Result{}, err
	}

	return r.ReconcileRoute(ctx, &route)
}

type UDPRouteReconcilerAdapter struct {
	*RouteReconciler
}

func (r *UDPRouteReconcilerAdapter) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	route := gatewayapisv1a2.UDPRoute{}
	err := r.Get(ctx, request.NamespacedName, &route)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch udp route", "error", err)
		return controllerruntime.Result{}, err
	}

	return r.ReconcileRoute(ctx, &route)
}
