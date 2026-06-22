package reconciler

// +kubebuilder:rbac:groups=router-policy-sync.homelab-images.benfiola.com,resources=routerpolicies,verbs=get;list;patch;update;watch
// +kubebuilder:rbac:groups=router-policy-sync.homelab-images.benfiola.com,resources=routerpolicies/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=cilium.io,resources=ciliumclusterwidenetworkpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/benfiola/homelab-images/router-policy-sync/internal/api"
	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/benfiola/homelab-images/shared/pkg/mikrotik"
	ciliumapis "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	ciliumpolicyapis "github.com/cilium/cilium/pkg/policy/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	FirewalFilterMarker = "router-policy-sync::marker"
	ConditionTypeReady  = "Ready"
)

type ProtocolPorts struct {
	Protocol string
	Ports    []int
}

type RouterPolicyReconciler struct {
	client.Client
	Mikrotik      *mikrotik.Client
	ReservedCIDRs []*net.IPNet
	Scheme        *runtime.Scheme
	SyncInterval  time.Duration
}

func (r *RouterPolicyReconciler) Register(manager controllerruntime.Manager) error {
	return controllerruntime.
		NewControllerManagedBy(manager).
		For(&api.RouterPolicy{}).
		Complete(r)
}

func (r *RouterPolicyReconciler) updateStatus(ctx context.Context, policy *api.RouterPolicy, conditionStatus v1.ConditionStatus, reason, message string) error {
	logger := logging.FromContext(ctx)

	policy.Status.LastReconciledTime = &v1.Time{Time: v1.Now().Time}
	policy.Status.ObservedGeneration = policy.Generation

	meta.SetStatusCondition(&policy.Status.Conditions, v1.Condition{
		Type:               ConditionTypeReady,
		Status:             conditionStatus,
		ObservedGeneration: policy.Generation,
		Reason:             reason,
		Message:            message,
	})

	err := r.Status().Update(ctx, policy)
	if err != nil {
		logger.Error("failed to update status", "error", err)
		return err
	}

	return nil
}

func (r *RouterPolicyReconciler) findPodsMatching(ctx context.Context, selector ciliumpolicyapis.EndpointSelector) ([]corev1.Pod, error) {
	if selector.LabelSelector == nil || selector.LabelSelector.MatchLabels == nil {
		return nil, fmt.Errorf("endpoint selector has no label selector")
	}

	const k8sPrefix = "k8s:"
	const namespaceLabel = "io.kubernetes.pod.namespace"
	k8sSelector := map[string]string{}
	var namespace string
	for key, value := range selector.LabelSelector.MatchLabels {
		labelKey := strings.TrimPrefix(key, k8sPrefix)
		if labelKey == namespaceLabel {
			namespace = value
			continue
		}
		k8sSelector[labelKey] = value
	}
	if namespace == "" {
		return nil, fmt.Errorf("endpoint selector missing namespace label (k8s:%s)", namespaceLabel)
	}
	if len(k8sSelector) == 0 {
		return nil, fmt.Errorf("endpoint selector has only namespace label, no pod labels to match")
	}

	labelSelector := labels.SelectorFromSet(labels.Set(k8sSelector))

	podList := &corev1.PodList{}
	err := r.List(ctx, podList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func (r *RouterPolicyReconciler) findLoadBalancerServiceForPod(ctx context.Context, pod *corev1.Pod) (*corev1.Service, error) {
	services := &corev1.ServiceList{}
	err := r.List(ctx, services, client.InNamespace(pod.Namespace))
	if err != nil {
		return nil, err
	}

	for i := range services.Items {
		svc := &services.Items[i]
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}

		selector := labels.SelectorFromSet(labels.Set(svc.Spec.Selector))
		if selector.Matches(labels.Set(pod.Labels)) {
			return svc, nil
		}
	}

	return nil, fmt.Errorf("no loadbalancer service found for pod %s/%s", pod.Namespace, pod.Name)
}

func (r *RouterPolicyReconciler) extractLoadBalancerIPs(service *corev1.Service) ([]string, error) {
	var ips []string
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("service %s/%s has no loadbalancer ingress ips", service.Namespace, service.Name)
	}

	return ips, nil
}

