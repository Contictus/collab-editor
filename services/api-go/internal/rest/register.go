package rest

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Contictus/collab-editor/services/api-go/internal/auth"
	"github.com/Contictus/collab-editor/services/api-go/internal/db"
)

type credentialsForm struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// issueSession signs the identity and sets the session cookie (createSession parity).
func (a *API) issueSession(w http.ResponseWriter, user *db.User) error {
	token, err := auth.SignSession(auth.SessionUser{ID: user.ID, Email: user.Email}, a.Secret)
	if err != nil {
		return err
	}
	auth.SetSession(w, token, a.SecureCookies)
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// handleRegister creates the user and signs them in (registerAction parity,
// JSON surface): 201 + user, 409 on taken email, 400 on bad shape.
func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	// REGISTER_PER_IP first (even malformed floods count), mirrors registerAction.
	if a.throttled(w, "register:ip:"+clientIP(r), 5, 600_000, "Too many attempts. Please try again later.") {
		return
	}
	form, ok := decodeBody[credentialsForm](r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := auth.ValidateCredentials(form.Email, form.Password); err != nil {
		writeErr(w, http.StatusBadRequest, "Enter a valid email and a password of at least 8 characters.")
		return
	}
	hash, err := auth.HashPassword(form.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	user, err := a.Store.CreateUser(r.Context(), form.Email, hash)
	if err != nil {
		if isUniqueViolation(err) {
			writeErr(w, http.StatusConflict, "That email is already registered.")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := a.issueSession(w, user); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": user.ID, "email": user.Email})
}
