// Package shamcash the client for Sham Cash's OFFICIAL merchant API (الدفع
// الإلكتروني) — built now and switched OFF (owner decision 2026-11-09: «جهّزها وخلّيها
// جاهزة»). The owner has no commercial register yet, so there are no credentials.
//
// The wire protocol mirrors the official SDKs, read line by line:
// github.com/OkToCode-L-L-C/Sham-Cash-PHP-SDK (oktocode/sham-cash-sdk 1.0.2) and the
// WooCommerce plugin github.com/obadatarazi/shamcash-payments that bundles it.
// No unofficial service is used — ever (owner rule).
//
// # Protocol (from the SDK code)
//
//   - POST {baseUrl}/api/ElectronicPayment/{createBill|getBillInfo|refundBill|getTransactions}
//     body {"encData": <JWE>, "agentKey": <agentKey>} — plain JSON, Accept/Content-Type JSON.
//   - encData = Direct JWE compact, A256GCM, header {"alg":"dir","enc":"A256GCM","cty":"json"},
//     12-byte IV, 16-byte tag, AAD = the header segment; key = Base64 32-byte secretKey.
//     The encrypted payload carries iat/exp (TTL 300 s).
//   - Responses are NOT encrypted: {"result": 2500, "succeeded": true, "message", "data": {...}}.
//     Business failures still use HTTP 200 with another result code (1703 not found, 1704
//     bill number exists, 1708 refund exceeds balance, ...).
//   - The webhook is an encrypted POST {"encData": "..."} carrying billNo, tranId, statusId,
//     iat, exp — paid (4) or expired (3). Retried until HTTP 200 within 10 s.
//   - Amounts are JSON numbers with at most two decimals (10.50), currency USD=1 / SYP=2.
//
// # What is needed to switch it ON
//
// 1. Sham Cash issues, after the verified commercial account and "Service Creation":
//    the agent key, the secret key (Base64, 32 bytes) and the base URL. Put them in the
//    server's environment — NEVER in the repo (it is public):
//
//	SHAMCASH_AGENT_KEY=...
//	SHAMCASH_SECRET_KEY=...            # Base64 32-byte AES key
//	SHAMCASH_BASE_URL=https://...      # as given by Sham Cash, no trailing path
//	SHAMCASH_CALLBACK_URL=https://api.rahalgo.com/api/v1/public/shamcash/webhook
//	SHAMCASH_RETURN_URL=https://rahalgo.com   # the browser returns to {this}/wallet/shamcash/{bill id}
//	SHAMCASH_AMOUNT_SCALE=1            # bill amount = wallet amount × scale (see below)
//
// 2. Register the callback URL above with Sham Cash if they ask for it at service creation
//    (it is also sent with every bill).
// 3. Restart the API, then turn on the setting `customers.shamcash_enabled` in the admin
//    settings (customers group). `customers.shamcash_min` / `customers.shamcash_max` bound
//    each bill. While the setting is off OR the env is incomplete, every endpoint answers
//    `shamcash_unavailable` and the apps hide the option.
//
// SHAMCASH_AMOUNT_SCALE: RahalGo amounts are in the NEW Syrian pound (1000 old = 10 new).
// If Sham Cash's SYP currency is still the old pound, set the scale to 100 — otherwise a
// customer would pay a hundredth of the amount. Confirm with Sham Cash before going live.
package shamcash

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Result codes (Enum/ResultCode.php).
const (
	ResultSuccess             = 2500
	ResultBillNotFound        = 1703
	ResultBillNoAlreadyExists = 1704
	ResultEncryptedInvalid    = 1705
	ResultOriginBillNotPaid   = 1707
	ResultRefundExceeds       = 1708
)

// Bill statuses (Enum/BillStatus.php).
const (
	StatusPending        = 1
	StatusRefund         = 2
	StatusExpired        = 3
	StatusPaid           = 4
	StatusPartlyRefunded = 5
)

// Currencies (Enum/Currency.php).
const (
	CurrencyUSD = 1
	CurrencySYP = 2
)