func (r *RouterPolicyReconciler) discoverLoadBalancerService(ctx context.Context, ciliumPolicy *ciliumapis.CiliumClusterwideNetworkPolicy) (*corev1.Service, error) {
	logger := logging.FromContext(ctx)

	selector := ciliumPolicy.Spec.EndpointSelector

	pods, err := r.findPodsMatching(ctx, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching pods: %w", err)
	}
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods matched the endpoint selector")
	}

	for i := range pods {
		pod := &pods[i]
		service, err := r.findLoadBalancerServiceForPod(ctx, pod)
		if err != nil {
			logger.Warn("skipped pod, no loadbalancer service found", "pod", GetNSNameFrom(pod), "error", err)
			continue
		}

		return service, nil
	}
	return nil, fmt.Errorf("no loadbalancer service discovered from matched pods")
}

func (r *RouterPolicyReconciler) routerFirewallIdFor(policy *api.RouterPolicy) string {
	return fmt.Sprintf("router-policy-sync::%s", policy.Name)
}

func (r *RouterPolicyReconciler) findRouterFirewalFilter(ctx context.Context, comment string) (*mikrotik.FirewallFilter, int, error) {
	filters, err := r.Mikrotik.ListFirewallFilters(ctx)
	if err != nil {
		return nil, 0, err
	}

	for index, filter := range filters {
		if filter.Comment == comment {
			return filter, index, nil
		}
	}

	return nil, 0, nil
}

