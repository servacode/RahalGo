package orders

// ══════════════════════════════════════════════════════════════════════
// **بطاقة رحّال غو وكرت اليوم** — قرارُ المالك ٢٠٢٦-١١-٢٧ (هجرة ٠٥٦٦)
// ══════════════════════════════════════════════════════════════════════
//
// **بطاقةُ رحّال غو** (`stamp`): كلُّ توصيلةٍ مسلَّمةٍ تختم دائرة (`stamp_card.size`، ١٠). **فإذا
// امتلأت فالتوصيلةُ التالية مجانيّة**، وبعد تسليمها تعود البطاقةُ فارغة.
//
// **كرتُ اليوم** (`daily`): `daily_card.size` (٥) توصيلاتٍ مسلَّمةٍ في **يوم شغل المنصّة** — من
// ساعة بدء اليوم (`platform.day_start_hour`، `account_day_start`) لإغلاق المنصّة بجدول دوامها
// (`platform.WorkDayClose`) — **فالتاليةُ في اليوم نفسِه مجانيّة**، وتسقط عند الإغلاق. ويومٌ جديدٌ
// يبدأ بكرتٍ فارغ.
//
// **للزبون بطاقتُه** (`c:<id>` — كلُّ طلبٍ مسلَّم: عاديّ ومشتريات وأمانة) **وللمتجر بطاقتُه**
// (`m:<id>` — «لدي توصيلة» مسلَّمة).
//
// # الختم
//
// يكتبه مشغّلُ القاعدة `orders_loyalty_stamp` في معاملة التسليم نفسِها، **بمفتاح الطلب**. **ولا
// يُعدّ ختمٌ إلّا وطلبُه مسلَّمٌ الآن** — فتصحيحُ التسليم والاسترجاعُ يُسقطانه بلا حذف. **وطلبُ
// الجائزة نفسُه لا يُختم** (أبسط: لا تبدأ البطاقةُ الجديدةُ بختمٍ من جائزة القديمة).
//
// # والمال — مسارُ عرض التوصيل المجانيّ نفسُه
//
// الجائزةُ إعفاءٌ في `freeDeliveryOffer` (`free_delivery.go`): **`driver_fee` يُحسب من الأجرة
// كاملةً قبله** فيقبض الكابتنُ أجرتَه كاملةً عند التسليم (`payDriver`)، **و`promo_delivery_waived`
// يحمل المعفى** و`creditTreasury` تُخرج الفرقَ من الخزينة. **ولا قيدَ جديد.** والمندوبُ عمولتُه
// كأن لا جائزة (كعرض التوصيل المجانيّ — `repPromoCostSQL`).
//
// # والأسبقيّة — إعفاءٌ واحدٌ للطلب
//
// إن صلح عرضُ التوصيل المجانيّ وجائزةُ بطاقة **يُطبَّق الأكبرُ وحدَه**، **ولا تُستهلك جائزةٌ لم
// تُطبَّق.** وعند التساوي: **العرضُ أوّلاً** (تبقى جائزةُ البطاقة لصاحبها)، ثمّ **كرتُ اليوم**
// قبل بطاقة رحّال غو (هو الذي يسقط بالإغلاق).
//
// # وعودةُ الجائزة
//
// **تُستهلك بتسليم طلبها**، **وتعود إن انتهى طلبُها بلا تسليم** (أُلغي أو رُفض أو فشل أو
// استُرجع). وكرتُ اليوم: طلبُ الجائزة أُنشئ قبل الإغلاق يبقى مجانيّاً ولو سُلّم بعده.

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// بطاقتا الولاء — كما تُكتبان في `loyalty_rewards.card`.
const (
	CardStamp = "stamp"
	CardDaily = "daily"
)

// loyaltyNow **ساعةُ البطاقتين** — دالّةٌ لتُزاح في الفحص (حدُّ اليوم وإغلاقُ المنصّة).
var loyaltyNow = time.Now

// SetLoyaltyClock **يُزيح ساعةَ البطاقتين — للفحص وحدَه**، ويردّ ما يُعيدها.
func SetLoyaltyClock(f func() time.Time) (restore func()) {
	old := loyaltyNow
	loyaltyNow = f
	return func() { loyaltyNow = old }
}

// customerCard **صاحبُ بطاقة الزبون.**
func customerCard(customerID string) string {
	if customerID == "" {
		return ""
	}
	return "c:" + customerID
}

