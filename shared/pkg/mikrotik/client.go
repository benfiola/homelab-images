package mikrotik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

type Opts struct {
	BaseURL  string
	Password string
	Username string
}

type Client struct {
	BaseURL  string
	Client   *http.Client
	Username string
	Password string
}

func New(opts *Opts) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base url unset")
	}

	if opts.Password == "" {
		return nil, fmt.Errorf("password unset")
	}

	if opts.Username == "" {
		return nil, fmt.Errorf("username unset")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
			MaxConnsPerHost: 5,
		},
	}

	client := Client{
		BaseURL:  opts.BaseURL,
		Client:   httpClient,
		Password: opts.Password,
		Username: opts.Username,
	}

	return &client, nil
}

type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, string(e.Body))
}

func (c *Client) doRequest(ctx context.Context, method, url string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.SetBasicAuth(c.Username, c.Password)

	response, err := c.Client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, &HTTPError{Body: responseBody, StatusCode: response.StatusCode}
	}

	return responseBody, nil
}

func (c *Client) ignoreStatusCode(err error, statusCodes ...int) error {
	httpErr, ok := err.(*HTTPError)
	if !ok {
		return err
	}
	if slices.Contains(statusCodes, httpErr.StatusCode) {
		return nil
	}
	return err
}

func (c *Client) ignoreNotFound(err error) error {
	return c.ignoreStatusCode(err, 404)
}

type FirewallFilter struct {
	ID             string `json:".id,omitempty"`
	Action         string `json:"action,omitempty"`
	Chain          string `json:"chain,omitempty"`
	Comment        string `json:"comment,omitempty"`
	Disabled       string `json:"disabled,omitempty"`
	DstPort        string `json:"dst-port,omitempty"`
	PlaceBefore    string `json:"place-before,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	SrcAddressList string `json:"src-address-list,omitempty"`
}

func (c *Client) ListFirewallFilters(ctx context.Context) ([]*FirewallFilter, error) {
	url := fmt.Sprintf("%s/%s", c.BaseURL, "rest/ip/firewall/filter")

	body, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	filters := []*FirewallFilter{}
	err = json.Unmarshal(body, &filters)
	if err != nil {
		return nil, err
	}

	return filters, nil
}

func (c *Client) DeleteFirewallFilter(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, "rest/ip/firewall/filter", id)

	_, err := c.doRequest(ctx, "DELETE", url, nil)
	if c.ignoreNotFound(err) != nil {
		return err
	}

	return nil
}

func (c *Client) InsertFirewallFilter(ctx context.Context, filter *FirewallFilter) error {
	url := fmt.Sprintf("%s/%s", c.BaseURL, "rest/ip/firewall/filter")

	body, err := c.doRequest(ctx, "PUT", url, filter)
	if err != nil {
		return err
	}

	created := &FirewallFilter{}
	err = json.Unmarshal(body, created)
	if err != nil {
		return err
	}

	filter.ID = created.ID

	return nil
}

type FirewallAddressList struct {
	ID      string `json:".id,omitempty"`
	List    string `json:"list"`
	Address string `json:"address"`
}

func (c *Client) ListFirewallAddressLists(ctx context.Context) ([]*FirewallAddressList, error) {
	url := fmt.Sprintf("%s/%s", c.BaseURL, "rest/ip/firewall/address-list")

	body, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	addressLists := []*FirewallAddressList{}
	err = json.Unmarshal(body, &addressLists)
	if err != nil {
		return nil, err
	}

	return addressLists, nil
}

func (c *Client) DeleteFirewallAddressList(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, "rest/ip/firewall/address-list", id)

	_, err := c.doRequest(ctx, "DELETE", url, nil)
	if c.ignoreNotFound(err) != nil {
		return err
	}

	return nil
}

func (c *Client) CreateFirewallAddressList(ctx context.Context, addressList *FirewallAddressList) error {
	url := fmt.Sprintf("%s/%s", c.BaseURL, "rest/ip/firewall/address-list")

	body, err := c.doRequest(ctx, "PUT", url, addressList)
	if err != nil {
		return err
	}

	created := &FirewallAddressList{}
	err = json.Unmarshal(body, created)
	if err != nil {
		return err
	}

	addressList.ID = created.ID

	return nil
}

type FirewallNat struct {
	ID             string `json:".id,omitempty"`
	Action         string `json:"action,omitempty"`
	Chain          string `json:"chain,omitempty"`
	Comment        string `json:"comment,omitempty"`
	Disabled       string `json:"disabled,omitempty"`
	DstPort        string `json:"dst-port,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	SrcAddressList string `json:"src-address-list,omitempty"`
	ToAddresses    string `json:"to-addresses,omitempty"`
	ToPort         string `json:"to-ports,omitempty"`
}

func (c *Client) ListFirewallNats(ctx context.Context) ([]*FirewallNat, error) {
	url := fmt.Sprintf("%s/%s", c.BaseURL, "rest/ip/firewall/nat")

	body, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	natsRules := []*FirewallNat{}
	err = json.Unmarshal(body, &natsRules)
	if err != nil {
		return nil, err
	}

	return natsRules, nil
}

func (c *Client) DeleteFirewallNat(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, "rest/ip/firewall/nat", id)

	_, err := c.doRequest(ctx, "DELETE", url, nil)
	if c.ignoreNotFound(err) != nil {
		return err
	}

	return nil
}

func (c *Client) InsertFirewallNat(ctx context.Context, nat *FirewallNat) error {
	url := fmt.Sprintf("%s/%s", c.BaseURL, "rest/ip/firewall/nat")

	body, err := c.doRequest(ctx, "PUT", url, nat)
	if err != nil {
		return err
	}

	created := &FirewallNat{}
	err = json.Unmarshal(body, created)
	if err != nil {
		return err
	}

	nat.ID = created.ID

	return nil
}

func (c *Client) UpdateFirewallNat(ctx context.Context, nat *FirewallNat) error {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, "rest/ip/firewall/nat", nat.ID)

	_, err := c.doRequest(ctx, "PATCH", url, nat)
	if err != nil {
		return err
	}

	return nil
}
