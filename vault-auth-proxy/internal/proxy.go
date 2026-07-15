package internal

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

type Opts struct {
	ListenAddr    string
	VaultAddr     string
	RootTokenPath string
}

type AuthProxy struct {
	ListenAddr    string
	VaultAddr     string
	RootTokenPath string
	Client        *http.Client
}

func New(opts *Opts) (*AuthProxy, error) {
	if opts.ListenAddr == "" {
		opts.ListenAddr = "127.0.0.1:8201"
	}
	if opts.VaultAddr == "" {
		opts.VaultAddr = "http://localhost:8200"
	}
	if opts.RootTokenPath == "" {
		opts.RootTokenPath = "/vault/data/root-token"
	}

	proxy := &AuthProxy{
		ListenAddr:    opts.ListenAddr,
		VaultAddr:     opts.VaultAddr,
		RootTokenPath: opts.RootTokenPath,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return proxy, nil
}

func (p *AuthProxy) getRootToken(ctx context.Context) (string, error) {
	logger := logging.FromContext(ctx)

	tokenBytes, err := os.ReadFile(p.RootTokenPath)
	if err != nil {
		logger.Error("failed to read root token file", "path", p.RootTokenPath, "error", err)
		return "", err
	}

	return string(tokenBytes), nil
}


func (p *AuthProxy) handler(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	token, err := p.getRootToken(r.Context())
	if err != nil {
		logger.Warn("root token not available", "error", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetURL, err := url.Parse(p.VaultAddr)
	if err != nil {
		logger.Error("invalid vault address", "addr", p.VaultAddr, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		logger.Error("failed to create proxy request", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("X-Vault-Token", token)

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	resp, err := p.Client.Do(proxyReq)
	if err != nil {
		logger.Error("failed to proxy request", "error", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		logger.Error("failed to write response", "error", err)
	}
}

func (p *AuthProxy) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	logger.Info("starting vault auth proxy", "listen_addr", p.ListenAddr, "vault_addr", p.VaultAddr, "root_token_path", p.RootTokenPath)

	server := &http.Server{
		Addr:    p.ListenAddr,
		Handler: http.HandlerFunc(p.handler),
	}

	errChan := make(chan error, 1)
	go func() {
		logger.Info("listening for requests", "addr", p.ListenAddr)
		errChan <- server.ListenAndServe()
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			return err
		}
	case sig := <-signalChannel:
		logger.Info("received signal, shutting down", "signal", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown server", "error", err)
			return err
		}
	}

	return nil
}