// merchantCard **صاحبُ بطاقة المتجر** — «لدي توصيلة».
func merchantCard(merchantID string) string {
	if merchantID == "" {
		return ""
	}
	return "m:" + merchantID
}

// cardAppliesTo **أتشمل بطاقةُ هذا الصاحب هذا النوع؟** — الزبونُ لطلباته، والمتجرُ لـ«لدي توصيلة».
func cardAppliesTo(card, kind string) bool {
	if len(card) < 3 {
		return false
	}
	if card[:2] == "m:" {
		return kind == FreeKindMerchant
	}
	return kind == FreeKindStandard || kind == FreeKindPurchase || kind == FreeKindAmanah
}

// cardSettings **إعداداتُ بطاقةٍ من اللوحة.**
type cardSettings struct {
	enabled bool
	size    int
	cap     *int64
}

func (s *Service) cardSettings(ctx context.Context, card string) cardSettings {
	prefix := "stamp_card."
	if card == CardDaily {
		prefix = "daily_card."
	}
	st := cardSettings{
		enabled: s.settings.GetBool(ctx, prefix+"enabled"),
		size:    int(s.settings.GetInt(ctx, prefix+"size")),
	}
	if st.size <= 0 {
		st.enabled = false
	}
	if c := s.settings.GetInt(ctx, prefix+"max_free_fee"); c > 0 {
		st.cap = &c
	}
	return st
}

// LoyaltyDay **يومُ شغل المنصّة الجاري** — بدايتُه وإغلاقُه وبدايةُ التالي.
type LoyaltyDay struct {
	Start time.Time
	// Close **إغلاقُ المنصّة باليوم** — أو بدايةُ التالي إن لا إغلاق (٢٤ ساعة).
	Close time.Time
	Next  time.Time
	// Open **أنحن قبل الإغلاق؟** — بعده لا جائزةَ ليومٍ انتهى.
	Open bool
}

// workDayCloser **إغلاقُ المنصّة في يوم** — تقدّمه آلةُ الأوقات (`platform.Service`).
type workDayCloser interface {
	WorkDayClose(ctx context.Context, q dbtx.Querier, start, next time.Time) (time.Time, bool, error)
}

// loyaltyDay **يومُ الشغل الجاري** — بدايةُ يوم الحساب (`account_day_start`) وإغلاقُ المنصّة.
func (s *Service) loyaltyDay(ctx context.Context, q wallet.Querier) (LoyaltyDay, error) {
	now := loyaltyNow()
	var d LoyaltyDay
	if err := q.QueryRow(ctx, `
		SELECT account_day_start(account_day($1)), account_day_start(account_day($1) + 1)`, now).
		Scan(&d.Start, &d.Next); err != nil {
		return d, err
	}
	d.Close = d.Next
	if wc, ok := s.zoneHours.(workDayCloser); ok {
		if c, has, err := wc.WorkDayClose(ctx, q, d.Start, d.Next); err != nil {
			return d, err
		} else if has {
			d.Close = c
		}
	}
	d.Open = !now.Before(d.Start) && now.Before(d.Close)
	return d, nil
}

// cardCount **ما على البطاقة الآن** — الأختامُ الحيّةُ ناقصَ ما استهلكته جوائزُها المسلَّمة.
type cardCount struct {
	stamps  int  // قد يتجاوز الحجم (أختامٌ وقعت وجائزةٌ في الطريق)
	pending bool // جائزةٌ على طلبٍ حيٍّ لم يُسلَّم
}

// liveRewardSQL **طلبُ جائزةٍ لم ينتهِ** — لا مسلَّمٌ ولا منتهٍ بلا تسليم.
const liveRewardSQL = `o.status NOT IN ('delivered', 'cancelled', 'rejected', 'failed', 'refunded')`

