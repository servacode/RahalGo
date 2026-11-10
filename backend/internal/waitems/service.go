// Package waitems **إضافة أصناف بصورة على واتساب** (قرارُ المالك 2026-11-09، المرحلة 1).
//
// صاحبُ المتجر أو موظّفه يبعت صورةً وفي التعليق «اسم وسعر» لرقم رحّال غو، فينضاف الصنفُ
// ويظهر للزبائن فوراً — كالإضافة من التطبيق تماماً (`catalog.CreateItemIn` نفسُه؛ لا مسارَ
// إدراجٍ ثانياً. ومفتاحُ «موافقة الأصناف» رُفع بقرار 2026-10-04 فلا مراجعةَ قبل النشر).
//
// **ولا تخمين**: لا سعرَ ⇒ «شو سعر (الاسم)؟»، ولا اسمَ ⇒ «شو اسم الصنف؟»، واسمٌ مكرّرٌ ⇒ يسأل،
// وسعرٌ أبعدُ بكثيرٍ من أصناف القسم ⇒ «متأكد؟». **والقسمُ يُسأل عنه** (قرار المالك 2026-11-09): قائمةُ أقسام المتجر مرقّمةً
// والمرسل يردّ برقم (أو اسم)؛ قسمٌ واحد ⇒ بلا سؤال؛ متجرٌ بلا أقسام فعّالة ⇒ قسم الانتظار القديم. والأسئلةُ تُحفظ بالخادم بمهلة (`wa_item_pending`).
// **والبوتُ لا يبدأ محادثةً أبداً**: يردّ فقط على من راسله، ومن ليس صاحبَ متجرٍ ولا موظّفاً يُهمَل بصمت.
package waitems

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notify"
)

// الثوابت.
const (
	// ViaWhatsApp قيمة `menu_items.created_via` لصنفٍ وُلد من هنا.
	ViaWhatsApp = "whatsapp"
	// sanityFactor ما فوقه (× وسيط القسم) يُسأل عنه.
	sanityFactor = 10.0
	// sanityMinItems أقلّ أصناف في القسم ليُعتدّ بوسيطها.
	sanityMinItems = 3
)

// Pending kinds.
const (
	kStore     = "store"
	kStoreWait = "store_wait"
	kName      = "name"
	kPrice     = "price"
	kDup       = "dup"
	kSanity    = "sanity"
	kSection   = "section"
)

// maxSectionList أقصى أقسام تظهر مرقّمةً بالرسالة (الباقي يُختار بالاسم) — كي تبقى القائمة مقروءةً بالواتساب.
const maxSectionList = 25

// Service خدمةُ أصناف الواتساب.
type Service struct {
	db    *pgxpool.Pool
	cat   *catalog.Service
	media *media.Service
	log   *slog.Logger

	// Send يرسل ردّاً مؤجَّلاً (ملخّص الدفعة). فارغ ⇒ لا يُرسَل.
	Send func(ctx context.Context, phone, text string) error
	// SiteURL أصل الموقع لرابط الصنف في ردّ التأكيد.
	SiteURL func() string
	// NameMax حدّ طول اسم الصنف (فارغ ⇒ حدّ الكتالوج).
	NameMax func(ctx context.Context) int
	// OnChange يُنادى بعد إضافة/تحديث صنف (بثّ للّوحات).
	OnChange func()

	// Quiet مهلة الهدوء قبل إرسال ملخّص دفعة الصور.
	Quiet time.Duration
	// PendingTTL عمر السؤال المعلّق.
	PendingTTL time.Duration
	// RatePerMin أقصى أصناف للمتجر في الدقيقة.
	RatePerMin int

	phoneMu sync.Mutex
	phones  map[string]*sync.Mutex

	bmu     sync.Mutex
	batches map[string]*batch
}

type batch struct {
	outs  []outcome
	timer *time.Timer
}

// New خدمةٌ بالقيم الافتراضيّة (هدوءُ ٦ ثوانٍ، مهلةُ سؤالٍ ٣٠ دقيقة، ٣٠ صنفاً/دقيقة).
func New(db *pgxpool.Pool, cat *catalog.Service, md *media.Service, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{db: db, cat: cat, media: md, log: log,
		Quiet: 6 * time.Second, PendingTTL: 30 * time.Minute, RatePerMin: 30,
		phones: map[string]*sync.Mutex{}, batches: map[string]*batch{}}
}

// ── من هذا الرقم؟ ─────────────────────────────────────────────────────

type storeInfo struct {
	ID, Name, Currency string
	Rate               float64
	ActorID            string
	StaffID, StaffName string
	Suspended          bool
}

type who struct {
	Phone  string
	Stores []storeInfo
}

