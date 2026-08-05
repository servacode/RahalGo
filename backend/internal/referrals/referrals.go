package referrals

// دعوةُ زبونٍ لزبون — **ومن جلب يُكافأ، والمنصةُ تدفع.**
//
// # ثلاثةُ أسئلةٍ وثلاثةُ أجوبة
//
//	متى يُنسب؟   ←  لحظةَ إنشاء الحساب بالرمز
//	متى يُصرف؟   ←  **زرٌّ ذكيّ**: عند إكمال التسجيل أو عند أوّل طلب
//	كم؟         ←  بترتيب الدعوة: الأولى ثمّ الثانية ثمّ الثالثة ثمّ ثابت
//
// **والنسبُ غيرُ الصرف** — بينهما شرطٌ يُختار.
//
// # والوضعان يختلفان في الثقة لا في المال
//
// **«عند أوّل طلب» أحوطُ**: زبونٌ اشترى فعلاً. **لكنّ من دعا صديقَه ثمّ انتظر
// أسبوعاً حتى يطلب يظنّ أنّنا نماطله** — فلا يدعو ثانياً ولا يُصدّق ما نقول.
//
// **و«عند إكمال التسجيل» يُوفي في دقيقة** — ويُشترى به ولاءٌ لا يُشتَرى
// بإعلان. **والواتسابُ هو الحارس**: بلاه تُفتح مئةُ حسابٍ بأرقامٍ تُشترى،
// **فتُدفع مئةُ مكافأةٍ على مئةٍ لا وجودَ لها.**
//
// **وهو الافتراض**: منصةٌ جديدةٌ تحتاج ثقةً قبل أن تحتاج حذراً.
//
// # والرتبةُ تُثبَّت لحظةَ النسب
//
// من دعا ثلاثةً في يومٍ ثمّ طلبوا بترتيبٍ مقلوبٍ **لا تنقلب مكافآتُه**:
// السُّلَّمُ يُقرأ من ترتيب الدعوة لا من ترتيب الطلب. **وإلّا صار من دعا أوّلاً
// يأخذ أقلَّ لأنّ صاحبَه تأخّر في الطلب** — وهو ما لا يفهمه أحد.

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrBadCode رمزٌ لا يعرفه أحد.
	ErrBadCode = httpx.NewError(http.StatusNotFound, "bad_invite_code", "errors.bad_invite_code")
	// ErrSelfInvite **ولا يدعو أحدٌ نفسَه** — حسابان يتبادلان الدعوةَ يطبعان
	// مالاً من العدم.
	ErrSelfInvite = httpx.NewError(http.StatusConflict, "self_invite", "errors.self_invite")
	// ErrAlreadyInvited دُعي مرّةً — **ولا يُنسب لغير أوّل من دعاه.**
	ErrAlreadyInvited = httpx.NewError(http.StatusConflict, "already_invited", "errors.already_invited")
)

// Settings ما تقرؤه هذه الحزمة — أربعةُ أرقامٍ ووضعٌ واحد.
type Settings interface {
	GetInt(ctx context.Context, key string) int64
	GetString(ctx context.Context, key string) string
}

// أوضاعُ صرف المكافأة.
const (
	// OnSignup عند إكمال التسجيل — **حسابٌ تمّ ورقمُ واتسابٍ وُثّق.**
	OnSignup = "signup"
	// OnFirstOrder عند أوّل طلبٍ يُسلَّم — **زبونٌ اشترى فعلاً.**
	OnFirstOrder = "first_order"
)

// rewardOn الوضعُ النافذ — **وافتراضُه التسجيل**: منصةٌ جديدةٌ تحتاج ثقةً
// قبل أن تحتاج حذراً.
func (s *Service) rewardOn(ctx context.Context) string {
	if v := s.settings.GetString(ctx, "referral.reward_on"); v != "" {
		return v
	}
	return OnSignup
}

type Service struct {
	db       *pgxpool.Pool
	wallet   *wallet.Service
	settings Settings
	treasury func(ctx context.Context) string
	notify   Notifier
}

// Notifier ما يلزم لإبلاغ من نال — **ومكافأةٌ لا يراها صاحبُها لم تُصرف
// في نظره.**
type Notifier interface {
	NotifyWallet(ctx context.Context, userID, title, body, href string)
}

func New(db *pgxpool.Pool, w *wallet.Service, st Settings,
	treasury func(context.Context) string, n Notifier) *Service {
	return &Service{db: db, wallet: w, settings: st, treasury: treasury, notify: n}
}