func countCard(ctx context.Context, q wallet.Querier, owner, card string, day *LoyaltyDay) (cardCount, error) {
	var c cardCount
	var from, until, dayStart *time.Time
	if day != nil {
		from, until, dayStart = &day.Start, &day.Close, &day.Start
	}
	var stamps, used int
	if err := q.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM loyalty_stamps s JOIN orders o ON o.id = s.order_id
		    WHERE s.owner = $1 AND o.status = 'delivered'
		      AND ($2 = 'daily' OR s.for_stamp)
		      AND ($3::timestamptz IS NULL OR s.created_at >= $3)
		      AND ($4::timestamptz IS NULL OR s.created_at < $4)),
		  (SELECT COALESCE(sum(r.size) FILTER (WHERE o.status = 'delivered'), 0)
		     FROM loyalty_rewards r JOIN orders o ON o.id = r.order_id
		    WHERE r.owner = $1 AND r.card = $2
		      AND ($5::timestamptz IS NULL OR r.day_start = $5)),
		  EXISTS (SELECT 1 FROM loyalty_rewards r JOIN orders o ON o.id = r.order_id
		    WHERE r.owner = $1 AND r.card = $2
		      AND ($5::timestamptz IS NULL OR r.day_start = $5) AND `+liveRewardSQL+`)`,
		owner, card, from, until, dayStart).Scan(&stamps, &used, &c.pending); err != nil {
		return c, err
	}
	c.stamps = max(0, stamps-used)
	return c, nil
}

// cardOffer **جائزةُ بطاقةٍ صالحةٌ الآن** — صفرٌ إن لا.
type cardOffer struct {
	card     string
	size     int
	cap      *int64
	waive    int64
	dayStart *time.Time
}

// loyaltyOffer **أكبرُ جائزة بطاقةٍ يناله هذا الصاحبُ على هذه الأجرة** — كرتُ اليوم عند التساوي.
// `lock` **يقفل البطاقةَ حتّى نهاية المعاملة** — فطلبان متزامنان لا ينالان جائزةً واحدة.
func (s *Service) loyaltyOffer(ctx context.Context, q wallet.Querier, owner, kind string,
	fee int64, lock bool) (cardOffer, error) {
	if fee <= 0 || !cardAppliesTo(owner, kind) {
		return cardOffer{}, nil
	}
	stamp, daily := s.cardSettings(ctx, CardStamp), s.cardSettings(ctx, CardDaily)
	if !stamp.enabled && !daily.enabled {
		return cardOffer{}, nil
	}
	if lock {
		if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('loyalty|' || $1))`, owner); err != nil {
			return cardOffer{}, err
		}
	}
	var best cardOffer
	if daily.enabled {
		day, err := s.loyaltyDay(ctx, q)
		if err != nil {
			return cardOffer{}, err
		}
		if day.Open {
			c, err := countCard(ctx, q, owner, CardDaily, &day)
			if err != nil {
				return cardOffer{}, err
			}
			if c.stamps >= daily.size && !c.pending {
				ds := day.Start
				best = cardOffer{card: CardDaily, size: daily.size, cap: daily.cap,
					waive: waiveOf(fee, daily.cap), dayStart: &ds}
			}
		}
	}
	if stamp.enabled {
		c, err := countCard(ctx, q, owner, CardStamp, nil)
		if err != nil {
			return cardOffer{}, err
		}
		if c.stamps >= stamp.size && !c.pending {
			if w := waiveOf(fee, stamp.cap); w > best.waive {
				best = cardOffer{card: CardStamp, size: stamp.size, cap: stamp.cap, waive: w}
			}
		}
	}
	return best, nil
}

// claimCard **يُثبّت جائزةَ البطاقة على طلبها.**
func (c cardOffer) claim(ctx context.Context, q wallet.Querier, owner, orderID string) error {
	if c.waive <= 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO loyalty_rewards (order_id, owner, card, size, max_fee, waived, day_start)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, orderID, owner, c.card, c.size, c.cap, c.waive, c.dayStart)
	return err
}

// loyaltyReapply **أجرةُ طلبِ جائزةٍ تبدّلت** — يُعاد الإعفاءُ بسقف جائزته. `ok=false` إن لا جائزة.
func loyaltyReapply(ctx context.Context, q wallet.Querier, orderID string, fee int64) (int64, bool, error) {
	var maxFee *int64
	err := q.QueryRow(ctx, `SELECT max_fee FROM loyalty_rewards WHERE order_id = $1`, orderID).Scan(&maxFee)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	w := waiveOf(fee, maxFee)
	if w <= 0 {
		_, err := q.Exec(ctx, `DELETE FROM loyalty_rewards WHERE order_id = $1`, orderID)
		return 0, true, err
	}
	_, err = q.Exec(ctx, `UPDATE loyalty_rewards SET waived = $2 WHERE order_id = $1`, orderID, w)
	return w, true, err
}

// ══════════════════════════════════════════════════════════════════════
// **ما تقرؤه الشاشة**
// ══════════════════════════════════════════════════════════════════════

