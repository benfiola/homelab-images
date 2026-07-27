package internal

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// soapClient is a minimal client for AzerothCore worldserver's SOAP
// interface - just enough to call its one operation, executeCommand, which
// runs the given string as if it were typed into the in-game/console GM
// chat handler (e.g. ".account delete someuser"). Deliberately hand-rolled
// against the fixed envelope shape AzerothCore's gSOAP server emits/expects,
// rather than pulling in a general-purpose SOAP library for a single
// operation.
type soapClient struct {
	addr     string
	username string
	password string
	http     *http.Client
}

func newSOAPClient(addr, username, password string) *soapClient {
	return &soapClient{
		addr:     addr,
		username: username,
		password: password,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

const soapRequestTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/1999/XMLSchema-instance" xmlns:xsd="http://www.w3.org/1999/XMLSchema">
  <SOAP-ENV:Body>
    <ns1:executeCommand xmlns:ns1="urn:AC">
      <command>%s</command>
    </ns1:executeCommand>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

type soapResponseEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Fault *struct {
			FaultCode   string `xml:"faultcode"`
			FaultString string `xml:"faultstring"`
		} `xml:"Fault"`
		ExecuteCommandResponse *struct {
			Result string `xml:"result"`
		} `xml:"executeCommandResponse"`
	} `xml:"Body"`
}

// ExecuteCommand runs command as an AzerothCore GM console command over
// SOAP, authenticated as the service account, and returns the command's
// text output.
func (c *soapClient) ExecuteCommand(ctx context.Context, command string) (string, error) {
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(command)); err != nil {
		return "", fmt.Errorf("escape command: %w", err)
	}
	reqBody := fmt.Sprintf(soapRequestTemplate, escaped.String())

	url := fmt.Sprintf("http://%s/", c.addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(reqBody)))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("soap request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", fmt.Errorf("soap: service account authentication rejected (401)")
	case http.StatusForbidden:
		return "", fmt.Errorf("soap: service account GM level too low to open a SOAP session (403)")
	}

	var env soapResponseEnvelope
	if err := xml.Unmarshal(body, &env); err != nil {
		return "", fmt.Errorf("parse soap response (status %d): %w", resp.StatusCode, err)
	}
	if env.Body.Fault != nil {
		return "", fmt.Errorf("soap command failed: %s", env.Body.Fault.FaultString)
	}
	if env.Body.ExecuteCommandResponse == nil {
		return "", fmt.Errorf("soap: unexpected response shape (status %d)", resp.StatusCode)
	}

	return env.Body.ExecuteCommandResponse.Result, nil
}

// SetPassword sets username's password via AzerothCore's own
// ".account set password" GM command - used both for admin-triggered resets
// of any account, and for backend-mediated self-service changes (after the
// caller has independently verified the user's old password).
func (c *soapClient) SetPassword(ctx context.Context, username, newPassword string) error {
	_, err := c.ExecuteCommand(ctx, fmt.Sprintf(".account set password %s %s %s", username, newPassword, newPassword))
	return err
}

// DeleteAccount deletes username via AzerothCore's own ".account delete" GM
// command, which cascades to the account's characters and related rows.
func (c *soapClient) DeleteAccount(ctx context.Context, username string) error {
	_, err := c.ExecuteCommand(ctx, fmt.Sprintf(".account delete %s", username))
	return err
}
