package shamcash

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"
)

// Fake **a fake Sham Cash server for tests** — the same encrypted protocol as the real
// API (requests are opened with the secret, iat/exp checked, responses in the
// result/succeeded/data envelope, webhooks sealed the same way). Nothing here talks to
// the network beyond httptest.
type Fake struct {
	Srv      *httptest.Server
	AgentKey string
	Secret   string

	mu      sync.Mutex
	jwe     *jwe
	bills   map[string]*fakeBill
	nextTx  int64
	refKeys map[string]int64 // idempotency key → refund tranId
	txs     []Transaction
	// FailNext makes the next N calls of an action answer HTTP 500 (transport fault).
	failNext map[string]int
	// Calls counts calls per action.
	Calls map[string]int
}

type fakeBill struct {
	no, amount, rem string
	currency, st    int
	tranID          *int64
	refunds         []Refund
	refunded        *big.Rat
	created         time.Time
}

// NewFake starts the fake with a random 32-byte secret.
func NewFake() *Fake {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i*7 + 3)
	}
	secret := base64.StdEncoding.EncodeToString(key)
	j, _ := newJWE(secret)
	f := &Fake{AgentKey: "agent-test-key", Secret: secret, jwe: j,
		bills: map[string]*fakeBill{}, nextTx: 3919000, refKeys: map[string]int64{},
		failNext: map[string]int{}, Calls: map[string]int{}}
	f.Srv = httptest.NewServer(http.HandlerFunc(f.serve))
	return f
}

// Close stops the server.
func (f *Fake) Close() { f.Srv.Close() }

// Config a client configuration pointing at the fake.
func (f *Fake) Config(callback, ret string) Config {
	return Config{AgentKey: f.AgentKey, SecretKey: f.Secret, BaseURL: f.Srv.URL,
		CallbackURL: callback, ReturnURL: ret, AmountScale: 1, Timeout: 5 * time.Second}
}

// FailNext the next n calls of action fail with HTTP 500.
func (f *Fake) FailNext(action string, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext[action] = n
}

func (f *Fake) reply(w http.ResponseWriter, result int, msg string, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"result": result, "succeeded": result == ResultSuccess, "message": msg, "data": data})
}

