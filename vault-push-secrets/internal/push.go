package internal

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/goccy/go-yaml"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type Opts struct {
	Continuous            *bool
	BitwardenAccessToken  string
	BitwardenSecretID     string
	Interval              time.Duration
	VaultAddr             string
	VaultAuthMount        string
	VaultAuthRole         string
	VaultAuthToken        string
	VaultSecretsMount     string
}

type Pusher struct {
	Continuous            bool
	Interval              time.Duration
	LastChecksum          string
	BitwardenAccessToken  string
	BitwardenSecretID     string
	Vault                 *vault.Client
	VaultAddr             string
	VaultAuthMount        string
	VaultAuthRole         string
	VaultAuthToken        string
	VaultSecretsMount     string
}

type VaultRole struct {
	Namespace      string `json:"namespace,omitempty"`
	Secret         string `json:"secret,omitempty"`
	ServiceAccount string `json:"service-account"`
	Roles          bool   `json:"roles,omitempty"`
	Policies       bool   `json:"policies,omitempty"`
}

type VaultSecrets struct {
	Secrets map[string]any       `json:"secrets"`
	Roles   map[string]VaultRole `json:"roles"`
}

func New(opts *Opts) (*Pusher, error) {
	if opts.VaultAddr == "" {
		return nil, fmt.Errorf("vault addr unset")
	}

	if opts.VaultAuthMount == "" {
		return nil, fmt.Errorf("vault auth mount path unset")
	}

	interval := opts.Interval
	if interval == 0 {
		interval = 10 * time.Minute
	}

	if opts.VaultAuthToken == "" && opts.VaultAuthRole == "" {
		return nil, fmt.Errorf("auth role unset")
	}

	continuous := true
	if opts.Continuous != nil {
		continuous = *opts.Continuous
	}

	if opts.VaultSecretsMount == "" {
		return nil, fmt.Errorf("secrets mount unset")
	}

	if opts.BitwardenAccessToken == "" {
		return nil, fmt.Errorf("bitwarden access token unset")
	}

	if opts.BitwardenSecretID == "" {
		return nil, fmt.Errorf("bitwarden secret id unset")
	}

	// Verify bws CLI is available
	if _, err := exec.LookPath("bws"); err != nil {
		return nil, fmt.Errorf("bws CLI not found: %w", err)
	}

	vaultClient, err := vault.New(
		vault.WithAddress(opts.VaultAddr),
	)
	if err != nil {
		return nil, err
	}

	pusher := Pusher{
		Continuous:            continuous,
		Interval:              interval,
		BitwardenAccessToken:  opts.BitwardenAccessToken,
		BitwardenSecretID:     opts.BitwardenSecretID,
		Vault:                 vaultClient,
		VaultAddr:             opts.VaultAddr,
		VaultAuthRole:         opts.VaultAuthRole,
		VaultAuthMount:        opts.VaultAuthMount,
		VaultAuthToken:        opts.VaultAuthToken,
		VaultSecretsMount:     opts.VaultSecretsMount,
	}
	return &pusher, nil
}


func (p *Pusher) toStringSlice(v any) ([]string, error) {
	anyValues, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("value not []any")
	}
	strValues := make([]string, len(anyValues))
	for index, anyValue := range anyValues {
		strValue, ok := anyValue.(string)
		if !ok {
			return nil, fmt.Errorf("item at %d is not string", index)
		}
		strValues[index] = strValue
	}
	return strValues, nil
}

func (p *Pusher) requireStringSlice(v any, min int, max int) ([]string, error) {
	strArr, err := p.toStringSlice(v)
	if err != nil {
		return nil, err
	}

	if min > len(strArr) {
		return nil, fmt.Errorf("not enough array items: %d > %d", min, len(strArr))
	}
	if max < len(strArr) {
		return nil, fmt.Errorf("too many array items: %d < %d", max, len(strArr))
	}

	return strArr, nil
}

func (p *Pusher) getPolicySecret(policy string) (string, error) {
	policySecretsMount := strings.Trim(p.VaultSecretsMount, "/")
	pattern := fmt.Sprintf("path \"%s/data/([^\"]+)\"", policySecretsMount)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	matches := re.FindStringSubmatch(policy)
	if matches != nil {
		return matches[1], nil
	}

	pattern = fmt.Sprintf("path \"%s/\\*\"", policySecretsMount)
	re, err = regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	matches = re.FindStringSubmatch(policy)
	if matches != nil {
		return "", nil
	}

	return "", fmt.Errorf("secret not found")
}

