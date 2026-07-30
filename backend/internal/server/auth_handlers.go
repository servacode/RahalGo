package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errValidation = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func decode[T any](r *http.Request) (*T, error) {
	var v T
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&v); err != nil {
		return nil, errValidation
	}
	return &v, nil
}

// respondErr يحول أخطاء المجال إلى استجابة موحدة، ويخفي التفاصيل الداخلية.
func (s *Server) respondErr(w http.ResponseWriter, err error) {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.Error(w, appErr)
		return
	}
	s.logger.Error("internal error", "error", err)
	httpx.Error(w, httpx.ErrInternal)
}

func (s *Server) handleOTPRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestOTP(r.Context(), req.Phone); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.VerifyOTP(r.Context(), req.Phone, req.Code, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.LoginPassword(r.Context(), req.Phone, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.Logout(r.Context(), req.RefreshToken, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"logged_out": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.identity.Me(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.SetPassword(r.Context(), userIDFrom(r), req.Password, req.CurrentPassword, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func clientIP(r *http.Request) string {
	return r.RemoteAddr
}
