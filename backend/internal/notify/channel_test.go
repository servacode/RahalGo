package notify

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// **قناةُ الرمز — أيُّ طريقٍ سلكه ومتى.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «نقدر نفعّلها أو نطفّيها من لوحة التحكّم».)
//
// **وما يُقاس هنا ليس أنّ الرسالةَ وصلت** — ذلك يحتاج مزوّداً. **بل
// أيَّ بابٍ طرق المحرّكُ ومتى تحوّل عنه.**

// fakeWA **واتسابٌ يقول ما أُمر أن يقول.**
type fakeWA struct {
	calls atomic.Int32
	err   error
}

func (f *fakeWA) SendOTP(context.Context, string, string) error {
	f.calls.Add(1)
	return f.err
}

// smsAt **بوّابةٌ تعدّ ما وصلها** — وتردّ ما طُلب منها.
func smsAt(t *testing.T, status int) (*SMSSender, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return NewSMSSender(SMSConfig{URL: srv.URL, Body: `{"to":"{phone}","text":"{text}"}`},
		slog.New(slog.DiscardHandler)), &hits
}

func chanWith(wa OTPSender, sms *SMSSender, mode string) *OTPChannel {
	return NewOTPChannel(wa, sms, func(code string) string { return "رمز: " + code },
		func(context.Context) string { return mode },
		slog.New(slog.DiscardHandler))
}

