package server

// هويّةُ المنصة — **من يتعاقد، وإلى من يُشتكى، وأين يقع.**
//
// # لماذا من الإعدادات لا من الشيفرة
//
// الشروطُ والخصوصيةُ وثيقتان قانونيّتان تذكران اسماً ورقماً وعنواناً، **وهذه
// تتغيّر**: يُسجَّل الاسمُ التجاريّ، ويُبدَّل رقمُ الدعم، **وينتقل المكتب.**
//
// **وما كُتب في شيفرةٍ لا يُبدَّل إلّا بنشر** — فيبقى الرقمُ القديمُ معروضاً
// شهراً، **ومن اتّصل به لم يجد أحداً**، ويقرؤها زبونٌ فيظنّ المنصةَ مهجورة.
//
// # وعامّةٌ لا محميّة
//
// **من يُسأل أن يوافق على الشروط يقرؤها قبل أن يدخل** — ووثيقةٌ لا تُقرأ إلّا
// بعد التسجيل وثيقةٌ يُوافَق عليها بلا قراءة.

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handlePublicContact ما تعرضه الصفحاتُ القانونية — **وفارغُه يُحذف لا يُعرض.**
//
// **وسطرٌ يقول «الهاتف: —» أسوأُ من غيابه**: يُقرأ عطباً في المنصة لا حقلاً لم
// يُملأ بعد. فتُعاد الحقولُ كما هي، **والشاشةُ تُسقط الفارغَ منها.**
func (s *Server) handlePublicContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	httpx.JSON(w, http.StatusOK, map[string]any{
		"legal_name":    s.settings.GetString(ctx, "platform.legal_name"),
		"support_phone": s.settings.GetString(ctx, "platform.support_phone"),
		"address":       s.settings.GetString(ctx, "platform.address"),
	})
}
