package orders

// ══════════════════════════════════════════════════════════════════════
// **مستلمو «لدي توصيلة» السابقون — يُقترحون وقتَ الكتابة** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «بنقطة المتجر وقت يبعث طلب لزبون لازم يصير في مثل التخزين لمعلوماتهم بحيث اذا رجع
// بعث مره ثانية بمجرد ما يكتب اسم الزبون يتعبى باقي المعلومات بشكل تلقائي عنوان موقع
// رقم وكلشي»
//
// # من أين تُقرأ
//
// **من توصيلات المتجر نفسِها** (`kind = 'merchant_delivery'`) — لا جدولَ ثانٍ يُنسخ إليه
// فيشيخ. **صفٌّ لكلّ رقم**، والاسمُ والعنوانُ والنقطةُ من **آخر** توصيلةٍ لذلك الرقم.
//
// # وما لا يُعطى
//
//   - **متجرٌ لا يرى مستلِمي غيره** — `merchant_id` شرطٌ في كلّ قراءة، والملكيّةُ تُفحص
//     في البوّابة (`ownsMerchant`) قبلها.
//   - **ولا شيءَ لا يراه المتجرُ اليوم**: الاسمُ والرقمُ والعنوانُ تُعرض له في قائمة
//     توصيلاته (`AudienceMerchantDelivery`)، **والنقطةُ هو من حدّدها** (رابطٌ لصقه أو
//     موقعٌ طلبه — ولا يُعدَّل `dropoff` بعد الإنشاء). **وهذا النوعُ لا يُنشئه إلّا المتجر
//     ولا يُحوَّل لمتجرٍ آخر** (`transfer_delivery_kind`).
//   - **والنقطةُ إن كانت معروفةً وحدَها** (`dropoff_known`) — وإلّا فـ`dropoff` موقعُ المتجر
//     نفسِه، **ونقطةٌ كهذه تُرسل السائقَ إلى المتجر بدل المستلِم.**
//
// # و«شيلو من القائمة»
//
// مستلِمٌ كُتب غلط يُخفى (`merchant_recipients_hidden`، هجرة ٠٥٠٨) **ويرجع وحدَه** إن بعث له
// المتجرُ توصيلةً بعد إخفائه.

import (
	"context"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// MaxRecipientSuggestions **أقصى ما يُقترح** — قائمةٌ تحت حقلٍ على شاشة هاتف.
const MaxRecipientSuggestions = 8

// maxRecipientQuery **أطولُ نصِّ بحثٍ يُقرأ** — اسمٌ أو رقم، وما زاد يُقصّ.
const maxRecipientQuery = 64

// MerchantRecipient **مستلِمٌ سابقٌ للمتجر** — من آخر توصيلةٍ إلى رقمه.
type MerchantRecipient struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	AddressText string `json:"address_text"`
	// Lat/Lng **نقطةُ آخر توصيلةٍ إن حُدّدت** — وnil إن أُرسلت بلا نقطة.
	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
	// LastUsedAt **آخرُ توصيلةٍ لهذا الرقم** — والقائمةُ الأحدثُ أوّلاً.
	LastUsedAt time.Time `json:"last_used_at"`
	// Times **كم توصيلةً بعث له.**
	Times int `json:"times"`
}

// recipientsSQL **صفٌّ لكلّ رقمٍ من آخر توصيلةٍ إليه** — بمسحٍ واحدٍ لتوصيلات المتجر.
//
// $1 المتجر · $2 الاسمُ مهرَّباً لـ`LIKE` (وفارغُه لا شرط) · $3 أرقامُ الهاتف (وفارغُها لا شرط)
// · $4 الحدّ. **والاسمُ يُطابَق بعد التطبيع** (`section_name_key`، هجرة ٠٢٠٠): أ إ آ ← ا ·
// ة ← ه · ى ← ي · بلا تشكيلٍ ولا تطويل — **دالّةُ القاعدة نفسُها لا نسخةٌ ثانية.**
const recipientsSQL = `
	WITH mine AS (
		SELECT o.recipient_phone AS phone,
		       COALESCE(o.recipient_name, '') AS name,
		       COALESCE(o.address_text, '') AS address_text,
		       o.dropoff_known, o.dropoff,
		       row_number() OVER w AS rn,
		       count(*) OVER (PARTITION BY o.recipient_phone) AS times,
		       max(o.created_at) OVER (PARTITION BY o.recipient_phone) AS last_used_at
		  FROM orders o
		 WHERE o.merchant_id = $1 AND o.kind = 'merchant_delivery'
		   AND COALESCE(o.recipient_phone, '') <> ''
		WINDOW w AS (PARTITION BY o.recipient_phone ORDER BY o.created_at DESC, o.id DESC)
	)
	SELECT m.phone, m.name, m.address_text,
	       CASE WHEN m.dropoff_known THEN ST_Y(m.dropoff::geometry) END,
	       CASE WHEN m.dropoff_known THEN ST_X(m.dropoff::geometry) END,
	       m.last_used_at, m.times
	  FROM mine m
	 WHERE m.rn = 1
	   AND NOT EXISTS (
	       SELECT 1 FROM merchant_recipients_hidden h
	        WHERE h.merchant_id = $1 AND h.phone_norm = m.phone
	          AND h.hidden_at >= m.last_used_at)
	   AND (($2::text = '' AND $3::text = '')
	        OR ($2::text <> '' AND section_name_key($2::text) <> ''
	            AND section_name_key(m.name) LIKE '%' || section_name_key($2::text) || '%' ESCAPE '\')
	        OR ($3::text <> '' AND m.phone LIKE '%' || $3::text || '%' ESCAPE '\'))
	 ORDER BY m.last_used_at DESC, m.phone
	 LIMIT $4`

