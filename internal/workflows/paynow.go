package workflows

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Provider represents supported mobile wallet providers for Express Checkout.
// Valid values: Ecocash, Onemoney, InnBucks, Omari
// Paynow API typically expects specific method strings.
type Provider string

const (
	ProviderEcocash  Provider = "ECOCASH"
	ProviderOnemoney Provider = "ONEMONEY"
	ProviderInnBucks Provider = "INNBUCKS"
	ProviderOmari    Provider = "OMARI"
)

// PaynowClient holds credentials and HTTP client configuration.
type PaynowClient struct {
	IntegrationID  string
	IntegrationKey string
	baseURL        string
	HTTPClient     *http.Client // optional
}

// NewClient creates a Paynow client with sane defaults.
func NewClient(integrationID, integrationKey string) *PaynowClient {
	return &PaynowClient{
		IntegrationID:  integrationID,
		IntegrationKey: integrationKey,
		baseURL:        "https://www.paynow.co.zw/interface",
		HTTPClient:     &http.Client{Timeout: 25 * time.Second},
	}
}

// ExpressCheckoutRequest is the payload for creating an Express Checkout transaction.
type ExpressCheckoutRequest struct {
	Id             string   `json:"id"`
	Reference      string   `json:"reference"`
	AuthEmail      string   `json:"email,omitempty"`
	Amount         float64  `json:"amount"`
	Method         Provider `json:"method"`
	Phone          string   `json:"phone"`
	ReturnURL      string   `json:"returnurl,omitempty"`
	ResultURL      string   `json:"resulturl,omitempty"`
	AdditionalInfo string   `json:"additionalinfo,omitempty"`
	MerchantTrace  string   `json:"merchanttrace,omitempty"`
	Status         string   `json:"status,omitempty"`
	Hash           string   `json:"hash,omitempty"`
}
type MerchantTraceRequest struct {
	Id            string `json:"id"`
	MerchantTrace string `json:"merchanttrace,omitempty"`
	Status        string `json:"status,omitempty"`
	Hash          string `json:"hash,omitempty"`
}

// ExpressCheckoutResponse represents the most relevant parts of the Paynow response.
type ExpressCheckoutResponse struct {
	Status          string `json:"status"`
	Instructions    string `json:"instructions,omitempty"`
	PayNowReference string `json:"paynowreference,omitempty"`
	PollURL         string `json:"pollurl,omitempty"`
	Hash            string `json:"hash,omitempty"`
	OtpReference    string `json:"otpreference,omitempty"`
	RemoteOtpUrl    string `json:"remoteotpurl,omitempty"`
}
type CheckPaymentResponse struct {
	Reference       string  `json:"reference"`
	PayNowReference string  `json:"paynowreference,omitempty"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"`
	PollURL         string  `json:"pollurl,omitempty"`
	Hash            string  `json:"hash,omitempty"`
}

func computePaynowHash(req ExpressCheckoutRequest, integrationKey string) string {
	// Map of fields (without hash)
	//note because the form is posted alphabetic we do an order by on these
	fields := map[string]string{
		"additionalinfo": req.AdditionalInfo,
		"amount":         fmt.Sprintf("%.2f", req.Amount),
		"authemail":      req.AuthEmail,
		"id":             req.Id,
		"method":         string(req.Method),
		"merchanttrace":  req.MerchantTrace,
		"phone":          req.Phone,
		"reference":      req.Reference,
		"resulturl":      req.ResultURL,
		"returnurl":      req.ReturnURL,
		"status":         req.Status,
	}

	return createHashFromFields(fields, integrationKey)
}
func computePaynowHashMerchantTrace(req MerchantTraceRequest, integrationKey string) string {
	// Map of fields (without hash)
	//note because the form is posted alphabetic we do an order by on these
	fields := map[string]string{
		"id":            req.Id,
		"merchanttrace": req.MerchantTrace,
		"status":        req.Status,
	}

	return createHashFromFields(fields, integrationKey)
}

func createHashFromFields(fields map[string]string, integrationKey string) string {
	// Sort keys alphabetically
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build concatenated string of values
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fields[k])
	}

	toHash := strings.Join(parts, "") + integrationKey
	//fmt.Println(toHash)
	sum := sha512.Sum512([]byte(toHash))
	//fmt.Println(strings.ToUpper(hex.EncodeToString(sum[:])))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func buildPaynowFields(req ExpressCheckoutRequest) url.Values {
	form := url.Values{}
	form.Set("reference", req.Reference)
	form.Set("authemail", req.AuthEmail)
	form.Set("amount", fmt.Sprintf("%.2f", req.Amount))
	form.Set("method", string(req.Method))
	form.Set("merchanttrace", req.MerchantTrace)
	form.Set("phone", req.Phone)
	form.Set("returnurl", req.ReturnURL)
	form.Set("id", req.Id)
	form.Set("resulturl", req.ResultURL)
	form.Set("additionalinfo", req.AdditionalInfo)
	form.Set("status", req.Status)
	form.Set("hash", req.Hash)
	return form
}
func buildPaynowMerchantTraceFields(req MerchantTraceRequest) url.Values {
	form := url.Values{}
	form.Set("id", req.Id)
	form.Set("merchanttrace", req.MerchantTrace)
	form.Set("status", req.Status)
	form.Set("hash", req.Hash)
	return form
}

