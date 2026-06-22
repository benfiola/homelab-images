package internal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type Opts struct {
	Continuous     *bool
	VaultAddr      string
	VaultUnsealKey string
}

type Unsealer struct {
	Continuous     bool
	Vault          *vault.Client
	VaultAddr      string
	VaultUnsealKey string
}

func New(opts *Opts) (*Unsealer, error) {
	vaultAddr := opts.VaultAddr
	if vaultAddr == "" {
		vaultAddr = "http://localhost:8200"
	}

	continuous := true
	if opts.Continuous != nil {
		continuous = *opts.Continuous
	}

	if opts.VaultUnsealKey == "" {
		return nil, fmt.Errorf("vault unseal key unset")
	}

	vaultClient, err := vault.New(
		vault.WithAddress(vaultAddr),
	)
	if err != nil {
		return nil, err
	}

	unsealer := Unsealer{
		Continuous:     continuous,
		Vault:          vaultClient,
		VaultAddr:      vaultAddr,
		VaultUnsealKey: opts.VaultUnsealKey,
	}
	return &unsealer, nil
}

func (u *Unsealer) WaitForPath(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	for {
		_, err := os.Lstat(u.VaultUnsealKey)
		if err == nil {
			return nil
		}
		logger.Debug("waiting for vault unseal key", "path", u.VaultUnsealKey)
		time.Sleep(1 * time.Second)
	}
}

func (u *Unsealer) WaitForVault(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	for {
		_, err := u.Vault.System.SealStatus(ctx)
		if err == nil {
			return nil
		}
		logger.Debug("vault not ready, retrying", "address", u.VaultAddr)
		time.Sleep(1 * time.Second)
	}
}

func (u *Unsealer) Unseal(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Debug("waiting for unseal key file")
	err := u.WaitForPath(ctx)
	if err != nil {
		logger.Error("failed while waiting for unseal key file", "error", err)
		return err
	}

	logger.Debug("waiting for vault to be reachable")
	err = u.WaitForVault(ctx)
	if err != nil {
		logger.Error("failed while waiting for vault", "error", err)
		return err
	}

	logger.Debug("checking vault seal status")
	response, err := u.Vault.System.SealStatus(ctx)
	if err != nil {
		logger.Error("failed to check vault seal status", "error", err)
		return err
	}
	if !response.Data.Sealed {
		logger.Info("vault already unsealed")
		return nil
	}

	logger.Debug("reading unseal key")
	unsealKeyBytes, err := os.ReadFile(u.VaultUnsealKey)
	if err != nil {
		logger.Error("failed to read vault unseal key", "path", u.VaultUnsealKey, "error", err)
		return err
	}
	unsealKey := string(unsealKeyBytes)

	logger.Debug("sending unseal request to vault")
	_, err = u.Vault.System.Unseal(ctx, schema.UnsealRequest{Key: unsealKey})
	if err != nil {
		logger.Error("failed to unseal vault", "error", err)
		return err
	}

	logger.Info("vault unsealed successfully")
	return nil
}

func (u *Unsealer) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	logger.Info("starting vault unseal process", "vault", u.VaultAddr)

	err := u.Unseal(ctx)
	if err != nil {
		logger.Error("unseal process failed", "error", err)
		return err
	}

	if !u.Continuous {
		return nil
	}

	logger.Info("unseal successful, waiting for shutdown signal")
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT)
	sig := <-signalChannel
	logger.Info("received signal, shutting down", "signal", sig)

	return nil
}