// TestChannel_WhatsAppOnly **والافتراضُ واتسابُ وحدَه.**
func TestChannel_WhatsAppOnly(t *testing.T) {
	wa := &fakeWA{}
	sms, hits := smsAt(t, 200)
	if err := chanWith(wa, sms, ChannelWhatsApp).SendOTP(t.Context(), "+963900000000", "123456"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if wa.calls.Load() != 1 {
		t.Errorf("**واتساب لم يُنادَ** — %d", wa.calls.Load())
	}
	if hits.Load() != 0 {
		t.Errorf("**رسالةٌ نصّيّةٌ أُرسلت والقناةُ واتساب** — %d", hits.Load())
	}
}

// TestChannel_SMSOnly **والرسائلُ وحدَها حين تُضبط.**
func TestChannel_SMSOnly(t *testing.T) {
	wa := &fakeWA{}
	sms, hits := smsAt(t, 200)
	if err := chanWith(wa, sms, ChannelSMS).SendOTP(t.Context(), "+963900000000", "123456"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("**البوّابةُ لم تُنادَ** — %d", hits.Load())
	}
	if wa.calls.Load() != 0 {
		t.Errorf("**واتساب نُودي والقناةُ رسائل** — %d", wa.calls.Load())
	}
}

// TestChannel_SMSWithoutGatewayFallsBack **ولا يُبدَّل إلى ما لا يعمل.**
//
// **ومن ضبط `sms` بلا بوّابةٍ أغلق بابَ الدخول على الجميع** — **وخيارٌ
// في اللوحة يُطفئ المنصّةَ بضغطةٍ ليس خياراً بل فخّ.**
func TestChannel_SMSWithoutGatewayFallsBack(t *testing.T) {
	wa := &fakeWA{}
	bare := NewSMSSender(SMSConfig{}, slog.New(slog.DiscardHandler))
	if err := chanWith(wa, bare, ChannelSMS).SendOTP(t.Context(), "+963900000000", "123456"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if wa.calls.Load() != 1 {
		t.Error("**قناةُ رسائلَ بلا بوّابةٍ لم تُردَّ إلى واتساب** — بابُ الدخول يقف")
	}
}

// TestChannel_FallsBackWhenWhatsAppFails **وSMS لمن لا يصله واتساب.**
//
// **ورقمٌ ليس على واتساب حالٌ قائمةٌ لا احتمال** — `ErrNoWhatsApp`
// كُتبت لأنّها وقعت.
func TestChannel_FallsBackWhenWhatsAppFails(t *testing.T) {
	wa := &fakeWA{err: errors.New("الرقمُ ليس على واتساب")}
	sms, hits := smsAt(t, 200)
	if err := chanWith(wa, sms, ChannelWhatsAppThen).SendOTP(t.Context(), "+963900000000", "123456"); err != nil {
		t.Fatalf("الإرسالُ فشل والبديلُ يعمل: %v", err)
	}
	if wa.calls.Load() != 1 {
		t.Errorf("**واتساب لم يُجرَّب أوّلاً** — %d", wa.calls.Load())
	}
	if hits.Load() != 1 {
		t.Errorf("**لم تُجرَّب الرسالةُ النصّيّةُ بعد فشل واتساب** — %d", hits.Load())
	}
}

// TestChannel_NoFallbackWhenWhatsAppWorks **ولا تُرسَل رسالتان.**
//
// **ورسالتان لرمزٍ واحدٍ تُكلّفان مرّتين** — ويقرؤهما صاحبُهما شكّاً في
// حسابه.
func TestChannel_NoFallbackWhenWhatsAppWorks(t *testing.T) {
	wa := &fakeWA{}
	sms, hits := smsAt(t, 200)
	if err := chanWith(wa, sms, ChannelWhatsAppThen).SendOTP(t.Context(), "+963900000000", "123456"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if hits.Load() != 0 {
		t.Errorf("**رسالةٌ نصّيّةٌ أُرسلت وواتساب نجح** — %d", hits.Load())
	}
}

// TestChannel_BothFailReturnsWhatsAppError **ويُردّ خطأُ القناة الأولى.**
//
// **ورسالةُ «بوّابةُ الرسائل لا تستجيب» لا تعني شيئاً لمن طلب رمزاً
// على واتساب.**
func TestChannel_BothFailReturnsWhatsAppError(t *testing.T) {
	waErr := errors.New("الرقمُ ليس على واتساب")
	wa := &fakeWA{err: waErr}
	sms, _ := smsAt(t, 500)
	err := chanWith(wa, sms, ChannelWhatsAppThen).SendOTP(t.Context(), "+963900000000", "123456")
	if !errors.Is(err, waErr) {
		t.Errorf("**رُدَّ خطأُ البوّابة لا خطأُ واتساب**: %v", err)
	}
}

// TestChannel_EmptyModeIsWhatsApp **وإعدادٌ لم يُضبط بعد يعمل كما كان.**
func TestChannel_EmptyModeIsWhatsApp(t *testing.T) {
	wa := &fakeWA{}
	sms, hits := smsAt(t, 200)
	if err := chanWith(wa, sms, "").SendOTP(t.Context(), "+963900000000", "123456"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if wa.calls.Load() != 1 || hits.Load() != 0 {
		t.Errorf("**قناةٌ فارغةٌ لم تُقرأ واتساب** — واتساب %d · رسائل %d",
			wa.calls.Load(), hits.Load())
	}
}

// TestSMS_ArabicIsHexEncoded **والعربيّةُ تُرمَّز حين تُطلب مُرمَّزة.**
//
// **ومعيارُ الرسائل يعرف أبجديّتين**: GSM-7 لِلاتينيّة، **وUCS-2 لكلّ
// ما عداها** — وكثيرٌ من البوّابات تطلبها ستّةَ عشرَ نظاما.
//
// **ومن أرسلها نصّاً خامّاً وصلت علاماتِ استفهام** — **ولا خطأَ ولا
// سجلّ**: البوّابةُ تردّ ٢٠٠ والرسالةُ تصل ممسوخة.
func TestSMS_ArabicIsHexEncoded(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := NewSMSSender(SMSConfig{
		URL:  srv.URL,
		Body: `{"to":"{phone}","text":"{text_hex}","type":"1"}`,
	}, slog.New(slog.DiscardHandler))
	if err := s.SendText(t.Context(), "+963900000000", "رمز: 12"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	// **«رمز: 12»** — ر=0631 م=0645 ز=0632 مسافة=0020 :=003A … والأرقامُ
	// لاتينيّةٌ في القالب.
	if !strings.Contains(got, `"text":"06310645063200`) {
		t.Errorf("**العربيّةُ لم تُرمَّز**: %s", got)
	}
	if strings.Contains(got, "رمز") {
		t.Errorf("**النصُّ الخامُّ مرّ كما هو**: %s", got)
	}
}

// TestSMS_HexIsFromRawNotEscaped **والترميزُ من النصّ الخامّ لا المهرَّب.**
//
// **وحسابُه من نصٍّ هُرِّب لِلJSON يُدخل شرطةً مائلةً في الترميز** —
// **وهو خطأٌ لا يظهر في العربيّة** فيبقى نائماً حتّى يكتب المالكُ
// علامةَ اقتباسٍ في قالبه.
func TestSMS_HexIsFromRawNotEscaped(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := NewSMSSender(SMSConfig{URL: srv.URL, Body: `{"t":"{text_hex}"}`},
		slog.New(slog.DiscardHandler))
	if err := s.SendText(t.Context(), "+963900000000", `"x`); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	// **`"` هي 0022 و`x` هي 0078** — ولا شرطةَ مائلةً بينهما.
	if !strings.Contains(got, `"t":"00220078"`) {
		t.Errorf("**الترميزُ حُسب من نصٍّ مهرَّب**: %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **بوّابةُ قوالبَ حقيقيّة — LinkSyria**
// ══════════════════════════════════════════════════════════════════════
//
// (قِيس من مخطّطهم الرسميّ `/api/v1/schema/client/` ٢٠٢٦-٠٨-٢٠.)
//
//	POST /api/v1/otp/send/
//	X-API-Key: ls_…
//	Idempotency-Key: …
//	{"phone_number":"+963…","custom_code":"123456","language":"ar"}
//
// **وهم يؤلّفون نصَّ الرسالة، ونحن نمرّر الرمز** — **ومن تركهم
// يولّدونه أرسل إلى الزبون رمزاً غيرَ الذي خُزِّن له فلا يدخل أبدا.**

// TestSMS_TemplateGatewayGetsTheCode **والرمزُ يصل البوّابةَ كما وُلّد.**
func TestSMS_TemplateGatewayGetsTheCode(t *testing.T) {
	var body, key, idem, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body, path = string(b), r.URL.Path
		key, idem = r.Header.Get("X-API-Key"), r.Header.Get("Idempotency-Key")
		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	s := NewSMSSender(SMSConfig{
		URL:        srv.URL + "/api/v1/otp/send/",
		Body:       `{"phone_number":"{phone}","custom_code":"{code}","language":"ar"}`,
		AuthHeader: "X-API-Key: ls_test | Idempotency-Key: {idem}",
	}, slog.New(slog.DiscardHandler))

	if err := s.SendOTP(t.Context(), "+963912345678", "582093", "رمز: 582093"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if path != "/api/v1/otp/send/" {
		t.Errorf("المسار %q", path)
	}
	want := `{"phone_number":"+963912345678","custom_code":"582093","language":"ar"}`
	if body != want {
		t.Errorf("**الجسمُ لا يطابق العقد**\n  جاء: %s\n  يُنتظر: %s", body, want)
	}
	if key != "ls_test" {
		t.Errorf("**المفتاحُ لم يُرسَل**: %q", key)
	}
	// **وترويستان من حقلٍ واحد** — انظر `sms.go`.
	if len(idem) != 24 {
		t.Errorf("**مفتاحُ منع التكرار لم يُملأ**: %q", idem)
	}
}

// TestSMS_IdemKeyIsStableAndHidesCode **ومفتاحُ التكرار ثابتٌ ولا يكشف.**
//
// **وشبكةٌ تتعثّر بعد أن وصلت الرسالة تجعل المحرّكَ يعيد النداء** —
// **ورسالتان لرمزٍ واحدٍ تُحاسَبان مرّتين.**
//
// **ولا يُكتب الرمزُ فيه**: يمرّ في ترويسةٍ تُسجَّل عند المزوّد،
// **ورمزُ تحقّقٍ في سجلٍّ ليس رمزَ تحقّق.**
func TestSMS_IdemKeyIsStableAndHidesCode(t *testing.T) {
	a := idemKey("+963912345678", "582093")
	if a != idemKey("+963912345678", "582093") {
		t.Error("**المفتاحُ يتبدّل بين نداءين متطابقين** — فتُحاسَب الرسالةُ مرّتين")
	}
	if a == idemKey("+963912345678", "111111") {
		t.Error("**رمزان مختلفان بمفتاحٍ واحد** — فتُبتلع الرسالةُ الثانية")
	}
	if a == idemKey("+963900000000", "582093") {
		t.Error("**رقمان مختلفان بمفتاحٍ واحد**")
	}
	if strings.Contains(a, "582093") {
		t.Errorf("**الرمزُ ظاهرٌ في المفتاح**: %s", a)
	}
}

// TestSMS_TextSenderCarriesNoCode **ورسالةُ المتجر لا رمزَ فيها.**
//
// **ومن وحّد البوّابتين أرسل إلى المتاجر رموزَ تحقّقٍ بدل إشعارات
// الطلب** — فيولّد المزوّدُ رمزاً من عنده ويرسله لصاحب المطعم.
func TestSMS_TextSenderCarriesNoCode(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := NewSMSSender(SMSConfig{URL: srv.URL, Body: `{"c":"{code}","t":"{text}"}`},
		slog.New(slog.DiscardHandler))
	if err := s.SendText(t.Context(), "+963912345678", "طلب جديد"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if body != `{"c":"","t":"طلب جديد"}` {
		t.Errorf("**نصٌّ حرٌّ حمل رمزا**: %s", body)
	}
}

// TestSMS_PhonePlainDropsThePlus **ورقمٌ بلا زائدٍ لمن يطلبه كذلك.**
func TestSMS_PhonePlainDropsThePlus(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := NewSMSSender(SMSConfig{URL: srv.URL, Body: `{"to":"{phone_plain}"}`},
		slog.New(slog.DiscardHandler))
	if err := s.SendText(t.Context(), "+963912345678", "x"); err != nil {
		t.Fatalf("الإرسالُ فشل: %v", err)
	}
	if body != `{"to":"963912345678"}` {
		t.Errorf("**الزائدُ لم يُحذف**: %s", body)
	}
}