// rewardFor مكافأةُ الرتبة — **والرابعةُ فما فوقها بالثابت.**
func (s *Service) rewardFor(ctx context.Context, rank int) int64 {
	switch rank {
	case 1:
		return s.settings.GetInt(ctx, "referral.reward_1")
	case 2:
		return s.settings.GetInt(ctx, "referral.reward_2")
	case 3:
		return s.settings.GetInt(ctx, "referral.reward_3")
	default:
		return s.settings.GetInt(ctx, "referral.reward_rest")
	}
}

// MyCode رمزُ الدعوة لهذا الحساب — **يُولَّد عند أوّل طلبٍ له.**
//
// **ولا يُولَّد للجميع مسبقاً**: أكثرُ الزبائن لا يدعو أحداً، **ورمزٌ لكلّ
// حسابٍ ملءُ جدولٍ بما لا يُستعمل.**
func (s *Service) MyCode(ctx context.Context, userID string) (string, error) {
	var code *string
	if err := s.db.QueryRow(ctx,
		`SELECT NULLIF(invite_code, '') FROM users WHERE id = $1`, userID).Scan(&code); err != nil {
		return "", err
	}
	if code != nil && *code != "" {
		return *code, nil
	}
	// **والتصادمُ يُعالَج بإعادة المحاولة لا بالرجاء**: الفهرسُ الفريدُ يمنع،
	// **وستُّ خاناتٍ من اثنين وثلاثين حرفاً تكفي لعشرات الآلاف.**
	for i := 0; i < 8; i++ {
		c := newCode()
		tag, err := s.db.Exec(ctx,
			`UPDATE users SET invite_code = $2 WHERE id = $1
			   AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.invite_code = $2)`,
			userID, c)
		if err == nil && tag.RowsAffected() == 1 {
			return c, nil
		}
		if err != nil && !strings.Contains(err.Error(), "users_invite_code_uniq") {
			return "", err
		}
	}
	return "", errors.New("referrals: تعذّر توليدُ رمزٍ فريد")
}

// alphabet **بلا حروفٍ تُقرأ خطأً**: لا صفرَ ولا O، ولا واحدَ ولا I ولا L.
// **ورمزٌ يُملى بالهاتف يُكتب خطأً مرّةً في العشر** لولا ذلك.
const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

func newCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out := make([]byte, 6)
	for i := range b {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out)
}

// Attach ينسب حساباً جديداً إلى من دعاه — **لحظةَ الإنشاء.**
//
// **ولا يُصرف هنا شيء**: النسبُ غيرُ الصرف، وبينهما زبونٌ قد لا يطلب أبداً.
func (s *Service) Attach(ctx context.Context, inviteeID, code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil
	}
	var inviterID string
	err := s.db.QueryRow(ctx,
		`SELECT id::text FROM users WHERE invite_code = $1`, code).Scan(&inviterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBadCode
	}
	if err != nil {
		return err
	}
	if inviterID == inviteeID {
		return ErrSelfInvite
	}

	// **والرتبةُ تُحسب داخل الإدراج** — لا تُقرأ ثمّ تُكتب: دعوتان متزامنتان
	// تأخذان الرتبةَ نفسَها لو فُصلا.
	_, err = s.db.Exec(ctx, `
		INSERT INTO referrals (invitee_id, inviter_id, rank)
		VALUES ($1, $2,
		        (SELECT count(*) + 1 FROM referrals WHERE inviter_id = $2))
		ON CONFLICT (invitee_id) DO NOTHING`, inviteeID, inviterID)
	return err
}

// SettleFirstOrder يصرف عند أوّل طلبٍ يُسلَّم — **إن كان ذاك هو الوضع.**
//
// **وتُنادى بعد كلّ تسليم** وتخرج صامتةً إن لم يكن ثمّة ما يُصرف: **حارسٌ في
// موضعٍ واحدٍ خيرٌ من شرطٍ يُكتب في كلّ نداء.**
func (s *Service) SettleFirstOrder(ctx context.Context, customerID, orderID, actorID string) {
	if s.rewardOn(ctx) != OnFirstOrder {
		return
	}
	s.settle(ctx, customerID, orderID, actorID)
}

