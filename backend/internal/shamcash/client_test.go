package shamcash

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func newPair(t *testing.T) (*Fake, *Client) {
	t.Helper()
	f := NewFake()
	t.Cleanup(f.Close)
	c, err := New(f.Config("https://api.example/hook", "https://shop.example"))
	if err != nil {
		t.Fatal(err)
	}
	return f, c
}

func TestJWE_RoundTripAndShape(t *testing.T) {
	j, err := newJWE(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := j.encrypt([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 5 || parts[1] != "" {
		t.Fatalf("not a direct JWE compact token: %s", tok)
	}
	hdr, _ := b64.DecodeString(parts[0])
	if string(hdr) != protectedHeader {
		t.Fatalf("header %s", hdr)
	}
	pt, err := j.decrypt(tok)
	if err != nil || string(pt) != `{"a":1}` {
		t.Fatalf("roundtrip: %s %v", pt, err)
	}
	// a flipped byte in the ciphertext is rejected (GCM tag)
	bad := parts[0] + ".." + parts[2] + "." + "AAAA" + parts[3][4:] + "." + parts[4]
	if _, err := j.decrypt(bad); !errors.Is(err, ErrCrypto) {
		t.Fatalf("tampered token accepted: %v", err)
	}
	if _, err := newJWE("c2hvcnQ="); err == nil {
		t.Fatal("short key accepted")
	}
}

func TestClient_CreateGetAndRetryOn1704(t *testing.T) {
	f, c := newPair(t)
	ctx := context.Background()
	b, err := c.CreateBill(ctx, "RG-1", 1500, CurrencySYP, "note", c.ReturnURLFor("x"))
	if err != nil {
		t.Fatal(err)
	}
	if b.StatusID != StatusPending || b.PaymentURL == "" || !c.AmountEquals(b.Amount, 1500) {
		t.Fatalf("bill %+v", b)
	}
	// a second create with the same number loads the stored bill (dropped-response retry)
	b2, err := c.CreateBill(ctx, "RG-1", 1500, CurrencySYP, "", c.ReturnURLFor("x"))
	if err != nil || b2.BillNo != "RG-1" || f.Bills() != 1 {
		t.Fatalf("retry made a second bill: %v %d", err, f.Bills())
	}
	// transport fault is retried transparently
	f.FailNext("getBillInfo", 1)
	if _, err := c.GetBill(ctx, "RG-1"); err != nil {
		t.Fatalf("getBill retry: %v", err)
	}
	if _, err := c.GetBill(ctx, "nope"); !IsResult(err, ResultBillNotFound) {
		t.Fatalf("missing bill: %v", err)
	}
}

func TestClient_RefundIdempotentAndBounded(t *testing.T) {
	f, c := newPair(t)
	ctx := context.Background()
	if _, err := c.CreateBill(ctx, "RG-2", 1000, CurrencySYP, "", c.ReturnURLFor("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RefundBill(ctx, "RG-2", 100, "key-not-paid-1", ""); !IsResult(err, ResultOriginBillNotPaid) {
		t.Fatalf("refund of unpaid bill: %v", err)
	}
	f.Pay("RG-2")
	tx1, err := c.RefundBill(ctx, "RG-2", 400, "refund-key-0001", "")
	if err != nil {
		t.Fatal(err)
	}
	tx2, err := c.RefundBill(ctx, "RG-2", 400, "refund-key-0001", "")
	if err != nil || tx1 != tx2 {
		t.Fatalf("same key gave another refund: %d %d %v", tx1, tx2, err)
	}
	if _, err := c.RefundBill(ctx, "RG-2", 700, "refund-key-0002", ""); !IsResult(err, ResultRefundExceeds) {
		t.Fatalf("over-refund accepted: %v", err)
	}
	if f.Status("RG-2") != StatusPartlyRefunded {
		t.Fatalf("status %d", f.Status("RG-2"))
	}
	if _, err := c.RefundBill(ctx, "RG-2", 1, "short", ""); err == nil {
		t.Fatal("short idempotency key accepted")
	}
}

func TestClient_TransactionsPaging(t *testing.T) {
	f, c := newPair(t)
	ctx := context.Background()
	for i := 0; i < 12; i++ {
		no := "RG-P" + string(rune('a'+i))
		if _, err := c.CreateBill(ctx, no, 10, CurrencySYP, "", c.ReturnURLFor("x")); err != nil {
			t.Fatal(err)
		}
		f.Pay(no)
	}
	p, err := c.ListTransactions(ctx, "", "", 0, 10)
	if err != nil || len(p.List) != 10 || !p.HasMore {
		t.Fatalf("page: %+v %v", p, err)
	}
	all, err := c.AllTransactions(ctx, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(all) != 12 {
		t.Fatalf("all: %d %v", len(all), err)
	}
}

func TestClient_Webhook(t *testing.T) {
	f, c := newPair(t)
	if _, err := c.CreateBill(context.Background(), "RG-W", 10, CurrencySYP, "", c.ReturnURLFor("x")); err != nil {
		t.Fatal(err)
	}
	w, err := c.ParseWebhook(f.Pay("RG-W"))
	if err != nil || w.BillNo != "RG-W" || w.StatusID != StatusPaid || w.TranID == nil {
		t.Fatalf("webhook: %+v %v", w, err)
	}
	// forged with another key
	other := NewFake()
	other.jwe, _ = newJWE(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	defer other.Close()
	if _, err := c.ParseWebhook(other.Webhook("RG-W", nil, StatusPaid)); !errors.Is(err, ErrCrypto) {
		t.Fatalf("forged webhook accepted: %v", err)
	}
	for _, body := range []string{`{}`, `not json`, `{"encData":"a.b.c"}`} {
		if _, err := c.ParseWebhook([]byte(body)); !errors.Is(err, ErrCrypto) {
			t.Fatalf("garbage accepted: %s", body)
		}
	}
	// expired token
	pt, _ := json.Marshal(map[string]any{"billNo": "RG-W", "statusId": 4, "iat": 1000, "exp": 1300})
	enc, _ := f.jwe.encrypt(pt)
	body, _ := json.Marshal(map[string]string{"encData": enc})
	if _, err := c.ParseWebhook(body); !errors.Is(err, ErrCrypto) {
		t.Fatal("expired webhook accepted")
	}
}
