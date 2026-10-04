package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// كشف محفظة مستخدم — أدمن/مالية/عمليات (قراءة).
//
// **ويُقلَّب ولا يُقصّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٥): كان يردّ خمسين ويقول
// `truncated` **ولا تقرؤه الشاشة**، فمن راجع دفترَ زبونٍ اشتكى «رصيدي ناقص»
// رأى خمسين حركةً وظنّها كلَّ ما وقع.
func (s *Server) handleAdminWalletStatement(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	st, err := s.wallet.Statement(r.Context(), chi.URLParam(r, "id"),
		wallet.StatementRange{Limit: limit, Page: page})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}