// Transaction types (Enum/TransactionType.php).
const (
	TranPayment = 1
	TranRefund  = 2
)

// Config the credentials and URLs — from the environment only.
type Config struct {
	AgentKey    string
	SecretKey   string
	BaseURL     string
	CallbackURL string
	ReturnURL   string
	// AmountScale multiplies wallet units into Sham Cash currency units (default 1).
	AmountScale int64
	Timeout     time.Duration
}

// ConfigFromEnv reads SHAMCASH_* (see the package comment).
func ConfigFromEnv() Config {
	scale, _ := strconv.ParseInt(strings.TrimSpace(os.Getenv("SHAMCASH_AMOUNT_SCALE")), 10, 64)
	return Config{
		AgentKey:    strings.TrimSpace(os.Getenv("SHAMCASH_AGENT_KEY")),
		SecretKey:   strings.TrimSpace(os.Getenv("SHAMCASH_SECRET_KEY")),
		BaseURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("SHAMCASH_BASE_URL")), "/"),
		CallbackURL: strings.TrimSpace(os.Getenv("SHAMCASH_CALLBACK_URL")),
		ReturnURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("SHAMCASH_RETURN_URL")), "/"),
		AmountScale: scale,
	}
}

// Complete whether every value needed to talk to Sham Cash is present.
func (c Config) Complete() bool {
	return c.AgentKey != "" && c.SecretKey != "" && c.BaseURL != "" && c.CallbackURL != "" && c.ReturnURL != ""
}

// APIError a business failure (HTTP 200, result ≠ 2500).
type APIError struct {
	Result  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("shamcash: result %d: %s", e.Result, e.Message) }

// ErrTransport the request may or may not have reached Sham Cash (timeout, 5xx, bad JSON).
var ErrTransport = errors.New("shamcash: transport failure")

// IsResult reports whether err is an APIError with that code.
func IsResult(err error, code int) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.Result == code
}

// Bill the data object of createBill / getBillInfo.
type Bill struct {
	StatusID   int      `json:"statusId"`
	StatusName string   `json:"statusName"`
	BillNo     string   `json:"billNo"`
	Amount     string   `json:"amount"`
	CurrencyID int      `json:"currencyId"`
	TranID     *int64   `json:"tranId"`
	PaymentURL string   `json:"paymentUrl"`
	Rem        string   `json:"rem"`
	Refunds    []Refund `json:"refunds"`
	Raw        json.RawMessage
}

// Refund one refund inside a bill.
type Refund struct {
	TranID int64  `json:"tranId"`
	Amount string `json:"amount"`
}

// Transaction one row of getTransactions.
type Transaction struct {
	BillNo      string `json:"billNo"`
	Amount      string `json:"amount"`
	CurrencyID  int    `json:"currencyId"`
	CreatedDate string `json:"createdDate"`
	CreatedTime string `json:"createdTime"`
	TranID      int64  `json:"tranId"`
	TranTypeID  int    `json:"tranTypeId"`
}

// TransactionPage one page of getTransactions.
type TransactionPage struct {
	List               []Transaction `json:"list"`
	HasMore            bool          `json:"hasMore"`
	LastReturnedTranID int64         `json:"lastReturnedTranId"`
}

// Webhook the decrypted notification.
type Webhook struct {
	BillNo   string `json:"billNo"`
	TranID   *int64 `json:"tranId"`
	StatusID int    `json:"statusId"`
	Iat      int64  `json:"iat"`
	Exp      int64  `json:"exp"`
}

// Client talks to one Sham Cash merchant service.
type Client struct {
	cfg  Config
	jwe  *jwe
	http *http.Client
	now  func() time.Time
	// tokenTTL / skew as in the SDK (300 s, 30 s).
	tokenTTL int64
	skew     int64
}