func (f *Fake) billData(b *fakeBill) map[string]any {
	refunds := []any{}
	for _, r := range b.refunds {
		refunds = append(refunds, map[string]any{"tranId": r.TranID, "amount": json.Number(r.Amount),
			"createdDate": b.created.Format("2006-01-02"), "createdTime": "10:00", "rem": ""})
	}
	names := map[int]string{1: "pending", 2: "refund", 3: "expired", 4: "paid", 5: "partly refunded"}
	return map[string]any{
		"statusId": b.st, "statusName": names[b.st], "billNo": b.no, "amount": json.Number(b.amount),
		"currencyId": b.currency, "createdDate": b.created.Format("2006-01-02"), "createdTime": "10:00",
		"tranId": b.tranID, "paymentUrl": f.Srv.URL + "/pay/" + b.no, "rem": b.rem, "refunds": refunds,
	}
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/ElectronicPayment/"
	if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	action := strings.TrimPrefix(r.URL.Path, prefix)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls[action]++
	if f.failNext[action] > 0 {
		f.failNext[action]--
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	raw, _ := io.ReadAll(r.Body)
	var env struct {
		EncData  string `json:"encData"`
		AgentKey string `json:"agentKey"`
	}
	if json.Unmarshal(raw, &env) != nil || env.AgentKey != f.AgentKey {
		f.reply(w, 1701, "agent key invalid", nil)
		return
	}
	pt, err := f.jwe.decrypt(env.EncData)
	if err != nil {
		f.reply(w, ResultEncryptedInvalid, "encrypted data invalid", nil)
		return
	}
	d := json.NewDecoder(bytes.NewReader(pt))
	d.UseNumber()
	var p map[string]any
	if d.Decode(&p) != nil {
		f.reply(w, ResultEncryptedInvalid, "encrypted data invalid", nil)
		return
	}
	iat, _ := p["iat"].(json.Number).Int64()
	exp, _ := p["exp"].(json.Number).Int64()
	if now := time.Now().Unix(); iat == 0 || exp < now || iat > now+60 {
		f.reply(w, ResultEncryptedInvalid, "token expired", nil)
		return
	}
	str := func(k string) string {
		switch v := p[k].(type) {
		case string:
			return v
		case json.Number:
			return v.String()
		}
		return ""
	}
	switch action {
	case "createBill":
		no := str("billNo")
		if _, ok := f.bills[no]; ok {
			f.reply(w, ResultBillNoAlreadyExists, "bill number already exists", nil)
			return
		}
		if str("callbackUrl") == "" || str("redirectUrl") == "" {
			f.reply(w, 1000, "urls required", nil)
			return
		}
		cur, _ := p["currencyId"].(json.Number).Int64()
		b := &fakeBill{no: no, amount: str("amount"), rem: str("note"), currency: int(cur),
			st: StatusPending, refunded: new(big.Rat), created: time.Now()}
		f.bills[no] = b
		f.reply(w, ResultSuccess, "Success", f.billData(b))
	case "getBillInfo":
		b, ok := f.bills[str("billNo")]
		if !ok {
			f.reply(w, ResultBillNotFound, "bill not found", nil)
			return
		}
		f.reply(w, ResultSuccess, "Success", f.billData(b))
	case "refundBill":
		b, ok := f.bills[str("billNo")]
		if !ok {
			f.reply(w, ResultBillNotFound, "bill not found", nil)
			return
		}
		key := str("idempotencyKey")
		if tx, ok := f.refKeys[key]; ok {
			f.reply(w, ResultSuccess, "Success", map[string]any{"tranId": tx})
			return
		}
		if b.st != StatusPaid && b.st != StatusPartlyRefunded {
			f.reply(w, ResultOriginBillNotPaid, "origin bill not paid", nil)
			return
		}
		amt, _ := new(big.Rat).SetString(str("amount"))
		total, _ := new(big.Rat).SetString(b.amount)
		if amt == nil || amt.Sign() <= 0 || new(big.Rat).Add(b.refunded, amt).Cmp(total) > 0 {
			f.reply(w, ResultRefundExceeds, "refund exceeds balance", nil)
			return
		}
		f.nextTx++
		tx := f.nextTx
		b.refunded.Add(b.refunded, amt)
		b.refunds = append(b.refunds, Refund{TranID: tx, Amount: str("amount")})
		if b.refunded.Cmp(total) == 0 {
			b.st = StatusRefund
		} else {
			b.st = StatusPartlyRefunded
		}
		f.refKeys[key] = tx
		f.txs = append(f.txs, Transaction{BillNo: b.no, Amount: str("amount"), CurrencyID: b.currency,
			CreatedDate: time.Now().Format("2006-01-02"), CreatedTime: "10:00", TranID: tx, TranTypeID: TranRefund})
		f.reply(w, ResultSuccess, "Success", map[string]any{"tranId": tx})
	case "getTransactions":
		after, _ := p["afterTranId"].(json.Number).Int64()
		limit, _ := p["limit"].(json.Number).Int64()
		from, to := str("fromDate"), str("toDate")
		sort.Slice(f.txs, func(i, j int) bool { return f.txs[i].TranID < f.txs[j].TranID })
		list := []any{}
		var last int64
		more := false
		for _, t := range f.txs {
			if t.TranID <= after || (from != "" && t.CreatedDate < from) || (to != "" && t.CreatedDate > to) {
				continue
			}
			if int64(len(list)) >= limit {
				more = true
				break
			}
			list = append(list, map[string]any{"billNo": t.BillNo, "amount": json.Number(t.Amount),
				"currencyId": t.CurrencyID, "createdDate": t.CreatedDate, "createdTime": t.CreatedTime,
				"tranId": t.TranID, "tranTypeId": t.TranTypeID, "tranType": map[int]string{1: "payment", 2: "refund"}[t.TranTypeID]})
			last = t.TranID
		}
		f.reply(w, ResultSuccess, "Success", map[string]any{"list": list, "hasMore": more, "lastReturnedTranId": last})
	default:
		http.NotFound(w, r)
	}
}

// Pay marks the bill paid (as if the customer paid in the Sham Cash app) and returns the
// encrypted webhook body Sham Cash would POST.
func (f *Fake) Pay(billNo string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.bills[billNo]
	if b == nil {
		panic("fake shamcash: unknown bill " + billNo)
	}
	f.nextTx++
	tx := f.nextTx
	b.tranID, b.st = &tx, StatusPaid
	f.txs = append(f.txs, Transaction{BillNo: b.no, Amount: b.amount, CurrencyID: b.currency,
		CreatedDate: time.Now().Format("2006-01-02"), CreatedTime: "10:00", TranID: tx, TranTypeID: TranPayment})
	return f.webhookLocked(b.no, &tx, StatusPaid)
}

// Expire marks the bill expired and returns the webhook body.
func (f *Fake) Expire(billNo string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if b := f.bills[billNo]; b != nil {
		b.st = StatusExpired
	}
	return f.webhookLocked(billNo, nil, StatusExpired)
}

// Webhook a webhook body for any (possibly lying) status — the bill itself is untouched.
func (f *Fake) Webhook(billNo string, tranID *int64, status int) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.webhookLocked(billNo, tranID, status)
}

func (f *Fake) webhookLocked(billNo string, tranID *int64, status int) []byte {
	now := time.Now().Unix()
	pt, _ := json.Marshal(map[string]any{"billNo": billNo, "tranId": tranID, "statusId": status,
		"iat": now, "exp": now + 300})
	enc, _ := f.jwe.encrypt(pt)
	b, _ := json.Marshal(map[string]string{"encData": enc})
	return b
}

// Status the fake's current status of a bill.
func (f *Fake) Status(billNo string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	if b := f.bills[billNo]; b != nil {
		return b.st
	}
	return 0
}

// Bills how many bills were created.
func (f *Fake) Bills() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bills)
}

// String for debugging.
func (f *Fake) String() string { return fmt.Sprintf("fake shamcash at %s", f.Srv.URL) }