// MerchantRecipients **مستلمو المتجر السابقون الأقربُ لما كُتب** — الأحدثُ أوّلاً، ثمانيةٌ
// على الأكثر.
//
// **والنصُّ اسمٌ أو رقم** (`NormalizeSearch`): الأرقامُ الهنديّةُ تُحوَّل، و`09…` و`00963…`
// تصير `963…` فتطابق المخزَّن `+963…`. **والفارغُ آخرُ ثمانية.**
//
// **والملكيّةُ على من يناديه** — البوّابةُ تفحص `ownsMerchant` قبله.
func (s *Service) MerchantRecipients(ctx context.Context, merchantID, q string) ([]MerchantRecipient, error) {
	if r := []rune(q); len(r) > maxRecipientQuery {
		q = string(r[:maxRecipientQuery])
	}
	t := NormalizeSearch(q)
	rows, err := s.db.Query(ctx, recipientsSQL, merchantID, t.Name, EscapeLike(t.Phone),
		MaxRecipientSuggestions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MerchantRecipient{}
	for rows.Next() {
		var r MerchantRecipient
		if err := rows.Scan(&r.Phone, &r.Name, &r.AddressText, &r.Lat, &r.Lng,
			&r.LastUsedAt, &r.Times); err != nil {
			return nil, err
		}
		// **والرقمُ بصيغته الموحّدة** — والمخزَّنُ موحّدٌ أصلاً منذ أوّل توصيلة
		// (`identity.NormalizePhone` في بوّابة الإنشاء)، **وصفٌّ قديمٌ بغيرها لا يُعرض بشكلين.**
		if p, ok := RecipientPhoneKey(r.Phone); ok {
			r.Phone = p
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RecipientPhoneKey **رقمُ المستلِم بصيغته المخزَّنة** — بمطبِّع الحسابات نفسِه
// (`identity.NormalizePhone`) **بعد تحويل الأرقام الهنديّة**: «٠٩٣٣…» و«+963 933…» و«00963933…»
// رقمٌ واحد.
func RecipientPhoneKey(raw string) (string, bool) {
	return identity.NormalizePhone(foldDigits(strings.TrimSpace(raw)))
}

// HideMerchantRecipient **«شيلو من القائمة»** — يُخفى المستلِمُ حتّى يبعث له المتجرُ من جديد.
//
// **ورقمٌ لم يبعث له هذا المتجرُ قطّ لا يُكتب** (`not_found`) — فلا يصير الجدولُ دفترَ أرقامٍ
// يُملأ بنداءاتٍ عمياء. **والإخفاءُ مرّتين يجدّد وقتَه** ولا يُكرَّر.
func (s *Service) HideMerchantRecipient(ctx context.Context, merchantID, rawPhone string) error {
	phone, ok := RecipientPhoneKey(rawPhone)
	if !ok {
		return identity.ErrInvalidPhone
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO merchant_recipients_hidden (merchant_id, phone_norm)
		SELECT $1::uuid, $2::text
		 WHERE EXISTS (SELECT 1 FROM orders
		                WHERE merchant_id = $1 AND kind = 'merchant_delivery'
		                  AND recipient_phone = $2)
		ON CONFLICT (merchant_id, phone_norm) DO UPDATE SET hidden_at = now()`,
		merchantID, phone)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