func (p *Pusher) canReadRoles(policy string) (bool, error) {
	pattern := "path \"auth/kubernetes/role\""
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	matches := re.FindStringSubmatch(policy)
	if matches != nil {
		return true, nil
	}
	return false, nil
}

func (p *Pusher) canReadPolicies(policy string) (bool, error) {
	pattern := "path \"sys/policies/acl/\\*\""
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	matches := re.FindStringSubmatch(policy)
	if matches != nil {
		return true, nil
	}
	return false, nil
}
func (p *Pusher) ExportSecrets(ctx context.Context) (*VaultSecrets, error) {
	logger := logging.FromContext(ctx)

	response, err := p.Vault.Secrets.KvV2List(ctx, "", vault.WithMountPath(p.VaultSecretsMount))
	if err != nil {
		logger.Error("failed to list secrets from vault", "secrets-path", p.VaultSecretsMount, "error", err)
		return nil, err
	}
	apps := response.Data.Keys

	secrets := map[string]any{}
	for _, app := range apps {
		response, err := p.Vault.Secrets.KvV2Read(ctx, app, vault.WithMountPath(p.VaultSecretsMount))
		if err != nil && !vault.IsErrorStatus(err, 404) {
			logger.Error("failed to read secret", "app", app, "error", err)
			return nil, err
		}
		data := map[string]any{}
		if err == nil {
			data = response.Data.Data
		}
		secrets[app] = data
	}

	response, err = p.Vault.Auth.KubernetesListAuthRoles(ctx, vault.WithMountPath(p.VaultAuthMount))
	if err != nil {
		logger.Error("failed to list roles from vault", "kubernetes-path", p.VaultAuthMount, "error", err)
		return nil, err
	}
	roles := map[string]VaultRole{}
	roleNames := response.Data.Keys
	for _, roleName := range roleNames {
		response, err := p.Vault.Auth.KubernetesReadAuthRole(ctx, roleName, vault.WithMountPath(p.VaultAuthMount))
		if err != nil {
			logger.Error("failed to read role", "role", roleName, "error", err)
			return nil, err
		}

		serviceAccounts, err := p.requireStringSlice(response.Data["bound_service_account_names"], 1, 1)
		if err != nil {
			logger.Error("failed to read role service account", "role", roleName, "error", err)
			return nil, err
		}
		serviceAccount := serviceAccounts[0]

		namespaces, err := p.requireStringSlice(response.Data["bound_service_account_namespaces"], 0, 1)
		if err != nil {
			logger.Error("failed to read role namespaces", "role", roleName, "error", err)
			return nil, err
		}
		namespace := ""
		if len(namespaces) == 1 {
			namespace = namespaces[0]
		}

		policyNames, err := p.requireStringSlice(response.Data["token_policies"], 1, 1)
		if err != nil {
			logger.Error("failed to read role policies", "role", roleName, "error", err)
			return nil, err
		}
		policyName := policyNames[0]

		policyResponse, err := p.Vault.System.PoliciesReadAclPolicy(ctx, policyName)
		if err != nil {
			logger.Error("failed to read policy", "policy", policyName, "error", err)
			return nil, err
		}
		policy := policyResponse.Data.Policy

		secret, err := p.getPolicySecret(policy)
		if err != nil {
			logger.Error("failed to get policy secret", "policy", policyName, "error", err)
			return nil, err
		}

		canReadRoles, err := p.canReadRoles(policy)
		if err != nil {
			logger.Error("failed to determine whether policy can read roles", "policy", policyName, "error", err)
			return nil, err
		}

		canReadPolicies, err := p.canReadPolicies(policy)
		if err != nil {
			logger.Error("failed to determine whether policy can read policies", "policy", policyName, "error", err)
			return nil, err
		}

		role := VaultRole{
			Namespace:      namespace,
			Secret:         secret,
			ServiceAccount: serviceAccount,
			Roles:          canReadRoles,
			Policies:       canReadPolicies,
		}
		roles[roleName] = role
	}

	data := VaultSecrets{
		Secrets: secrets,
		Roles:   roles,
	}
	return &data, nil
}