// LoyaltyStamp **ختمٌ على البطاقة** — رقمُ الطلب وتاريخُه.
type LoyaltyStamp struct {
	OrderID string    `json:"order_id"`
	Number  int64     `json:"number"`
	At      time.Time `json:"at"`
}

// DailyCardView **كرتُ اليوم.**
type DailyCardView struct {
	Enabled   bool  `json:"enabled"`
	Size      int   `json:"size"`
	Stamps    int   `json:"stamps"`
	FreeReady bool  `json:"free_ready"`
	FreeCap   int64 `json:"free_cap"`
	// RewardPending **الجائزةُ على طلبٍ لم يُسلَّم بعد.**
	RewardPending bool `json:"reward_pending"`
	// Open **أيومُ الشغل قائم؟** — بعد الإغلاق يُقال «يتجدّد مع بداية دوام الغد».
	Open     bool      `json:"open"`
	DayStart time.Time `json:"day_start"`
	ClosesAt time.Time `json:"closes_at"`
	RenewsAt time.Time `json:"renews_at"`
}

// StampCardView **بطاقةُ رحّال غو — وكرتُ اليوم معها** (`daily`).
type StampCardView struct {
	Enabled       bool           `json:"enabled"`
	Size          int            `json:"size"`
	Stamps        int            `json:"stamps"`
	FreeReady     bool           `json:"free_ready"`
	FreeCap       int64          `json:"free_cap"`
	RewardPending bool           `json:"reward_pending"`
	Recent        []LoyaltyStamp `json:"recent"`
	Daily         DailyCardView  `json:"daily"`
}

// CustomerLoyalty **بطاقتا الزبون.**
func (s *Service) CustomerLoyalty(ctx context.Context, customerID string) (StampCardView, error) {
	return s.loyaltyView(ctx, customerCard(customerID))
}

// MerchantLoyalty **بطاقتا المتجر** («لدي توصيلة»).
func (s *Service) MerchantLoyalty(ctx context.Context, merchantID string) (StampCardView, error) {
	return s.loyaltyView(ctx, merchantCard(merchantID))
}

func (s *Service) loyaltyView(ctx context.Context, owner string) (StampCardView, error) {
	v := StampCardView{Recent: []LoyaltyStamp{}}
	if owner == "" {
		return v, nil
	}
	stamp, daily := s.cardSettings(ctx, CardStamp), s.cardSettings(ctx, CardDaily)
	v.Enabled, v.Size = stamp.enabled, stamp.size
	if stamp.cap != nil {
		v.FreeCap = *stamp.cap
	}
	if stamp.enabled {
		c, err := countCard(ctx, s.db, owner, CardStamp, nil)
		if err != nil {
			return v, err
		}
		v.Stamps = min(c.stamps, stamp.size)
		v.RewardPending = c.pending
		v.FreeReady = c.stamps >= stamp.size && !c.pending
		if v.Stamps > 0 {
			rows, err := s.db.Query(ctx, `
				SELECT o.id::text, o.number, s.created_at
				  FROM loyalty_stamps s JOIN orders o ON o.id = s.order_id
				 WHERE s.owner = $1 AND s.for_stamp AND o.status = 'delivered'
				 ORDER BY s.created_at DESC LIMIT $2`, owner, v.Stamps)
			if err != nil {
				return v, err
			}
			for rows.Next() {
				var st LoyaltyStamp
				if err := rows.Scan(&st.OrderID, &st.Number, &st.At); err != nil {
					rows.Close()
					return v, err
				}
				v.Recent = append(v.Recent, st)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return v, err
			}
		}
	}
	v.Daily.Enabled, v.Daily.Size = daily.enabled, daily.size
	if daily.cap != nil {
		v.Daily.FreeCap = *daily.cap
	}
	if daily.enabled {
		day, err := s.loyaltyDay(ctx, s.db)
		if err != nil {
			return v, err
		}
		v.Daily.Open, v.Daily.DayStart, v.Daily.ClosesAt, v.Daily.RenewsAt = day.Open, day.Start, day.Close, day.Next
		if day.Open {
			c, err := countCard(ctx, s.db, owner, CardDaily, &day)
			if err != nil {
				return v, err
			}
			v.Daily.Stamps = min(c.stamps, daily.size)
			v.Daily.RewardPending = c.pending
			v.Daily.FreeReady = c.stamps >= daily.size && !c.pending
		}
	}
	return v, nil
}
