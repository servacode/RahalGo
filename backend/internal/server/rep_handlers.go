package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// لوحة المندوب: كوده للدعوة، متاجره، عمولاته — قراءة فقط (الترويج يتم ميدانياً).

// handleRepMe ملخص المندوب: كود الدعوة والإحصاءات المالية.
func (s *Server) handleRepMe(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	var out struct {
		InviteCode       *string `json:"invite_code"`
		FullName         string  `json:"full_name"`
		Merchants        int     `json:"merchants"`
		DeliveredOrders  int     `json:"delivered_orders"`
		TotalCommissions int64   `json:"total_commissions"`
		Balance          int64   `json:"balance"`
		// الشهر الجاري — الهدف يُقاس عليه لا على المجموع التراكمي
		MonthMerchants   int   `json:"month_merchants"`
		MonthDelivered   int   `json:"month_delivered"`
		MonthCommissions int64 `json:"month_commissions"`
		MonthlyTarget    int   `json:"monthly_target"`
		PendingLeads     int   `json:"pending_leads"`
		// أدوات الدعوة مقفلة حتى يوثّق المندوب قناة تواصله
		WhatsAppVerified bool `json:"whatsapp_verified"`
		// **ورقمُ حسابه يُرسَل** — لتوثّقَه الشاشةُ بلا أن تسأله عنه.
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-١١: «كيف المندوبُ يرجع يكتب رقمَه وهو
		//  موجود؟ ما يصير — خلص، بس وثّق حسابَك بدون ما يرجع يكتب رقم».)
		//
		// **وشاشةُ التوثيق كانت تطلب الرقمَ في حقلٍ فارغ** — تمرّر
		// `me.phone` وهو غيرُ موجودٍ في هذه الاستجابة أصلاً، **فيُقرأ
		// `undefined` ويصير الحقلُ خالياً.** ومن يكتب رقمَه بيده يُخطئ
		// فيه، **فيصله الرمزُ على رقمٍ ليس رقمَه — أو لا يصل.**
		Phone string `json:"phone"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.invite_code, u.full_name,
		       (SELECT count(*) FROM merchants m WHERE m.sales_rep_user_id = u.id),
		       (SELECT count(*) FROM orders o JOIN merchants m ON m.id = o.merchant_id
		        WHERE m.sales_rep_user_id = u.id AND o.status = 'delivered'),
		       -- العمولة **الصافية** لا الإجمالية: العمولات ناقصَ ما عُكس منها عن
		       -- طلبات مُسترجَعة. عرض الإجمالي كان يناقض بطاقات العملاء في الشاشة
		       -- نفسها، ويَعِد المندوب بمالٍ سُحب منه فعلاً. (التسويات اليدوية من
		       -- الإدارة مرجعها فارغ فلا تدخل هنا.)
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = u.id
		                   AND (t.kind = 'commission'
		                        OR (t.kind = 'adjustment' AND t.ref <> ''
		                            AND EXISTS (SELECT 1 FROM orders o3 WHERE o3.id::text = t.ref)))), 0),
		       COALESCE((SELECT w.balance FROM wallets w WHERE w.user_id = u.id), 0),
		       -- الشهر الجاري بتوقيت سوريا (لا UTC: نهاية الشهر تهمّ المندوب)
		       (SELECT count(*) FROM merchants m
		        WHERE m.sales_rep_user_id = u.id
		          AND date_trunc('month', m.created_at AT TIME ZONE 'Asia/Damascus')
		            = date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')),
		       (SELECT count(*) FROM orders o JOIN merchants m ON m.id = o.merchant_id
		        WHERE m.sales_rep_user_id = u.id AND o.status = 'delivered'
		          AND date_trunc('month', o.delivered_at AT TIME ZONE 'Asia/Damascus')
		            = date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = u.id
		                   AND (t.kind = 'commission'
		                        OR (t.kind = 'adjustment' AND t.ref <> ''
		                            AND EXISTS (SELECT 1 FROM orders o4 WHERE o4.id::text = t.ref)))
		                   AND date_trunc('month', t.created_at AT TIME ZONE 'Asia/Damascus')
		                     = date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')), 0),
		       -- **وهدفُ الشهر يُمرَّر من المخزن** — لا يُقرأ هنا برقمٍ مكتوب.
		       $2::int,
		       (SELECT count(*) FROM merchant_leads l
		        WHERE l.sales_rep_user_id = u.id AND l.status = 'new'),
		       u.whatsapp_verified_at IS NOT NULL, u.phone
		FROM users u WHERE u.id = $1`, uid, s.settings.GetInt(r.Context(), "sales.monthly_target")).
		Scan(&out.InviteCode, &out.FullName, &out.Merchants, &out.DeliveredOrders,
			&out.TotalCommissions, &out.Balance,
			&out.MonthMerchants, &out.MonthDelivered, &out.MonthCommissions,
			&out.MonthlyTarget, &out.PendingLeads, &out.WhatsAppVerified, &out.Phone)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// القفل في الخادم لا في الواجهة: إخفاء الكود من الشاشة وحدها ليس قفلاً —
	// من يفتح أدوات المتصفح يقرأه من الاستجابة. فلا يُرسل أصلاً قبل التوثيق.
	if !out.WhatsAppVerified {
		out.InviteCode = nil
	}
	httpx.JSON(w, http.StatusOK, out)
}