// CreateExpressCheckout creates an Express Checkout transaction for the given mobile wallet provider.
// It returns a response containing poll URL/redirect URL and status. The function is generic to support
// EcoCash, OneMoney, InnBucks, and O'mari (Omari).
func (c *PaynowClient) CreateExpressCheckout(ctx context.Context, req ExpressCheckoutRequest) (*ExpressCheckoutResponse, error) {
	if c == nil {
		return nil, errors.New("nil client")
	}
	if c.IntegrationID == "" || c.IntegrationKey == "" {
		return nil, errors.New("missing integration credentials")
	}

	if req.Reference == "" {
		return nil, errors.New("reference is required")
	}
	if req.AuthEmail == "" {
		return nil, errors.New("AuthEmail is required")
	}
	if req.Amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}
	if req.Method == "" {
		return nil, errors.New("Method is required")
	}
	if req.Phone == "" {
		return nil, errors.New("phone is required")
	}
	//this is just the integration ID
	req.Id = c.IntegrationID

	if req.AdditionalInfo == "" {
		return nil, errors.New("AdditionalInfo is required")
	}
	if req.Status == "" {
		return nil, errors.New("Status is required")
	}
	if req.MerchantTrace == "" {
		return nil, errors.New("MerchantTrace is required")
	}
	//check that merchantTrace is 32 or less chars
	if len(req.MerchantTrace) > 32 {
		return nil, errors.New("MerchantTrace must be 32 or less characters")
	}

	fullPath := fmt.Sprintf("%s/remotetransaction", c.baseURL)

	req.Hash = computePaynowHash(req, c.IntegrationKey)

	form := buildPaynowFields(req)

	output := strings.NewReader(form.Encode())

	fmt.Println(output)

	// Encode to application/x-www-form-urlencoded
	//encoded := form.Encode() // e.g. "Status=Ok&Phone=0771234567..."

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullPath, strings.NewReader(form.Encode()))
	if err != nil {
		panic(err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Paynow v2 docs indicate Basic Auth using Integration ID and Key
	//auth := base64.StdEncoding.EncodeToString([]byte(c.IntegrationID + ":" + c.IntegrationKey))
	//httpReq.Header.Set("Authorization", "Basic "+auth)

	cli := c.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: 25 * time.Second}
	}

	resp, err := cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status submitting request: %s", resp.Status)
	}
	// Read the response body
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the URL-encoded response
	values, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response form: %w", err)
	}

	ec := &ExpressCheckoutResponse{
		Status:          values.Get("status"),
		Instructions:    values.Get("instructions"),
		PayNowReference: values.Get("paynowreference"),
		PollURL:         values.Get("pollurl"),
		Hash:            values.Get("hash"),
		OtpReference:    values.Get("otpreference"),
		RemoteOtpUrl:    values.Get("remoteotpurl"),
	}

	return ec, nil
}

func (c *PaynowClient) PollStatus(ctx context.Context, pollUrl string) (*CheckPaymentResponse, error) {
	if c == nil {
		return nil, errors.New("nil client")
	}

	slog.Info("PollStatus Request", "pollUrl", pollUrl)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pollUrl, nil)
	if err != nil {
		panic(err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	cli := c.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: 25 * time.Second}
	}

	resp, err := cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status submitting request: %s", resp.Status)
	}
	// Read the response body
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	slog.Info("PollStatus Response", "resp", resp)
	// Parse the URL-encoded response
	values, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response form: %w", err)
	}
	amountStr := values.Get("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)

	ec := &CheckPaymentResponse{
		Reference:       values.Get("reference"),
		PayNowReference: values.Get("paynowreference"),
		Amount:          amount,
		Status:          values.Get("status"),
		PollURL:         values.Get("pollurl"),
		Hash:            values.Get("hash"),
	}

	return ec, nil
}

func (c *PaynowClient) GetMerchantTrace(ctx context.Context, req MerchantTraceRequest) (*CheckPaymentResponse, error) {
	if c == nil {
		return nil, errors.New("nil client")
	}
	req.Hash = computePaynowHashMerchantTrace(req, c.IntegrationKey)

	fullPath := fmt.Sprintf("%s/trace", c.baseURL)

	slog.Info("MerchantTrace Request", "req", req)

	form := buildPaynowMerchantTraceFields(req)

	output := strings.NewReader(form.Encode())

	slog.Info("MerchantTrace Output", "output", output)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullPath, strings.NewReader(form.Encode()))
	if err != nil {
		panic(err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	cli := c.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: 25 * time.Second}
	}

	resp, err := cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status submitting request: %s", resp.Status)
	}
	// Read the response body

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	slog.Info("MerchantTrace Response", "body", string(b))

	// Parse the URL-encoded response
	values, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response form: %w", err)
	}
	amountStr := values.Get("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)

	ec := &CheckPaymentResponse{
		Reference:       values.Get("reference"),
		PayNowReference: values.Get("paynowreference"),
		Amount:          amount,
		Status:          values.Get("status"),
		PollURL:         values.Get("pollurl"),
		Hash:            values.Get("hash"),
	}

	return ec, nil
}