// New builds a client; it fails on a malformed secret.
func New(cfg Config) (*Client, error) {
	if !cfg.Complete() {
		return nil, errors.New("shamcash: incomplete configuration")
	}
	j, err := newJWE(cfg.SecretKey)
	if err != nil {
		return nil, err
	}
	if cfg.AmountScale <= 0 {
		cfg.AmountScale = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	return &Client{cfg: cfg, jwe: j, http: &http.Client{Timeout: cfg.Timeout},
		now: time.Now, tokenTTL: 300, skew: 30}, nil
}

// Config the configuration in use (without exposing it to responses).
func (c *Client) Config() Config { return c.cfg }

// FormatAmount wallet units → the decimal literal Sham Cash expects.
func (c *Client) FormatAmount(units int64) string {
	return strconv.FormatInt(units*c.cfg.AmountScale, 10)
}

// AmountEquals whether a Sham Cash decimal equals local wallet units.
func (c *Client) AmountEquals(remote string, units int64) bool {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(remote))
	if !ok {
		return false
	}
	return r.Cmp(new(big.Rat).SetInt64(units*c.cfg.AmountScale)) == 0
}

// ReturnURLFor where the browser comes back to after paying a bill.
func (c *Client) ReturnURLFor(billID string) string {
	return c.cfg.ReturnURL + "/wallet/shamcash/" + billID
}

func (c *Client) encrypt(payload map[string]any) (string, error) {
	now := c.now().Unix()
	payload["iat"] = now
	payload["exp"] = now + c.tokenTTL
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return c.jwe.encrypt(b)
}