// repMerchantsMax سقف صلب لقائمة عملاء المندوب.
//
// ليس ترقيماً بل حاجزُ حجم: مندوبٌ بمئات المتاجر يجرّها كلها في طلب واحد
// فيثقل الخادم والمتصفح معاً. والسقف بعيدٌ عن أي رقم واقعي — من يبلغه فحاله
// يستحقّ بحثاً وترقيماً حقيقيَّين لا تمريرَ قائمة أطول.
const repMerchantsMax = 500

// handleRepMerchants متاجر المندوب مع نشاط كل متجر.
func (s *Server) handleRepMerchants(w http.ResponseWriter, r *http.Request) {
	// البطاقة تجيب سؤال المندوب الحقيقي عن كل عميل: **هل ينتج لي؟** فلا يكفي
	// اسمٌ وتاريخ انضمام — تلزمه العمولة الصافية وآخر نشاط ووسيلة تواصل.
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, c.icon, c.name, lm.thumb_path, m.status,
		       m.created_at::date::text,
		       NULLIF(ou.phone::text, ''),
		       (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id AND o.status = 'delivered'),
		       -- «الملغية» تجمع كل نهاية غير التسليم: رفضٌ من المتجر، وإلغاءٌ من
		       -- الزبون، وفشلُ توصيل، واسترجاعٌ بعد التسليم. تفريقها في بطاقة
		       -- ملخّص يشتّت، والمندوب يقرأها سؤالاً واحداً: كم طلباً ضاع؟
		       (SELECT count(*) FROM orders o WHERE o.merchant_id = m.id
		        AND o.status IN ('rejected', 'cancelled', 'failed', 'refunded')),
		       -- العمولة **الصافية**: العمولات ناقصَ ما عُكس منها عن طلبات مُسترجَعة.
		       -- عرض الإجمالي وحده يَعِد المندوب بمالٍ سُحب منه فعلاً.
		       -- تسويات الإدارة اليدوية لا تدخل هنا: مرجعها فارغ فلا يطابق طلباً.
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 JOIN orders o2 ON t.ref <> '' AND o2.id::text = t.ref
		                 WHERE t.user_id = $1 AND t.kind IN ('commission', 'adjustment')
		                   AND o2.merchant_id = m.id), 0),
		       (SELECT max(o.delivered_at) FROM orders o
		        WHERE o.merchant_id = m.id AND o.status = 'delivered'),
		       -- **عدّادُ التفعيل — بشرط القاعدة حرفياً لا بشرطٍ يشبهه.**
		       --
		       -- العمولة محجوزة حتى يُسلّم المتجرُ عدداً من الطلبات، والقاعدة
		       -- في orders.merchantActivated **تستثني ما اشتراه المندوبُ
		       -- نفسه**. فلو عُدّ هنا بلا هذا الاستثناء لرأى «٥ من ٥» ولم يقبض
		       -- شيئاً — **وعدّادٌ يقول «اكتمل» ومالٌ لا يأتي أسوأ من لا عدّاد**:
		       -- الأول يجعله يشكّ في المنصة، والثاني يجعله يسأل.
		       (SELECT count(*) FROM orders o
		        WHERE o.merchant_id = m.id AND o.status = 'delivered'
		          AND o.customer_id IS DISTINCT FROM m.sales_rep_user_id)
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN users ou ON ou.id = m.owner_user_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.sales_rep_user_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2`, userIDFrom(r), repMerchantsMax)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type repMerchant struct {
		ID           string     `json:"id"`
		Name         string     `json:"name"`
		CategoryIcon string     `json:"category_icon"`
		CategoryName string     `json:"category_name"`
		LogoThumbURL *string    `json:"logo_thumb_url"`
		Status       string     `json:"status"`
		JoinedAt     string     `json:"joined_at"`
		OwnerPhone   *string    `json:"owner_phone"`
		Delivered    int        `json:"delivered_orders"`
		Cancelled    int        `json:"cancelled_orders"`
		MyCommission int64      `json:"my_commission"`
		LastOrderAt  *time.Time `json:"last_order_at"`
		// ActivationDone كم طلباً احتُسب نحو التفعيل، و**ActivationNeeded** كم
		// يلزم. متساويان أو أكثر يعني أن العمولة تجري.
		ActivationDone   int   `json:"activation_done"`
		ActivationNeeded int64 `json:"activation_needed"`
	}
	// **العتبةُ تُقرأ مرّةً لا لكل متجر** — وهي إعدادُ منصةٍ لا خاصّيةُ متجر.
	activationNeeded := s.settings.GetInt(r.Context(), "sales.activation_orders")
	out := []repMerchant{}
	for rows.Next() {
		var m repMerchant
		if err := rows.Scan(&m.ID, &m.Name, &m.CategoryIcon, &m.CategoryName, &m.LogoThumbURL,
			&m.Status, &m.JoinedAt, &m.OwnerPhone, &m.Delivered, &m.Cancelled,
			&m.MyCommission, &m.LastOrderAt, &m.ActivationDone); err != nil {
			s.respondErr(w, err)
			return
		}
		m.ActivationNeeded = activationNeeded
		m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
		out = append(out, m)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRepWallet كشف محفظة المندوب (العمولات والتسويات).
func (s *Server) handleRepWallet(w http.ResponseWriter, r *http.Request) {
	// بلا مدى: لمحة اللوحة (آخر 50). بمدى: كشف حساب كامل قابل للطباعة.
	st, err := s.wallet.Statement(r.Context(), userIDFrom(r), statementRange(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}