func (p *Pusher) Checksum(ctx context.Context, data *VaultSecrets) (string, error) {
	logger := logging.FromContext(ctx)

	dataBytes, err := json.Marshal(data)
	if err != nil {
		logger.Error("failed to marshal secrets for checksum calculation", "error", err)
		return "", err
	}

	hash := sha256.Sum256(dataBytes)
	checksum := fmt.Sprintf("%x", hash)

	return checksum, nil
}

func (p *Pusher) Upload(ctx context.Context, data *VaultSecrets) error {
	logger := logging.FromContext(ctx)

	dataBytes, err := yaml.Marshal(data)
	if err != nil {
		logger.Error("failed to marshal secrets to YAML", "error", err)
		return err
	}

	dataStr := string(dataBytes)

	// Get the secret metadata using bws CLI
	cmd := exec.CommandContext(ctx, "bws", "secret", "get", p.BitwardenSecretID, "--output", "json", "--access-token", p.BitwardenAccessToken)
	output, err := cmd.Output()
	if err != nil {
		logger.Error("failed to get secret from bitwarden", "secret_id", p.BitwardenSecretID, "error", err)
		return err
	}

	var secretData map[string]interface{}
	if err := json.Unmarshal(output, &secretData); err != nil {
		logger.Error("failed to parse secret data from bitwarden", "error", err)
		return err
	}

	// Update the secret using bws CLI
	cmd = exec.CommandContext(ctx, "bws", "secret", "update", p.BitwardenSecretID, "--value", dataStr, "--access-token", p.BitwardenAccessToken)
	if err := cmd.Run(); err != nil {
		logger.Error("failed to update secret in bitwarden", "secret_id", p.BitwardenSecretID, "error", err)
		return err
	}

	return nil
}

func (p *Pusher) AuthVault(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	authToken := p.VaultAuthToken
	if authToken == "" {
		authTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
		jwtBytes, err := os.ReadFile(authTokenPath)
		if err != nil {
			logger.Error("failed to read service account token", "path", authTokenPath, "error", err)
			return err
		}
		jwt := string(jwtBytes)

		response, err := p.Vault.Auth.KubernetesLogin(ctx, schema.KubernetesLoginRequest{
			Jwt:  jwt,
			Role: p.VaultAuthRole,
		})
		if err != nil {
			logger.Error("failed to authenticate with vault using kubernetes", "auth-role", p.VaultAuthRole, "error", err)
			return err
		}

		authToken = response.Auth.ClientToken
	}

	err := p.Vault.SetToken(authToken)
	if err != nil {
		logger.Error("failed to set vault client token", "error", err)
		return err
	}

	return nil
}

func (p *Pusher) Push(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Debug("authenticating with vault")
	err := p.AuthVault(ctx)
	if err != nil {
		logger.Error("vault authentication failed", "error", err)
		return err
	}
	defer p.Vault.ClearToken()

	logger.Debug("exporting secrets")
	secrets, err := p.ExportSecrets(ctx)
	if err != nil {
		logger.Error("failed to export secrets", "error", err)
		return err
	}

	logger.Debug("calculating checksum")
	checksum, err := p.Checksum(ctx, secrets)
	if err != nil {
		logger.Error("failed to calculate checksum", "error", err)
		return err
	}

	if checksum == p.LastChecksum {
		logger.Info("secrets unchanged, skipping upload")
		return nil
	}
	logger.Debug("secrets changed, uploading", "previous-checksum", p.LastChecksum, "current-checksum", checksum)
	p.LastChecksum = checksum

	err = p.Upload(ctx, secrets)
	if err != nil {
		logger.Error("failed to upload secrets", "error", err)
		return err
	}

	logger.Info("secrets successfully pushed", "checksum", checksum)
	return nil
}

func (p *Pusher) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	logger.Info("starting vault push", "vault", p.VaultAddr)

	err := p.Push(ctx)
	if err != nil {
		logger.Error("initial push failed", "error", err)
		return err
	}

	if !p.Continuous {
		return nil
	}

	logger.Info("entering continuous push loop", "interval", p.Interval)

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	running := true
	pushCount := 0
	for running {
		select {
		case <-ticker.C:
			pushCount++
			logger.Debug("executing scheduled push", "push-number", pushCount)
			err := p.Push(ctx)
			if err != nil {
				logger.Error("push failed", "push-number", pushCount, "error", err)
				return err
			}
		case sig := <-signalChannel:
			logger.Info("shutdown signal received", "signal", sig)
			running = false
		}
	}

	logger.Info("vault push shutdown complete")
	return nil
}