// lookup **متاجرُ هذا الرقم** (صاحباً أو موظّفاً) — nil إن كان غريباً.
func (s *Service) lookup(ctx context.Context, phone string) (*who, error) {
	rows, err := s.db.Query(ctx, `
		SELECT m.id::text, m.name, m.display_currency, COALESCE(m.exchange_rate::float8, 0),
		       m.owner_user_id::text, '', '', (m.status = 'suspended'), m.created_at, 0
		  FROM users u JOIN merchants m ON m.owner_user_id = u.id
		 WHERE u.phone = $1
		UNION ALL
		SELECT m.id::text, m.name, m.display_currency, COALESCE(m.exchange_rate::float8, 0),
		       m.owner_user_id::text, st.id::text, st.name, (m.status = 'suspended'), m.created_at, 1
		  FROM merchant_staff st JOIN merchants m ON m.owner_user_id = st.owner_user_id
		 WHERE st.phone = $1
		ORDER BY 10, 9, 1`, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	w := &who{Phone: phone}
	seen := map[string]bool{}
	for rows.Next() {
		var st storeInfo
		var ts time.Time
		var ord int
		if err := rows.Scan(&st.ID, &st.Name, &st.Currency, &st.Rate, &st.ActorID, &st.StaffID, &st.StaffName, &st.Suspended, &ts, &ord); err != nil {
			return nil, err
		}
		if seen[st.ID] {
			continue
		}
		seen[st.ID] = true
		w.Stores = append(w.Stores, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(w.Stores) == 0 {
		return nil, nil
	}
	return w, nil
}

func (w *who) active() []storeInfo {
	var out []storeInfo
	for _, s := range w.Stores {
		if !s.Suspended {
			out = append(out, s)
		}
	}
	return out
}

func (w *who) byID(id string) (storeInfo, bool) {
	for _, s := range w.Stores {
		if s.ID == id {
			return s, true
		}
	}
	return storeInfo{}, false
}

// ── المسودّة والأسئلة المعلّقة ──────────────────────────────────────

// Draft ما نعرفه عن صنفٍ قيد الإنجاز — يُخزَّن مع السؤال المعلّق.
type Draft struct {
	MerchantID string   `json:"m"`
	MediaID    string   `json:"media,omitempty"`
	Caption    string   `json:"cap,omitempty"`
	Name       string   `json:"name,omitempty"`
	Price      float64  `json:"price,omitempty"`
	HasPrice   bool     `json:"has,omitempty"`
	Currency   string   `json:"cur,omitempty"`
	Dup        string   `json:"dup,omitempty"` // "" | update | new
	DupItemID  string   `json:"dup_id,omitempty"`
	SanityOK   bool     `json:"ok,omitempty"`
	Stores     []string `json:"stores,omitempty"`
	// القسم: معرّفه واسمه ومصدره (manual = اختاره المرسل، name/pending = احتياط متجر بلا أقسام).
	Section     string   `json:"sec,omitempty"`
	SectionName string   `json:"secn,omitempty"`
	SectionSrc  string   `json:"ssrc,omitempty"`
	OptIDs      []string `json:"opt,omitempty"`  // أقسام السؤال المعلّق بترتيب القائمة
	OptNames    []string `json:"optn,omitempty"` // وأسماؤها
}

type pending struct {
	ID    string
	MsgID string
	Kind  string
	D     Draft
}

type outcome struct {
	Kind string
	Text string
	List string // قائمة الأقسام (لملخّص الدفعة)
}

const (
	oCreated = "created"
	oUpdated = "updated"
	oPrice   = "ask_price"
	oName    = "ask_name"
	oDup     = "ask_dup"
	oSanity  = "ask_sanity"
	oStore   = "ask_store"
	oSection = "ask_section"
	oRate    = "rate"
	oFailed  = "failed"
	oInfo    = "info"
)

func (s *Service) lockPhone(phone string) func() {
	s.phoneMu.Lock()
	m := s.phones[phone]
	if m == nil {
		m = &sync.Mutex{}
		s.phones[phone] = m
	}
	s.phoneMu.Unlock()
	m.Lock()
	return m.Unlock
}

func (s *Service) loadPending(ctx context.Context, phone string) ([]pending, error) {
	_, _ = s.db.Exec(ctx, `DELETE FROM wa_item_pending WHERE expires_at < now()`)
	rows, err := s.db.Query(ctx, `
		SELECT id::text, msg_id, kind, data FROM wa_item_pending
		 WHERE phone = $1 AND expires_at > now() ORDER BY created_at, id`, phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pending
	for rows.Next() {
		var p pending
		var raw []byte
		if err := rows.Scan(&p.ID, &p.MsgID, &p.Kind, &raw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &p.D)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) putPending(ctx context.Context, phone, msgID, kind string, d Draft) error {
	raw, _ := json.Marshal(d)
	_, err := s.db.Exec(ctx, `
		INSERT INTO wa_item_pending (phone, msg_id, kind, data, expires_at)
		VALUES ($1, $2, $3, $4, now() + $5::interval)`,
		phone, msgID, kind, raw, fmt.Sprintf("%d seconds", int(s.PendingTTL.Seconds())))
	return err
}

func (s *Service) dropPending(ctx context.Context, id string) {
	_, _ = s.db.Exec(ctx, `DELETE FROM wa_item_pending WHERE id = $1::uuid`, id)
}

// ── نقطة الدخول: صورة ─────────────────────────────────────────────

// HandleImage **صورةٌ واردة** — يردّ نصّاً فورياً أو فارغاً (والرسائلُ الأخرى تُجمَّع وتُرسَل بعد هدوء).
func (s *Service) HandleImage(ctx context.Context, from string, img notify.InboundImage) string {
	phone, ok := identity.NormalizePhone(from)
	if !ok {
		return ""
	}
	w, err := s.lookup(ctx, phone)
	if err != nil {
		s.log.Warn("waitems: lookup", "error", err)
		return ""
	}
	if w == nil {
		return "" // غريب: صمت
	}
	// واتساب قد يعيد التسليم — رسالةٌ عولجت لا تُعالَج ثانية.
	if img.MsgID != "" {
		tag, err := s.db.Exec(ctx, `INSERT INTO wa_item_msgs (msg_id) VALUES ($1) ON CONFLICT DO NOTHING`, img.MsgID)
		if err == nil && tag.RowsAffected() == 0 {
			return ""
		}
		_, _ = s.db.Exec(ctx, `DELETE FROM wa_item_msgs WHERE created_at < now() - interval '2 days'`)
	}
	if img.NotImage || len(img.Data) == 0 {
		return "ابعت صورة 📷"
	}
	act := w.active()
	if len(act) == 0 {
		return "متجرك موقوف حالياً، تواصل مع رحّال غو."
	}

	unlock := s.lockPhone(phone)
	defer unlock()

	st, needAsk, err := s.chooseStore(ctx, phone, act)
	if err != nil {
		s.log.Warn("waitems: choose store", "error", err)
		return ""
	}
	if st.ID != "" && s.overLimit(ctx, st.ID) {
		s.enqueue(phone, outcome{Kind: oRate, Text: "شوي شوي 🙏 أضفت أصناف كتير بدقيقة، ابعت الباقي بعد دقيقة."})
		return ""
	}
	actor := act[0].ActorID
	if st.ID != "" {
		actor = st.ActorID
	}
	m, err := s.media.Save(ctx, actor, "menu_item", bytes.NewReader(img.Data))
	if err != nil {
		s.enqueue(phone, outcome{Kind: oFailed, Text: "ما قدرت أقرأ الصورة، ابعتها مرة تانية 🙏"})
		return ""
	}
	if needAsk {
		d := Draft{MediaID: m.ID, Caption: img.Caption}
		o, err := s.askStore(ctx, phone, img.MsgID, d, act)
		if err != nil {
			s.log.Warn("waitems: ask store", "error", err)
			return ""
		}
		if o.Text != "" {
			s.enqueue(phone, o)
		}
		return ""
	}
	d := s.draftFromCaption(st, m.ID, img.Caption)
	s.enqueue(phone, s.advance(ctx, st, phone, img.MsgID, d))
	return ""
}

func (s *Service) draftFromCaption(st storeInfo, mediaID, caption string) Draft {
	p := ParseCaption(caption)
	return Draft{MerchantID: st.ID, MediaID: mediaID, Caption: caption, Name: p.Name,
		Price: p.Price, HasPrice: p.HasPrice, Currency: p.Currency}
}

// chooseStore **فرعُ هذا الرقم**: متجرٌ واحد ⇒ هو، وإلا المحفوظ، وإلا يُسأل.
func (s *Service) chooseStore(ctx context.Context, phone string, act []storeInfo) (storeInfo, bool, error) {
	if len(act) == 1 {
		return act[0], false, nil
	}
	var mid string
	err := s.db.QueryRow(ctx, `SELECT merchant_id::text FROM wa_item_store_choice WHERE phone = $1`, phone).Scan(&mid)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return storeInfo{}, false, err
	}
	for _, a := range act {
		if a.ID == mid {
			return a, false, nil
		}
	}
	return storeInfo{}, true, nil
}

func (s *Service) askStore(ctx context.Context, phone, msgID string, d Draft, act []storeInfo) (outcome, error) {
	d.Stores = nil
	for _, a := range act {
		d.Stores = append(d.Stores, a.ID)
	}
	raw, _ := json.Marshal(d)
	ttl := fmt.Sprintf("%d seconds", int(s.PendingTTL.Seconds()))
	tag, err := s.db.Exec(ctx, `
		INSERT INTO wa_item_pending (phone, msg_id, kind, data, expires_at)
		VALUES ($1, $2, 'store', $3, now() + $4::interval) ON CONFLICT DO NOTHING`, phone, msgID, raw, ttl)
	if err != nil {
		return outcome{}, err
	}
	if tag.RowsAffected() == 0 {
		// سؤال الفرع قائم — هذه تنتظر خلفه.
		_, err = s.db.Exec(ctx, `
			INSERT INTO wa_item_pending (phone, msg_id, kind, data, expires_at)
			VALUES ($1, $2, 'store_wait', $3, now() + $4::interval)`, phone, msgID, raw, ttl)
		return outcome{}, err
	}
	return outcome{Kind: oStore, Text: storeQuestion(act)}, nil
}

func storeQuestion(act []storeInfo) string {
	var b strings.Builder
	b.WriteString("لأي فرع؟\n")
	for i, a := range act {
		fmt.Fprintf(&b, "%s - %s\n", ar(i+1), a.Name)
	}
	b.WriteString("ردّ برقم الفرع")
	return b.String()
}

// ── نقطة الدخول: نصّ ──────────────────────────────────────────────

// HandleText **نصٌّ واردٌ يُجيب عن سؤالٍ معلّق** (أو أمر «فرع» / «إلغاء») — (ردّ، عولج).
// وغير المعالَج يكمل إلى معالِج النصوص العامّ.
func (s *Service) HandleText(ctx context.Context, from, text, quotedID string) (string, bool) {
	phone, ok := identity.NormalizePhone(from)
	if !ok {
		return "", false
	}
	txt := strings.TrimSpace(ToASCIIDigits(text))
	isBranch := isBranchWord(txt)
	unlock := s.lockPhone(phone)
	defer unlock()

	pend, err := s.loadPending(ctx, phone)
	if err != nil {
		s.log.Warn("waitems: load pending", "error", err)
		return "", false
	}
	if len(pend) == 0 && !isBranch {
		return "", false
	}
	w, err := s.lookup(ctx, phone)
	if err != nil {
		return "", false
	}
	if w == nil {
		_, _ = s.db.Exec(ctx, `DELETE FROM wa_item_pending WHERE phone = $1`, phone)
		return "", false
	}
	act := w.active()
	if len(act) == 0 {
		return "", false
	}

	if isBranch && len(pend) == 0 || isBranch && !hasKind(pend, kStore) {
		if len(act) == 1 {
			return "عندك فرع واحد: " + act[0].Name, true
		}
		_, _ = s.db.Exec(ctx, `DELETE FROM wa_item_store_choice WHERE phone = $1`, phone)
		o, err := s.askStore(ctx, phone, "", Draft{}, act)
		if err != nil {
			return "", false
		}
		if o.Text == "" {
			o.Text = storeQuestion(act)
		}
		return o.Text, true
	}
	if isCancel(txt) && len(pend) > 0 {
		_, _ = s.db.Exec(ctx, `DELETE FROM wa_item_pending WHERE phone = $1`, phone)
		return "تمام، لغيت كل شي معلّق 👍", true
	}

	target := pickTarget(pend, quotedID)
	if target == nil {
		return "عندك أكتر من صنف بانتظار ردّك — ردّ على صورة الصنف نفسها (اضغط عليها ← ردّ) أو اكتب «الغاء».", true
	}
	var outs []outcome
	switch target.Kind {
	case kStore:
		outs = s.answerStore(ctx, w, act, phone, pend, target, txt)
	default:
		st, ok := w.byID(target.D.MerchantID)
		if !ok || st.Suspended {
			s.dropPending(ctx, target.ID)
			return "", false
		}
		outs = []outcome{s.answerItem(ctx, st, phone, target, txt)}
	}
	return summarize(outs), true
}

func hasKind(p []pending, kind string) bool {
	for _, x := range p {
		if x.Kind == kind {
			return true
		}
	}
	return false
}

// pickTarget **أيُّ سؤالٍ يُجيبه هذا النصّ**: المردودُ عليه بالاقتباس، وإلا سؤالُ الفرع، وإلا الوحيد.
func pickTarget(p []pending, quotedID string) *pending {
	if quotedID != "" {
		for i := range p {
			if p[i].MsgID == quotedID && p[i].Kind != kStoreWait {
				return &p[i]
			}
			if p[i].MsgID == quotedID && p[i].Kind == kStoreWait {
				for j := range p {
					if p[j].Kind == kStore {
						return &p[j]
					}
				}
			}
		}
	}
	for i := range p {
		if p[i].Kind == kStore {
			return &p[i]
		}
	}
	var ask []int
	for i := range p {
		if p[i].Kind != kStoreWait {
			ask = append(ask, i)
		}
	}
	if len(ask) == 1 {
		return &p[ask[0]]
	}
	return nil
}

func (s *Service) answerStore(ctx context.Context, w *who, act []storeInfo, phone string, all []pending, target *pending, txt string) []outcome {
	opts := target.D.Stores
	var chosen storeInfo
	found := false
	if n, ok := pickNumber(txt); ok && n >= 1 && n <= len(opts) {
		chosen, found = w.byID(opts[n-1])
	} else {
		for _, id := range opts {
			if a, ok := w.byID(id); ok && txt != "" && strings.Contains(catalog.NormName(a.Name), catalog.NormName(txt)) {
				if found {
					found = false
					break
				}
				chosen, found = a, true
			}
		}
	}
	if !found || chosen.Suspended {
		return []outcome{{Kind: oStore, Text: "ما فهمت، " + storeQuestion(act)}}
	}
	_, _ = s.db.Exec(ctx, `
		INSERT INTO wa_item_store_choice (phone, merchant_id) VALUES ($1, $2::uuid)
		ON CONFLICT (phone) DO UPDATE SET merchant_id = EXCLUDED.merchant_id, updated_at = now()`, phone, chosen.ID)
	var outs []outcome
	waiting := 0
	for _, p := range all {
		if p.Kind != kStore && p.Kind != kStoreWait {
			continue
		}
		s.dropPending(ctx, p.ID)
		if p.D.MediaID == "" {
			continue
		}
		waiting++
		d := s.draftFromCaption(chosen, p.D.MediaID, p.D.Caption)
		outs = append(outs, s.advance(ctx, chosen, phone, p.MsgID, d))
	}
	if waiting == 0 {
		return []outcome{{Kind: oInfo, Text: "تمام، صار الفرع: " + chosen.Name + " ✓"}}
	}
	return outs
}

// answerItem **جوابٌ على سؤال صنف**: اسم / سعر / مكرّر / تأكيد.
func (s *Service) answerItem(ctx context.Context, st storeInfo, phone string, p *pending, txt string) outcome {
	d := p.D
	switch p.Kind {
	case kName:
		pr := ParseCaption(txt)
		name := pr.Name
		if name == "" && !pr.HasPrice {
			name = cleanName(txt)
		}
		if name == "" {
			return s.requeue(ctx, phone, p, "اكتب اسم الصنف بالحروف 🙏 شو اسمه؟")
		}
		d.Name = name
		if pr.HasPrice && !d.HasPrice {
			d.Price, d.HasPrice = pr.Price, true
			if d.Currency == "" {
				d.Currency = pr.Currency
			}
		}
	case kPrice:
		pr := ParseCaption(txt)
		if !pr.HasPrice {
			return s.requeue(ctx, phone, p, "ما فهمت السعر 🙏 اكتبه برقم، مثلاً ٢٥٠٠٠ أو ٢٥ ألف أو 5$")
		}
		d.Price, d.HasPrice = pr.Price, true
		if pr.Currency != "" {
			d.Currency = pr.Currency
		}
		d.SanityOK = false
	case kDup:
		n, _ := pickNumber(txt)
		switch {
		case n == 1:
			d.Dup = "update"
		case n == 2:
			d.Dup = "new"
		default:
			return s.requeue(ctx, phone, p, "ردّ ١ لتحديث الصنف الموجود أو ٢ لإضافته صنف جديد")
		}
	case kSection:
		i, ok := pickSection(txt, d.OptNames)
		if !ok {
			shown := min(len(d.OptNames), maxSectionList)
			return s.requeue(ctx, phone, p, fmt.Sprintf("ما فهمت 🙏 ردّ برقم القسم من ١ لـ %s", ar(shown)))
		}
		d.Section, d.SectionName, d.SectionSrc = d.OptIDs[i], d.OptNames[i], catalog.SourceManual
	case kSanity:
		switch {
		case isYes(txt):
			d.SanityOK = true
		case isNo(txt):
			d.HasPrice, d.SanityOK = false, false
		default:
			return s.requeue(ctx, phone, p, "ردّ «نعم» لتأكيد السعر أو «لا» لتغييره")
		}
	}
	s.dropPending(ctx, p.ID)
	return s.advance(ctx, st, phone, p.MsgID, d)
}

// requeue يُبقي السؤال كما هو ويعيد صياغته.
func (s *Service) requeue(_ context.Context, _ string, _ *pending, text string) outcome {
	return outcome{Kind: oInfo, Text: text}
}

// ── المحرّك: يتقدّم بالمسودّة خطوةً خطوة ─────────────────────────────

func (s *Service) ask(ctx context.Context, phone, msgID, kind string, d Draft, askKind, text string) outcome {
	if err := s.putPending(ctx, phone, msgID, kind, d); err != nil {
		s.log.Warn("waitems: put pending", "error", err)
		return outcome{Kind: oFailed, Text: "صار خطأ، جرّب مرة تانية 🙏"}
	}
	return outcome{Kind: askKind, Text: text}
}

func (s *Service) nameMax(ctx context.Context) int {
	if s.NameMax != nil {
		if n := s.NameMax(ctx); n > 0 && n <= catalog.MaxItemName {
			return n
		}
	}
	return catalog.MaxItemName
}

// advance **يكمل المسودّةَ حتى الحفظ أو سؤالٍ جديد.**
func (s *Service) advance(ctx context.Context, st storeInfo, phone, msgID string, d Draft) outcome {
	d.MerchantID = st.ID
	d.Name = cleanName(d.Name)
	if d.Name == "" {
		return s.ask(ctx, phone, msgID, kName, d, oName, "شو اسم الصنف؟")
	}
	if n := len([]rune(d.Name)); n > s.nameMax(ctx) {
		d.Name = ""
		return s.ask(ctx, phone, msgID, kName, d, oName,
			fmt.Sprintf("اسم الصنف طويل (أكتر من %s حرف)، اكتب اسم أقصر.", ar(s.nameMax(ctx))))
	}
	if !d.HasPrice || d.Price <= 0 {
		d.HasPrice = false
		return s.ask(ctx, phone, msgID, kPrice, d, oPrice, "شو سعر ("+d.Name+")؟")
	}
	if d.Currency == "" {
		d.Currency = st.Currency
	}
	if d.Currency == catalog.CurrencyUSD && st.Rate <= 0 {
		d.Currency, d.HasPrice = catalog.CurrencySYP, false
		return s.ask(ctx, phone, msgID, kPrice, d, oPrice,
			"ما في سعر صرف لمتجرك فما فيني أحسب الدولار — اكتب سعر ("+d.Name+") بالليرة.")
	}
	sypPrice := d.Price
	if d.Currency == catalog.CurrencyUSD {
		sypPrice = d.Price * st.Rate
	}
	if d.Dup == "" {
		id, err := s.findDup(ctx, st.ID, d.Name)
		if err != nil {
			s.log.Warn("waitems: dup", "error", err)
		}
		if id != "" {
			d.DupItemID = id
			return s.ask(ctx, phone, msgID, kDup, d, oDup,
				"في صنف بنفس الاسم ("+d.Name+"): ١ حدّث صورته وسعره · ٢ أضفه صنف جديد")
		}
		d.Dup = "new"
	}
	// القسم: يسأل المرسلَ عنه (قرار المالك 2026-11-09) — إلا تحديثَ صنفٍ موجود (قسمه قائم).
	secID, secName, src := d.Section, d.SectionName, d.SectionSrc
	if d.Dup == "update" {
		secID, secName, src = s.itemSection(ctx, d.DupItemID)
	} else if secID == "" {
		opts, err := s.sectionOptions(ctx, st.ID)
		if err != nil {
			s.log.Warn("waitems: section options", "error", err)
			return outcome{Kind: oFailed, Text: "صار خطأ، جرّب مرة تانية 🙏"}
		}
		switch len(opts) {
		case 0:
			// متجرٌ بلا أقسام فعّالة: الاحتياط القديم — مقترح الاسم، وإلا قسم الانتظار (الإدارة ترتّبه).
			secID, secName, src, err = catalog.SectionByName(ctx, s.db, d.Name)
			if err != nil {
				s.log.Warn("waitems: section", "error", err)
				return outcome{Kind: oFailed, Text: "صار خطأ، جرّب مرة تانية 🙏"}
			}
		case 1:
			secID, secName, src = opts[0].ID, opts[0].Name, catalog.SourceManual
		default:
			d.OptIDs, d.OptNames = nil, nil
			for _, o := range opts {
				d.OptIDs, d.OptNames = append(d.OptIDs, o.ID), append(d.OptNames, o.Name)
			}
			head, list, foot := sectionQuestion(d.Name, d.OptNames)
			o := s.ask(ctx, phone, msgID, kSection, d, oSection, head+list+foot)
			o.List = list + foot
			return o
		}
		d.Section, d.SectionName, d.SectionSrc = secID, secName, src
	}
	if !d.SanityOK && secID != "" && src != catalog.SourcePending {
		if n, med := s.sectionMedian(ctx, st.ID, secID); n >= sanityMinItems && med > 0 && sypPrice > sanityFactor*med {
			return s.ask(ctx, phone, msgID, kSanity, d, oSanity, fmt.Sprintf(
				"متأكد؟ سعر (%s) %s وأصناف قسم %s عادةً حوالي %s — ردّ «نعم» للتأكيد أو «لا» لتغييره",
				d.Name, money(d.Price, d.Currency), secName, money(med, catalog.CurrencySYP)))
		}
	}
	if d.Dup == "new" && s.overLimit(ctx, st.ID) {
		return outcome{Kind: oRate, Text: "شوي شوي 🙏 أضفت أصناف كتير بدقيقة، ابعت الباقي بعد دقيقة."}
	}
	return s.save(ctx, st, phone, d, secID, secName, src)
}

type secOpt struct{ ID, Name string }

// sectionOptions **أقسامُ هذا المتجر** (ما أعلنه، وفارغٌ يعني كلَّ أقسام السوق الفعّالة) — بترتيب السوق.
func (s *Service) sectionOptions(ctx context.Context, merchantID string) ([]secOpt, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ps.id::text, ps.name FROM platform_sections ps
		 WHERE ps.active
		   AND (NOT EXISTS (SELECT 1 FROM store_sections x WHERE x.store_id = $1::uuid)
		        OR ps.id IN (SELECT section_id FROM store_sections WHERE store_id = $1::uuid))
		 ORDER BY ps.sort_order, ps.name, ps.id`, merchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []secOpt
	for rows.Next() {
		var o secOpt
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// itemSection قسمُ صنفٍ موجود (لتحديثه) — فارغٌ إن لم يُعرف.
func (s *Service) itemSection(ctx context.Context, itemID string) (id, name, src string) {
	_ = s.db.QueryRow(ctx, `SELECT COALESCE(m.platform_section_id::text, ''), COALESCE(ps.name, ''), m.section_source
		  FROM menu_items m LEFT JOIN platform_sections ps ON ps.id = m.platform_section_id
		 WHERE m.id = $1::uuid`, itemID).Scan(&id, &name, &src)
	return id, name, src
}

// sectionQuestion نصُّ السؤال: (سطر العنوان، القائمة المرقّمة، التذييل). فوق maxSectionList يُعرض الأوّل والباقي بالاسم.
func sectionQuestion(item string, names []string) (head, list, foot string) {
	head = "بأي قسم أحط (" + item + ")؟\n"
	var b strings.Builder
	shown := min(len(names), maxSectionList)
	for i := 0; i < shown; i++ {
		fmt.Fprintf(&b, "%s - %s\n", ar(i+1), names[i])
	}
	foot = "ردّ برقم القسم"
	if len(names) > shown {
		foot += "، أو اكتب اسم القسم إذا مو بالقائمة"
	}
	return head, b.String(), foot
}

// pickSection **القسم المختار**: رقمٌ من القائمة المعروضة، أو اسمٌ (مطابقةٌ كاملة ثم جزئيّةٌ وحيدة).
func pickSection(txt string, names []string) (int, bool) {
	if n, ok := pickNumber(txt); ok {
		if n >= 1 && n <= min(len(names), maxSectionList) {
			return n - 1, true
		}
		return 0, false
	}
	q := catalog.NormName(txt)
	if q == "" {
		return 0, false
	}
	hit := -1
	for i, n := range names {
		if catalog.NormName(n) == q {
			return i, true
		}
	}
	for i, n := range names {
		if strings.Contains(catalog.NormName(n), q) {
			if hit >= 0 {
				return 0, false // غامض
			}
			hit = i
		}
	}
	return hit, hit >= 0
}

func (s *Service) findDup(ctx context.Context, merchantID, name string) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		SELECT id::text FROM menu_items
		 WHERE merchant_id = $1::uuid
		   AND btrim(regexp_replace(translate(lower(name), 'أإآةىـ', 'اااهي'), '\s+', ' ', 'g')) = $2
		 ORDER BY created_at LIMIT 1`, merchantID, catalog.NormName(name)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *Service) sectionMedian(ctx context.Context, merchantID, sectionID string) (int, float64) {
	var n int
	var med float64
	if err := s.db.QueryRow(ctx, `
		SELECT count(*), COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY merchant_price), 0)::float8
		  FROM menu_items
		 WHERE merchant_id = $1::uuid AND platform_section_id = $2::uuid AND merchant_price > 0`,
		merchantID, sectionID).Scan(&n, &med); err != nil {
		return 0, 0
	}
	return n, med
}

func (s *Service) overLimit(ctx context.Context, merchantID string) bool {
	if s.RatePerMin <= 0 {
		return false
	}
	var n int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM menu_items
		 WHERE merchant_id = $1::uuid AND created_via = $2 AND created_at > now() - interval '1 minute'`,
		merchantID, ViaWhatsApp).Scan(&n); err != nil {
		return false
	}
	return n >= s.RatePerMin
}

// save **يحفظ الصنفَ (جديداً أو تحديثاً للمكرّر)** بالمسار نفسه الذي يستعمله التطبيق.
func (s *Service) save(ctx context.Context, st storeInfo, phone string, d Draft, secID, secName, src string) outcome {
	usd := d.Currency == catalog.CurrencyUSD
	in := catalog.MenuItemInput{ImageMediaID: &d.MediaID}
	if usd {
		cur := catalog.CurrencyUSD
		p := math.Round(d.Price*100) / 100
		in.PriceCurrency, in.PriceUSD = &cur, &p
	} else {
		cur := catalog.CurrencySYP
		p := int64(math.Round(d.Price))
		in.PriceCurrency, in.Price = &cur, &p
	}
	detail := map[string]any{"via": ViaWhatsApp, "phone": phone, "merchant_id": st.ID}
	if st.StaffID != "" {
		detail["staff_id"], detail["staff_name"] = st.StaffID, st.StaffName
	}

	if d.Dup == "update" && d.DupItemID != "" {
		if err := s.cat.UpdateItem(ctx, st.ActorID, d.DupItemID, in, ""); err != nil {
			return s.fail(d.Name, err)
		}
		s.auditRow(ctx, st.ActorID, "menu.item_wa_update", d.DupItemID, detail)
		s.changed()
		return outcome{Kind: oUpdated, Text: fmt.Sprintf("حدّثت صورة وسعر (%s) → %s ✓\n%s",
			d.Name, money(d.Price, d.Currency), s.itemLink(d.DupItemID))}
	}

	// قسمُ الانتظار غيرُ فعّال، و`CreateItemIn` لا يقبل إلا قسماً فعّالاً — فيُدرَج بقسمٍ فعّالٍ ثم يُنقل في
	// المعاملة نفسها (الصنف يبقى مخفياً حتى يرتّبه المكتب، كأصناف الأمين).
	carrier := secID
	if src == catalog.SourcePending {
		if err := s.db.QueryRow(ctx, `SELECT id::text FROM platform_sections WHERE active ORDER BY sort_order, id LIMIT 1`).Scan(&carrier); err != nil {
			return s.fail(d.Name, err)
		}
	}
	in.Name, in.PlatformSectionID = &d.Name, &carrier
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return s.fail(d.Name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := s.cat.CreateItemIn(ctx, tx, st.ID, in)
	if err != nil {
		return s.fail(d.Name, err)
	}
	if _, err := tx.Exec(ctx, `UPDATE menu_items SET created_via = $2, section_source = $3, platform_section_id = $4::uuid WHERE id = $1::uuid`,
		id, ViaWhatsApp, src, secID); err != nil {
		return s.fail(d.Name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return s.fail(d.Name, err)
	}
	s.auditRow(ctx, st.ActorID, "menu.item_create", id, detail)
	s.changed()
	text := fmt.Sprintf("انضاف (%s) بسعر %s بقسم %s ✓", d.Name, money(d.Price, d.Currency), secName)
	if src == catalog.SourcePending {
		text = fmt.Sprintf("انضاف (%s) بسعر %s ✓ — القسم لسا بانتظار التصنيف، الإدارة بترتبه وبعدها بيظهر للزبائن.", d.Name, money(d.Price, d.Currency))
	}
	return outcome{Kind: oCreated, Text: text + "\n" + s.itemLink(id)}
}

func (s *Service) fail(name string, err error) outcome {
	var ae *httpx.AppError
	if errors.As(err, &ae) && ae.Code == "item_price_too_high" {
		return outcome{Kind: oFailed, Text: "سعر (" + name + ") أكبر من المسموح، تأكد منه وابعت الصورة من جديد."}
	}
	if errors.As(err, &ae) && ae.Status == http.StatusBadRequest {
		return outcome{Kind: oFailed, Text: "ما انضاف (" + name + ") — تأكد من الاسم والسعر وابعت من جديد."}
	}
	s.log.Warn("waitems: save", "error", err)
	return outcome{Kind: oFailed, Text: "صار خطأ وما انضاف (" + name + ")، جرّب من التطبيق أو ابعته مرة تانية."}
}

func (s *Service) auditRow(ctx context.Context, actorID, action, itemID string, detail map[string]any) {
	raw, _ := json.Marshal(detail)
	_, _ = s.db.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, details)
		VALUES ($1, $2, 'menu_item', $3, $4)`, actorID, action, itemID, raw)
}

func (s *Service) changed() {
	if s.OnChange != nil {
		s.OnChange()
	}
}

func (s *Service) itemLink(id string) string {
	base := ""
	if s.SiteURL != nil {
		base = strings.TrimRight(s.SiteURL(), "/")
	}
	return base + "/store/menu/" + id
}

// ── تجميع الردود ─────────────────────────────────────────────────────

func (s *Service) enqueue(phone string, o outcome) {
	if o.Kind == "" && o.Text == "" {
		return
	}
	s.bmu.Lock()
	defer s.bmu.Unlock()
	b := s.batches[phone]
	if b == nil {
		b = &batch{}
		s.batches[phone] = b
	}
	b.outs = append(b.outs, o)
	if b.timer != nil {
		b.timer.Stop()
	}
	b.timer = time.AfterFunc(s.Quiet, func() { s.flush(phone) })
}

func (s *Service) flush(phone string) {
	s.bmu.Lock()
	b := s.batches[phone]
	delete(s.batches, phone)
	s.bmu.Unlock()
	if b == nil || len(b.outs) == 0 {
		return
	}
	if b.timer != nil {
		b.timer.Stop()
	}
	text := summarize(b.outs)
	if s.Send == nil || strings.TrimSpace(text) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Send(ctx, phone, text); err != nil {
		s.log.Warn("waitems: send", "error", err)
	}
}

// FlushAll يرسل كلَّ الدفعات المعلّقة فوراً (للاختبارات وللإيقاف).
func (s *Service) FlushAll() {
	s.bmu.Lock()
	phones := make([]string, 0, len(s.batches))
	for p := range s.batches {
		phones = append(phones, p)
	}
	s.bmu.Unlock()
	for _, p := range phones {
		s.flush(p)
	}
}

// summarize **رسالةٌ واحدة لمجموعة نتائج** — واحدةٌ ⇒ نصُّها، وأكثر ⇒ ملخّص.
func summarize(outs []outcome) string {
	switch len(outs) {
	case 0:
		return ""
	case 1:
		return outs[0].Text
	}
	cnt := map[string]int{}
	var first string
	for _, o := range outs {
		cnt[o.Kind]++
		if o.Kind == oStore && first == "" {
			first = o.Text
		}
	}
	if first != "" {
		return first
	}
	var parts []string
	done := cnt[oCreated] + cnt[oUpdated]
	if done > 0 {
		p := "انضاف " + countNoun(cnt[oCreated], "صنف", "صنفين", "أصناف")
		if cnt[oCreated] == 0 {
			p = "حدّثت " + countNoun(cnt[oUpdated], "صنف", "صنفين", "أصناف")
		} else if cnt[oUpdated] > 0 {
			p += " وتحدّث " + ar(cnt[oUpdated])
		}
		parts = append(parts, p)
	}
	conn := func(n int, tail string) {
		if n == 0 {
			return
		}
		lead := ar(n)
		if len(parts) > 0 {
			lead = "و" + lead
		}
		parts = append(parts, lead+" "+tail)
	}
	conn(cnt[oPrice], "ناقصهن سعر")
	conn(cnt[oName], "ناقصهن اسم")
	conn(cnt[oDup], "اسمهن مكرّر")
	conn(cnt[oSection], "ناقصهن قسم")
	conn(cnt[oSanity], "بانتظار تأكيد السعر")
	conn(cnt[oRate], "ما انضافوا (سرعة)")
	conn(cnt[oFailed], "ما انضافوا (خطأ)")
	out := strings.Join(parts, "، ")
	if cnt[oPrice]+cnt[oName]+cnt[oDup]+cnt[oSanity]+cnt[oSection] > 0 {
		out += "\nردّ على صورة كل صنف ناقص (اضغط عليها ← ردّ) بالمطلوب."
	}
	for _, o := range outs {
		if o.Kind == oSection && o.List != "" {
			out += "\n\nالأقسام:\n" + o.List
			break
		}
	}
	return out
}

func countNoun(n int, one, two, many string) string {
	switch {
	case n == 1:
		return one + " واحد"
	case n == 2:
		return two
	case n >= 3 && n <= 10:
		return ar(n) + " " + many
	default:
		return ar(n) + " " + one
	}
}

// ── ألفاظ ────────────────────────────────────────────────────────────

var toArabic = strings.NewReplacer("0", "٠", "1", "١", "2", "٢", "3", "٣", "4", "٤", "5", "٥", "6", "٦", "7", "٧", "8", "٨", "9", "٩")

func ar(n int) string { return toArabic.Replace(fmt.Sprintf("%d", n)) }

// money «٢٥٬٠٠٠ ل.س» أو «٥ $».
func money(v float64, cur string) string {
	if cur == catalog.CurrencyUSD {
		s := fmt.Sprintf("%.2f", v)
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
		return toArabic.Replace(s) + " $"
	}
	n := int64(math.Round(v))
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString("٬")
		}
		b.WriteRune(c)
	}
	return toArabic.Replace(b.String()) + " ل.س"
}

func normWord(s string) string {
	return strings.ToLower(strings.TrimFunc(strings.TrimSpace(s), func(r rune) bool {
		return strings.ContainsRune("!.?؟،,:;👍 ", r)
	}))
}

func isYes(s string) bool {
	switch normWord(s) {
	case "نعم", "ايه", "أيه", "ايوه", "أيوه", "ايوا", "أيوا", "اي", "أي", "اكيد", "أكيد", "تمام", "صح", "yes", "y", "ok", "اوكي", "أوكي":
		return true
	}
	return false
}

func isNo(s string) bool {
	switch normWord(s) {
	case "لا", "لأ", "لاء", "no", "n", "غلط", "كلا":
		return true
	}
	return false
}

func isBranchWord(s string) bool {
	switch normWord(s) {
	case "فرع", "الفرع", "غير الفرع", "غيّر الفرع", "تغيير الفرع", "تغيير فرع", "غير فرع":
		return true
	}
	return false
}

func isCancel(s string) bool {
	switch normWord(s) {
	case "الغاء", "إلغاء", "ألغاء", "الغ", "الغي", "cancel":
		return true
	}
	return false
}

// pickNumber رقمٌ صحيحٌ وحده («٢» أو «2.» أو «رقم 2»).
func pickNumber(s string) (int, bool) {
	t := strings.TrimSpace(ToASCIIDigits(s))
	t = strings.TrimFunc(t, func(r rune) bool { return r == '.' || r == ')' || r == '-' || r == ' ' })
	t = strings.TrimPrefix(t, "رقم ")
	if t == "" || len(t) > 3 {
		return 0, false
	}
	n := 0
	for _, c := range t {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