// SettleOnSignup يصرف عند إكمال التسجيل — **إن كان ذاك هو الوضع.**
//
// **و«إكمالُ التسجيل» توثيقُ الواتساب لا فتحُ الحساب**: بلاه تُفتح مئةُ حسابٍ
// في ساعةٍ بأرقامٍ تُشترى، **فتُدفع مئةُ مكافأةٍ على مئةٍ لا وجودَ لها.**
//
// **ومرجعُ القيد فارغٌ هنا** — لا طلبَ بعد. **ورقمٌ بلا مرجعٍ يُقرأ في الكشف
// بنصّه**: «مكافأةُ دعوةِ زبون» تكفي.
func (s *Service) SettleOnSignup(ctx context.Context, customerID, actorID string) {
	if s.rewardOn(ctx) != OnSignup {
		return
	}
	s.settle(ctx, customerID, "", actorID)
}

// settle جسدُ الصرف — **واحدٌ للوضعين.**
//
// **وحسبتان لصرفٍ واحدٍ تفترقان يوماً**: يُصلَح الختمُ في أحدهما ويبقى الآخر
// **فتُصرف المكافأةُ مرّتين.**
func (s *Service) settle(ctx context.Context, customerID, orderID, actorID string) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **والقفلُ داخل المعاملة**: طلبان يُسلَّمان معاً لا يصرفان مكافأتين.
	var inviterID string
	var rank int
	err = tx.QueryRow(ctx, `
		SELECT inviter_id::text, rank FROM referrals
		WHERE invitee_id = $1 AND rewarded_at IS NULL
		FOR UPDATE`, customerID).Scan(&inviterID, &rank)
	if err != nil {
		return // لا دعوةَ معلّقة — وهو الغالب
	}

	amount := s.rewardFor(ctx, rank)
	if _, err := tx.Exec(ctx, `
		UPDATE referrals SET rewarded_at = now(), reward_amount = $2
		WHERE invitee_id = $1`, customerID, amount); err != nil {
		return
	}
	// **وصفرٌ يُختم ولا يُدفع**: الدعوةُ تُحسب وإن توقّف الصرف — **والقياسُ
	// يبقى.** ولولا الختمُ لَبقيت معلّقةً فتُصرف يومَ يُرفع الرقم عن طلبٍ
	// قديم.
	if amount > 0 {
		if _, err := s.wallet.ApplyTx(ctx, tx, inviterID, amount, "reward",
			orderID, "مكافأةُ دعوةِ زبون", &actorID); err != nil {
			return
		}
		// **والخزينةُ الطرفُ المقابل** — دفترٌ يأخذ من طرفٍ ولا يعطي آخرَ
		// لا يتوازن. (قرارُ المالك: «المنصةُ تتحمّل التكاليف».)
		if tid := s.treasury(ctx); tid != "" {
			if _, err := s.wallet.ApplyTx(ctx, tx, tid, -amount, "reward",
				orderID, "مكافأةُ دعوةٍ صُرفت", &actorID); err != nil {
				return
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	if amount > 0 && s.notify != nil {
		// **والنصُّ يتبع الوضع.**
		//
		// **و«طلب أوّلَ طلبٍ له» تُقال لمن سجّل للتوّ كذبةٌ صغيرة** — يقرؤها
		// الداعي فيسأل صديقَه عن طلبٍ لم يقع، **ويظنّ أنّ في الحساب خللاً.**
		body := "صديقٌ دعوتَه أكمل تسجيلَه"
		if orderID != "" {
			body = "صديقٌ دعوتَه طلب أوّلَ طلبٍ له"
		}
		s.notify.NotifyWallet(ctx, inviterID, "مكافأةُ دعوة", body, "/wallet")
	}
}

// Standing حالُ دعوات صاحب الحساب.
type Standing struct {
	Code string `json:"code"`
	// Invited من سُجّلوا برمزه، و Rewarded من طلبوا فعلاً.
	Invited  int   `json:"invited"`
	Rewarded int   `json:"rewarded"`
	Earned   int64 `json:"earned"`
	// NextReward ما سيناله عن الدعوة القادمة — **والرقمُ يُقال قبل الفعل.**
	//
	// **ووعدٌ مبهمٌ لا يُحرّك أحداً**: «ادعُ أصدقاءك» لا تعني شيئاً،
	// **و«ادعُ صديقاً واربح ٥٬٠٠٠» تعني.**
	NextReward int64 `json:"next_reward"`
}

func (s *Service) Standing(ctx context.Context, userID string) (*Standing, error) {
	code, err := s.MyCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := Standing{Code: code}
	if err := s.db.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE rewarded_at IS NOT NULL),
		       COALESCE(sum(reward_amount), 0)
		FROM referrals WHERE inviter_id = $1`, userID).
		Scan(&out.Invited, &out.Rewarded, &out.Earned); err != nil {
		return nil, err
	}
	out.NextReward = s.rewardFor(ctx, out.Invited+1)
	return &out, nil
}