func (r *RouterPolicyReconciler) syncAddressLists(ctx context.Context, id string, policyIPs []string) error {
	desired := map[string]bool{}
	for _, ip := range policyIPs {
		desired[ip] = true
	}

	addressLists, err := r.Mikrotik.ListFirewallAddressLists(ctx)
	if err != nil {
		return err
	}

	exists := map[string]*mikrotik.FirewallAddressList{}
	for _, addressList := range addressLists {
		if addressList.List != id {
			continue
		}

		if _, ok := desired[addressList.Address]; ok {
			exists[addressList.Address] = addressList
			continue
		}

		err = r.Mikrotik.DeleteFirewallAddressList(ctx, addressList.ID)
		if err != nil {
			return err
		}
	}

	for address := range desired {
		if _, ok := exists[address]; ok {
			continue
		}

		if err := r.Mikrotik.CreateFirewallAddressList(ctx, &mikrotik.FirewallAddressList{
			List:    id,
			Address: address,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *RouterPolicyReconciler) syncFirewallFilter(ctx context.Context, id string, protocolPorts []ProtocolPorts) error {
	prefix := id + "::"

	desired := map[string]bool{}
	for _, pp := range protocolPorts {
		comment := fmt.Sprintf("%s::%s", id, pp.Protocol)
		desired[comment] = true
	}

	filters, err := r.Mikrotik.ListFirewallFilters(ctx)
	if err != nil {
		return err
	}

	exists := map[string]*mikrotik.FirewallFilter{}
	for _, filter := range filters {
		if !strings.HasPrefix(filter.Comment, prefix) {
			continue
		}

		if desired[filter.Comment] {
			exists[filter.Comment] = filter
			continue
		}

		if err := r.Mikrotik.DeleteFirewallFilter(ctx, filter.ID); err != nil {
			return err
		}
	}

	marker, index, err := r.findRouterFirewalFilter(ctx, FirewalFilterMarker)
	if err != nil {
		return err
	}
	if marker == nil {
		return fmt.Errorf("firewall filter with comment %s not found", FirewalFilterMarker)
	}

	for _, pp := range protocolPorts {
		comment := fmt.Sprintf("%s::%s", id, pp.Protocol)

		_, ok := exists[comment]
		if ok {
			continue
		}

		portStrs := []string{}
		for _, port := range pp.Ports {
			portStrs = append(portStrs, strconv.Itoa(port))
		}

		filterRule := &mikrotik.FirewallFilter{
			Action:         "accept",
			Chain:          marker.Chain,
			Comment:        comment,
			PlaceBefore:    strconv.Itoa(index + 1),
			SrcAddressList: id,
			Protocol:       pp.Protocol,
			DstPort:        strings.Join(portStrs, ","),
		}

		if err := r.Mikrotik.InsertFirewallFilter(ctx, filterRule); err != nil {
			return err
		}
	}

	return nil
}

func (r *RouterPolicyReconciler) syncFirewallNat(ctx context.Context, id string, protocolPorts []ProtocolPorts, gatewayIP string) error {
	if gatewayIP == "" {
		return fmt.Errorf("no gateway ip provided")
	}

	prefix := id + "::"

	desired := map[string]bool{}
	for _, pp := range protocolPorts {
		for _, port := range pp.Ports {
			comment := fmt.Sprintf("%s::%s::%d", id, pp.Protocol, port)
			desired[comment] = true
		}
	}

	nats, err := r.Mikrotik.ListFirewallNats(ctx)
	if err != nil {
		return err
	}

	exists := map[string]*mikrotik.FirewallNat{}
	for _, nat := range nats {
		if !strings.HasPrefix(nat.Comment, prefix) {
			continue
		}

		if desired[nat.Comment] {
			exists[nat.Comment] = nat
			continue
		}

		if err := r.Mikrotik.DeleteFirewallNat(ctx, nat.ID); err != nil {
			return err
		}
	}

	for _, pp := range protocolPorts {
		for _, port := range pp.Ports {
			comment := fmt.Sprintf("%s::%s::%d", id, pp.Protocol, port)

			nat, natExists := exists[comment]

			portStr := strconv.Itoa(port)
			natRule := &mikrotik.FirewallNat{
				Action:         "dst-nat",
				Chain:          "dstnat",
				Comment:        comment,
				SrcAddressList: id,
				ToAddresses:    gatewayIP,
				Protocol:       pp.Protocol,
				DstPort:        portStr,
				ToPort:         portStr,
			}

			if !natExists {
				err = r.Mikrotik.InsertFirewallNat(ctx, natRule)
				if err != nil {
					return err
				}

				continue
			}

			if nat.ToAddresses != natRule.ToAddresses || nat.DstPort != natRule.DstPort || nat.Protocol != natRule.Protocol || nat.ToPort != natRule.ToPort {
				nat.ToAddresses = natRule.ToAddresses
				nat.DstPort = natRule.DstPort
				nat.Protocol = natRule.Protocol
				nat.ToPort = natRule.ToPort
				err = r.Mikrotik.UpdateFirewallNat(ctx, nat)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *RouterPolicyReconciler) syncFirewallRules(ctx context.Context, policy *api.RouterPolicy, protocolPorts []ProtocolPorts, policyIPs []string, gatewayIP string) error {
	id := r.routerFirewallIdFor(policy)

	if err := r.syncFirewallFilter(ctx, id, protocolPorts); err != nil {
		return fmt.Errorf("failed to sync firewall filter: %w", err)
	}

	if err := r.syncFirewallNat(ctx, id, protocolPorts, gatewayIP); err != nil {
		return fmt.Errorf("failed to sync firewall nat: %w", err)
	}

	err := r.syncAddressLists(ctx, id, policyIPs)
	if err != nil {
		return fmt.Errorf("failed to sync firewall address list: %w", err)
	}

	return nil
}

func (r *RouterPolicyReconciler) deleteRouterFirewallRules(ctx context.Context, policy *api.RouterPolicy) error {
	id := r.routerFirewallIdFor(policy)
	prefix := id + "::"

	filters, err := r.Mikrotik.ListFirewallFilters(ctx)
	if err != nil {
		return err
	}
	for _, filter := range filters {
		if strings.HasPrefix(filter.Comment, prefix) {
			if err := r.Mikrotik.DeleteFirewallFilter(ctx, filter.ID); err != nil {
				return err
			}
		}
	}

	addressLists, err := r.Mikrotik.ListFirewallAddressLists(ctx)
	if err != nil {
		return err
	}
	for _, addressList := range addressLists {
		if addressList.List == id {
			if err := r.Mikrotik.DeleteFirewallAddressList(ctx, addressList.ID); err != nil {
				return err
			}
		}
	}

	nats, err := r.Mikrotik.ListFirewallNats(ctx)
	if err != nil {
		return err
	}
	for _, nat := range nats {
		if strings.HasPrefix(nat.Comment, prefix) {
			if err := r.Mikrotik.DeleteFirewallNat(ctx, nat.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *RouterPolicyReconciler) extractProtocolPorts(service *corev1.Service) ([]ProtocolPorts, error) {
	if len(service.Spec.Ports) == 0 {
		return nil, fmt.Errorf("service %s/%s has no ports defined", service.Namespace, service.Name)
	}

	protocolPortsMap := map[string]map[int]bool{}

	for _, port := range service.Spec.Ports {
		protocol := string(port.Protocol)
		if protocol == "" {
			protocol = "tcp"
		}

		switch protocol {
		case "TCP":
			protocol = "tcp"
		case "UDP":
			protocol = "udp"
		default:
			return nil, fmt.Errorf("service %s/%s port %s: unsupported protocol %q", service.Namespace, service.Name, port.Name, protocol)
		}

		portNum := int(port.Port)
		if portNum <= 0 || portNum > 65535 {
			return nil, fmt.Errorf("service %s/%s port %s: invalid port number %d", service.Namespace, service.Name, port.Name, portNum)
		}

		if _, exists := protocolPortsMap[protocol]; !exists {
			protocolPortsMap[protocol] = map[int]bool{}
		}
		protocolPortsMap[protocol][portNum] = true
	}

	var result []ProtocolPorts
	for protocol, portSet := range protocolPortsMap {
		pp := ProtocolPorts{
			Protocol: protocol,
		}
		for port := range portSet {
			pp.Ports = append(pp.Ports, port)
		}
		result = append(result, pp)
	}

	return result, nil
}

func (r *RouterPolicyReconciler) extractPolicyIPs(ciliumPolicy *ciliumapis.CiliumClusterwideNetworkPolicy) ([]string, error) {
	if ciliumPolicy.Spec == nil || len(ciliumPolicy.Spec.Ingress) == 0 {
		return nil, fmt.Errorf("cilium policy has no ingress rules")
	}

	var ips []string
	for _, rule := range ciliumPolicy.Spec.Ingress {
		for _, cidr := range rule.FromCIDR {
			cidrStr := string(cidr)
			_, parsedCIDR, err := net.ParseCIDR(cidrStr)
			if err != nil {
				return nil, fmt.Errorf("invalid cidr %q: %w", cidrStr, err)
			}

			ones, bits := parsedCIDR.Mask.Size()
			if ones != bits {
				return nil, fmt.Errorf("cidr %q is not a /32", cidrStr)
			}

			ip := parsedCIDR.IP
			for _, reservedCIDR := range r.ReservedCIDRs {
				if reservedCIDR.Contains(ip) {
					return nil, fmt.Errorf("ip %s is in reserved range %s", ip.String(), reservedCIDR.String())
				}
			}

			ips = append(ips, ip.String())
		}
	}

	return ips, nil
}

func (r *RouterPolicyReconciler) Reconcile(pctx context.Context, request controllerruntime.Request) (controllerruntime.Result, error) {
	logger := logging.FromContext(pctx).With("resource", request.NamespacedName)
	ctx := logging.WithLogger(pctx, logger)

	policy := &api.RouterPolicy{}
	err := r.Get(ctx, request.NamespacedName, policy)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return controllerruntime.Result{}, nil
		}
		logger.Error("failed to fetch router policy", "error", err)
		return controllerruntime.Result{}, err
	}

	if policy.DeletionTimestamp != nil {
		if !controllerutil.ContainsFinalizer(policy, Finalizer) {
			return controllerruntime.Result{}, nil
		}

		routerErr := r.deleteRouterFirewallRules(ctx, policy)
		if routerErr != nil {
			logger.Error("failed to delete router firewall rules", "error", routerErr)
			return controllerruntime.Result{}, routerErr
		}

		controllerutil.RemoveFinalizer(policy, Finalizer)
		err = r.Update(ctx, policy)
		if err != nil {
			logger.Error("failed to remove finalizer", "error", err)
			return controllerruntime.Result{}, err
		}

		return controllerruntime.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(policy, Finalizer) {
		controllerutil.AddFinalizer(policy, Finalizer)
		err = r.Update(ctx, policy)
		if err != nil {
			logger.Error("failed to add finalizer", "error", err)
			return controllerruntime.Result{}, err
		}
	}

	if _, ok := policy.Annotations[AnnotationRefresh]; ok {
		delete(policy.Annotations, AnnotationRefresh)
		err = r.Update(ctx, policy)
		if err != nil {
			logger.Error("failed to remove refresh annotation", "error", err)
			return controllerruntime.Result{}, err
		}
	}

	ciliumPolicy := &ciliumapis.CiliumClusterwideNetworkPolicy{}
	err = r.Get(ctx, client.ObjectKey{Name: policy.Name}, ciliumPolicy)
	if err != nil {
		logger.Error("failed to fetch cilium policy", "error", err)
		statusErr := r.updateStatus(ctx, policy, v1.ConditionFalse, "ValidationFailed", err.Error())
		if statusErr != nil {
			logger.Error("failed to update status after validation failure", "error", statusErr)
		}
		return controllerruntime.Result{}, err
	}

	policyIPs, err := r.extractPolicyIPs(ciliumPolicy)
	if err != nil {
		logger.Error("failed to extract policy ips from cilium policy", "error", err)
		statusErr := r.updateStatus(ctx, policy, v1.ConditionFalse, "ValidationFailed", err.Error())
		if statusErr != nil {
			logger.Error("failed to update status", "error", statusErr)
		}
		return controllerruntime.Result{}, nil
	}

	service, err := r.discoverLoadBalancerService(ctx, ciliumPolicy)
	if err != nil {
		logger.Error("failed to discover loadbalancer service", "error", err)
		statusErr := r.updateStatus(ctx, policy, v1.ConditionFalse, "DiscoveryFailed", err.Error())
		if statusErr != nil {
			logger.Error("failed to update status", "error", statusErr)
		}
		return controllerruntime.Result{RequeueAfter: r.SyncInterval}, nil
	}

	ips, err := r.extractLoadBalancerIPs(service)
	if err != nil {
		logger.Error("failed to extract loadbalancer ips", "error", err)
		statusErr := r.updateStatus(ctx, policy, v1.ConditionFalse, "DiscoveryFailed", err.Error())
		if statusErr != nil {
			logger.Error("failed to update status", "error", statusErr)
		}
		return controllerruntime.Result{RequeueAfter: r.SyncInterval}, nil
	}
	gatewayIP := ips[0]

	protocolPorts, err := r.extractProtocolPorts(service)
	if err != nil {
		logger.Error("failed to extract protocol and ports from service", "error", err)
		statusErr := r.updateStatus(ctx, policy, v1.ConditionFalse, "ValidationFailed", err.Error())
		if statusErr != nil {
			logger.Error("failed to update status", "error", statusErr)
		}
		return controllerruntime.Result{}, nil
	}

	routerErr := r.syncFirewallRules(ctx, policy, protocolPorts, policyIPs, gatewayIP)
	if routerErr != nil {
		logger.Error("failed to sync firewall rules", "error", routerErr)
		statusErr := r.updateStatus(ctx, policy, v1.ConditionFalse, "SyncFailed", routerErr.Error())
		if statusErr != nil {
			logger.Error("failed to update status", "error", statusErr)
		}
		return controllerruntime.Result{}, routerErr
	}

	statusErr := r.updateStatus(ctx, policy, v1.ConditionTrue, "Synced", fmt.Sprintf("router policy synced with firewall (protocols: %v, policy ips: %v, gateway ip: %s)", protocolPorts, policyIPs, gatewayIP))
	if statusErr != nil {
		logger.Error("failed to update status after successful sync", "error", statusErr)
	}

	return controllerruntime.Result{RequeueAfter: r.SyncInterval}, nil
}