type envelope struct {
	Result    *int            `json:"result"`
	Succeeded *bool           `json:"succeeded"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
}

// post one call; transport problems are retried `attempts` times (only for safe calls).
func (c *Client) post(ctx context.Context, action string, payload map[string]any, attempts int) (json.RawMessage, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i) * 500 * time.Millisecond):
			}
		}
		data, err := c.postOnce(ctx, action, payload)
		if err == nil || !errors.Is(err, ErrTransport) {
			return data, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *Client) postOnce(ctx context.Context, action string, payload map[string]any) (json.RawMessage, error) {
	enc, err := c.encrypt(payload)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]string{"encData": enc, "agentKey": c.cfg.AgentKey})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/api/ElectronicPayment/"+action, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "rahalgo-shamcash-go/1.0")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransport, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransport, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d for %s", ErrTransport, resp.StatusCode, action)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Result == nil || env.Succeeded == nil {
		return nil, fmt.Errorf("%w: malformed response for %s", ErrTransport, action)
	}
	if !*env.Succeeded || *env.Result != ResultSuccess {
		return nil, &APIError{Result: *env.Result, Message: env.Message}
	}
	if len(env.Data) == 0 || env.Data[0] != '{' {
		return nil, fmt.Errorf("%w: no data object for %s", ErrTransport, action)
	}
	return env.Data, nil
}

// numbers decode JSON numbers as strings so amounts keep their exact literal.
func decodeData(data json.RawMessage, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var m map[string]any
	if err := d.Decode(&m); err != nil {
		return fmt.Errorf("%w: %v", ErrTransport, err)
	}
	stringifyNumbers(m)
	b, _ := json.Marshal(m)
	return json.Unmarshal(b, out)
}

// stringifyNumbers turns amount-like json.Number into strings and keeps ids numeric.
func stringifyNumbers(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			if n, ok := x.(json.Number); ok && k == "amount" {
				t[k] = n.String()
				continue
			}
			stringifyNumbers(x)
		}
	case []any:
		for _, x := range t {
			stringifyNumbers(x)
		}
	}
}

func (c *Client) bill(data json.RawMessage) (*Bill, error) {
	var b Bill
	if err := decodeData(data, &b); err != nil {
		return nil, err
	}
	if b.BillNo == "" {
		return nil, fmt.Errorf("%w: bill without billNo", ErrTransport)
	}
	b.Raw = data
	return &b, nil
}

// CreateBill creates a bill; a dropped response followed by 1704 loads the stored bill
// (as the SDK's retry example does), so a retry never makes a second bill.
func (c *Client) CreateBill(ctx context.Context, billNo string, units int64, currency int, note, redirectURL string) (*Bill, error) {
	payload := map[string]any{
		"billNo":      billNo,
		"amount":      json.Number(c.FormatAmount(units)),
		"currencyId":  currency,
		"callbackUrl": c.cfg.CallbackURL,
		"redirectUrl": redirectURL,
	}
	if note != "" {
		payload["note"] = note
	}
	data, err := c.post(ctx, "createBill", payload, 2)
	if IsResult(err, ResultBillNoAlreadyExists) {
		return c.GetBill(ctx, billNo)
	}
	if err != nil {
		return nil, err
	}
	return c.bill(data)
}

// GetBill reads a bill (safe to retry).
func (c *Client) GetBill(ctx context.Context, billNo string) (*Bill, error) {
	data, err := c.post(ctx, "getBillInfo", map[string]any{"billNo": billNo}, 3)
	if err != nil {
		return nil, err
	}
	return c.bill(data)
}

// RefundBill refunds (partly) a paid bill. Retrying with the same key is safe: Sham Cash
// returns the original refund when that key already succeeded.
func (c *Client) RefundBill(ctx context.Context, billNo string, units int64, idempotencyKey, note string) (int64, error) {
	if n := len([]rune(idempotencyKey)); n < 10 || n > 100 {
		return 0, errors.New("shamcash: idempotency key must be 10..100 characters")
	}
	payload := map[string]any{
		"billNo":         billNo,
		"amount":         json.Number(c.FormatAmount(units)),
		"idempotencyKey": idempotencyKey,
	}
	if note != "" {
		payload["note"] = note
	}
	data, err := c.post(ctx, "refundBill", payload, 3)
	if err != nil {
		return 0, err
	}
	var out struct {
		TranID int64 `json:"tranId"`
	}
	if err := decodeData(data, &out); err != nil {
		return 0, err
	}
	if out.TranID == 0 {
		return 0, fmt.Errorf("%w: refund without tranId", ErrTransport)
	}
	return out.TranID, nil
}

// ListTransactions one page (limit 10..2500; dates yyyy-MM-dd).
func (c *Client) ListTransactions(ctx context.Context, from, to string, afterTranID int64, limit int) (*TransactionPage, error) {
	if limit < 10 || limit > 2500 {
		return nil, errors.New("shamcash: limit must be 10..2500")
	}
	payload := map[string]any{"afterTranId": afterTranID, "limit": limit}
	if from != "" {
		payload["fromDate"] = from
	}
	if to != "" {
		payload["toDate"] = to
	}
	data, err := c.post(ctx, "getTransactions", payload, 3)
	if err != nil {
		return nil, err
	}
	var p TransactionPage
	if err := decodeData(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// AllTransactions follows hasMore/lastReturnedTranId like the SDK's eachTransaction.
func (c *Client) AllTransactions(ctx context.Context, from, to string) ([]Transaction, error) {
	var out []Transaction
	var after int64
	for {
		p, err := c.ListTransactions(ctx, from, to, after, 500)
		if err != nil {
			return nil, err
		}
		out = append(out, p.List...)
		if !p.HasMore {
			return out, nil
		}
		if len(p.List) == 0 || p.LastReturnedTranID <= after {
			return nil, fmt.Errorf("%w: pagination cursor did not advance", ErrTransport)
		}
		after = p.LastReturnedTranID
	}
}

// ParseWebhook decrypts the raw body (do not re-encode it) and checks iat/exp like the SDK.
func (c *Client) ParseWebhook(raw []byte) (*Webhook, error) {
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, ErrCrypto
	}
	enc, _ := env["encData"].(string)
	if enc == "" {
		return nil, ErrCrypto
	}
	pt, err := c.jwe.decrypt(enc)
	if err != nil {
		return nil, err
	}
	var w Webhook
	d := json.NewDecoder(bytes.NewReader(pt))
	if err := d.Decode(&w); err != nil || w.BillNo == "" || w.StatusID == 0 || w.Iat == 0 || w.Exp == 0 {
		return nil, ErrCrypto
	}
	now := c.now().Unix()
	if w.Iat > now+c.skew || now > w.Exp+c.skew {
		return nil, ErrCrypto
	}
	return &w, nil
}
