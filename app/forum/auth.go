package forum

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type requestState struct {
	User        User
	Token, CSRF string
}
type stateKey struct{}

func state(r *http.Request) requestState {
	value, _ := r.Context().Value(stateKey{}).(requestState)
	return value
}
func randomToken() (string, error) {
	var bytes [32]byte
	_, err := rand.Read(bytes[:])
	return hex.EncodeToString(bytes[:]), err
}
func hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
func (a *App) cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (a *App) requestState(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	st := requestState{}
	if cookie, err := r.Cookie("session"); err == nil {
		err = a.store.DB.QueryRow("SELECT u.id,u.username,u.email,u.role FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>?", hash(cookie.Value), a.now().Unix()).Scan(&st.User.ID, &st.User.Name, &st.User.Email, &st.User.Role)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			st.Token = cookie.Value
			st.CSRF = a.sign("csrf:" + cookie.Value)
		}
	}
	if st.User.ID == 0 {
		cookie, err := r.Cookie("csrf")
		if err == nil && a.validPreauth(cookie.Value) {
			st.CSRF = cookie.Value
		}
		if st.CSRF == "" && r.Method == http.MethodGet {
			token, err := randomToken()
			if err != nil {
				return nil, err
			}
			payload := token + "." + strconv.FormatInt(a.now().Add(time.Hour).Unix(), 10)
			st.CSRF = payload + "." + a.sign("preauth:"+payload)
			a.cookie(w, "csrf", st.CSRF, 3600)
		}
	}
	return r.WithContext(context.WithValue(r.Context(), stateKey{}, st)), nil
}
func (a *App) validPreauth(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || expiry <= a.now().Unix() || expiry > a.now().Add(time.Hour).Unix() {
		return false
	}
	expected := a.sign("preauth:" + parts[0] + "." + parts[1])
	return hmac.Equal([]byte(parts[2]), []byte(expected))
}
func validCSRF(expected, actual string) bool {
	return expected != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
func (a *App) startSession(w http.ResponseWriter, userID int64) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	tx, err := a.store.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM sessions WHERE user_id=? OR expires_at<=?", userID, a.now().Unix()); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO sessions(token_hash,user_id,created_at,expires_at) VALUES(?,?,?,?)", hash(token), userID, a.now().Unix(), a.now().Add(24*time.Hour).Unix()); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	a.cookie(w, "session", token, 86400)
	a.cookie(w, "csrf", "", -1)
	return nil
}
func (s *Store) Register(username, email, password string) (int64, error) {
	encoded, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	result, err := s.DB.Exec("INSERT INTO users(username,email,password) VALUES(?,?,?)", username, email, string(encoded))
	if err != nil {
		if constraint(err) {
			return 0, ErrConflict
		}
		return 0, err
	}
	return result.LastInsertId()
}
func (s *Store) Authenticate(username, password string) (int64, error) {
	var id int64
	var encoded string
	err := s.DB.QueryRow("SELECT id,password FROM users WHERE username=?", username).Scan(&id, &encoded)
	if err != nil {
		return 0, err
	}
	if bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password)) != nil {
		return 0, fmt.Errorf("invalid credentials: %w", sql.ErrNoRows)
	}
	return id, nil
}
