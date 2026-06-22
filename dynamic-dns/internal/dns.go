package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/cloudflare/cloudflare-go/v6/option"
	"github.com/cloudflare/cloudflare-go/v6/zones"
)

type Zone struct {
	Name    string   `json:"name"`
	Domains []string `json:"domains"`
}

type Opts struct {
	CloudflareAPIToken string
	CloudflareZones    []Zone
	Continuous         *bool
	Interval           time.Duration
}

type Client struct {
	Cloudflare      *cloudflare.Client
	CloudflareZones []Zone
	Continuous      bool
	Interval        time.Duration
}

func New(opts *Opts) (*Client, error) {
	if opts.CloudflareAPIToken == "" {
		return nil, fmt.Errorf("cloudflare api token unset")
	}

	cf := cloudflare.NewClient(option.WithAPIToken(opts.CloudflareAPIToken))

	continuous := true
	if opts.Continuous != nil {
		continuous = *opts.Continuous
	}

	interval := opts.Interval
	if interval == (time.Duration(0)) {
		interval = 10 * time.Minute
	}

	client := &Client{
		Cloudflare:      cf,
		Continuous:      continuous,
		Interval:        interval,
		CloudflareZones: opts.CloudflareZones,
	}

	return client, nil
}

func (c *Client) getPublicIp(pctx context.Context) (string, error) {
	url := "https://api.ipify.org"

	ctx, cancel := context.WithTimeout(pctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyStr := strings.TrimSpace(string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, bodyStr)
	}

	return bodyStr, nil
}

func (c *Client) findZone(ctx context.Context, zone string) (*zones.Zone, error) {
	listResponse, err := c.Cloudflare.Zones.List(ctx, zones.ZoneListParams{Name: cloudflare.F(zone)})
	if err != nil {
		return nil, err
	}
	if len(listResponse.Result) == 0 {
		return nil, fmt.Errorf("no zones with name %s", zone)
	}

	cfZone := listResponse.Result[0]
	return &cfZone, nil
}

func (c *Client) findDnsRecord(ctx context.Context, zoneID string, domain string) (*dns.RecordResponse, error) {
	recordList, err := c.Cloudflare.DNS.Records.List(ctx, dns.RecordListParams{Name: cloudflare.F(dns.RecordListParamsName{Exact: cloudflare.F(domain)}), ZoneID: cloudflare.F(zoneID)})
	if err != nil {
		return nil, err
	}
	var found *dns.RecordResponse
	for _, record := range recordList.Result {
		if record.Type == dns.RecordResponseTypeA {
			found = &record
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("no dns A records with name %s", domain)
	}
	return found, nil
}

func (c *Client) updateDnsRecord(ctx context.Context, zoneID string, recordID string, ip string) error {
	_, err := c.Cloudflare.DNS.Records.Edit(ctx, recordID, dns.RecordEditParams{Body: dns.ARecordParam{Content: cloudflare.F(ip)}, ZoneID: cloudflare.F(zoneID)})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Sync(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	ip, err := c.getPublicIp(ctx)
	if err != nil {
		logger.Error("failed to get public ip address", "error", err)
		return err
	}

	var lastError error
	for _, zone := range c.CloudflareZones {
		cfZone, err := c.findZone(ctx, zone.Name)
		if err != nil {
			logger.Error("failed to find zone", "name", zone.Name, "error", err)
			lastError = err
			continue
		}
		for _, domain := range zone.Domains {
			cfRecord, err := c.findDnsRecord(ctx, cfZone.ID, domain)
			if err != nil {
				logger.Error("failed to find dns record", "zone", zone.Name, "domain", domain, "error", err)
				lastError = err
				continue
			}

			if cfRecord.Content == ip {
				logger.Debug("ip address has not changed", "zone", zone.Name, "domain", domain, "ip", ip)
				continue
			}

			err = c.updateDnsRecord(ctx, cfZone.ID, cfRecord.ID, ip)
			if err != nil {
				logger.Error("failed to update dns record", "zone", zone.Name, "domain", domain, "error", err)
				lastError = err
				continue
			}
		}
	}

	return lastError
}

func (c *Client) Run(pctx context.Context) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	logger := logging.FromContext(ctx)
	logger.Info("starting dynamic dns client")

	err := c.Sync(ctx)
	if err != nil {
		logger.Error("initial sync failed", "error", err)
		return err
	}

	if !c.Continuous {
		return nil
	}

	logger.Info("entering continuous loop", "interval", c.Interval)

	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)

	go func() {
		<-signalChannel
		logger.Info("shutdown signal received")
		cancel()
	}()

	running := true
	syncCount := 0
	for running {
		select {
		case <-ticker.C:
			syncCount++
			logger.Debug("executing scheduled sync", "sync-number", syncCount)
			err := c.Sync(ctx)
			if err != nil {
				logger.Error("sync failed", "sync-number", syncCount, "error", err)
				return err
			}
		case <-ctx.Done():
			logger.Info("dynamic dns shutdown")
			running = false
		}
	}

	logger.Info("dynamic dns client shutdown complete")
	return nil
}
