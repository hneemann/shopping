package session

import (
	"context"
	"errors"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Manager[D any] interface {
	// CreateUser is called in a new user needs to be created
	CreateUser(user string, pass string) (*D, error)
	// CheckPassword is called to check if the password is correct
	CheckPassword(user string, pass string) bool
	// RestoreData is called to restore the data for a user
	// In most cases this will mean to load a file or similar
	RestoreData(user string, pass string) (*D, error)
	// PersistData is called to save the data for a user
	// In most cases this will mean to write a file or similar
	PersistData(user string, data *D)
}

type sessionCacheEntry[D any] struct {
	mutex      sync.Mutex
	lastAccess time.Time
	user       string
	data       *D
}

type sessionCache[D any] struct {
	mutex    sync.Mutex
	lifeTime time.Duration
	sessions map[string]*sessionCacheEntry[D]
	sm       Manager[D]
	shutDown chan struct{}
}

// NewSessionCache creates a new session cache
func NewSessionCache[S any](sm Manager[S], sessionLifeTime time.Duration) *sessionCache[S] {
	shutDown := make(chan struct{})
	sc := sessionCache[S]{
		sessions: make(map[string]*sessionCacheEntry[S]),
		sm:       sm,
		shutDown: shutDown,
		lifeTime: sessionLifeTime,
	}

	go func() {
		for {
			select {
			case <-time.After(sessionLifeTime):
				sc.checkSessions()
			case <-shutDown:
				return
			}
		}
	}()

	return &sc
}

func (s *sessionCache[S]) getSession(id string) *sessionCacheEntry[S] {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if sce, ok := s.sessions[id]; ok {
		if time.Since(sce.lastAccess) < s.lifeTime {
			return sce
		} else {
			s.sm.PersistData(sce.user, sce.data)
			delete(s.sessions, id)
		}
	}
	return nil
}

func (s *sessionCache[S]) createSessionId(user string, pass string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sm == nil {
		return "", errors.New("session manager closed")
	}

	for id, sce := range s.sessions {
		if sce.user == user {
			if s.sm.CheckPassword(user, pass) {
				sce.lastAccess = time.Now()
				return id, nil
			} else {
				return "", errors.New("wrong password")
			}
		}
	}

	data, err := s.sm.RestoreData(user, pass)
	if err != nil {
		return "", err
	}
	id := createRandomString()

	ses := &sessionCacheEntry[S]{lastAccess: time.Now(), data: data, user: user}
	s.sessions[id] = ses

	return id, nil
}

func (s *sessionCache[S]) registerUser(user, pass, pass2 string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if pass != pass2 {
		return "", errors.New("passwords do not match")
	}

	if s.sm == nil {
		return "", errors.New("session manager closed")
	}

	data, err := s.sm.CreateUser(user, pass)
	if err != nil {
		return "", err
	}
	id := createRandomString()

	ses := &sessionCacheEntry[S]{lastAccess: time.Now(), data: data, user: user}
	s.sessions[id] = ses

	return id, nil
}

func (s *sessionCache[S]) checkSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sm == nil {
		return
	}

	for id, sce := range s.sessions {
		if time.Since(sce.lastAccess) > s.lifeTime {
			s.sm.PersistData(sce.user, sce.data)
			delete(s.sessions, id)
		}
	}
}

func (s *sessionCache[S]) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	close(s.shutDown)

	for _, sce := range s.sessions {
		s.sm.PersistData(sce.user, sce.data)
	}
	s.sm = nil
}

func (s *sessionCache[D]) doHandler(w http.ResponseWriter, r *http.Request, parent http.Handler) bool {
	if c, err := r.Cookie("id"); err == nil {
		id := c.Value
		if se := s.getSession(id); se != nil {
			se.mutex.Lock()
			defer se.mutex.Unlock()
			se.lastAccess = time.Now()
			nc := context.WithValue(r.Context(), "data", se.data)
			parent.ServeHTTP(w, r.WithContext(nc))
			return true
		} else {
			log.Println("no session found")
		}
	} else {
		log.Println("no cookie send")
	}
	return false
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func createRandomString() string {
	b := make([]byte, 20)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// CheckSessionFunc is a wrapper that redirects to /login if no valid session id is found
func CheckSessionFunc[S any](sc *sessionCache[S], parent http.HandlerFunc) http.HandlerFunc {
	return CheckSession(sc, parent)
}

// CheckSession is a wrapper that redirects to /login if no valid session id is found
func CheckSession[S any](sc *sessionCache[S], parent http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok := sc.doHandler(w, r, parent); !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

// CheckSessionRest is a wrapper that returns a 403 Forbidden if no valid session id is found
func CheckSessionRest[S any](sc *sessionCache[S], parent http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok := sc.doHandler(w, r, parent); !ok {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	}
}

// LoginHandler is a handler that handles the login.
// The given template is used to render the login page.
// It needs to contain a form with the fields username and password.
func LoginHandler[S any](sc *sessionCache[S], loginTemp *template.Template) http.HandlerFunc {
	if loginTemp == nil {
		panic("login template is nil")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		if r.Method == http.MethodPost {
			user := r.FormValue("username")
			pass := r.FormValue("password")

			var id string
			if id, err = sc.createSessionId(user, pass); err == nil {
				http.SetCookie(w, &http.Cookie{Value: id, Name: "id"})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}
		err = loginTemp.Execute(w, err)
		if err != nil {
			log.Println(err)
		}
	}
}

func LogoutHandler[S any](sc *sessionCache[S], logoutTemp *template.Template) http.HandlerFunc {
	if logoutTemp == nil {
		panic("logout template is nil")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("id"); err == nil {
			id := c.Value
			if se := sc.getSession(id); se != nil {
				se.mutex.Lock()
				defer se.mutex.Unlock()
				sc.sm.PersistData(se.user, se.data)
				delete(sc.sessions, id)
			}
			http.SetCookie(w, &http.Cookie{Value: "", Name: "id", Expires: time.Now().Add(-time.Hour)})
		}
		err := logoutTemp.Execute(w, nil)
		if err != nil {
			log.Println(err)
		}
	}
}

// RegisterHandler is a handler that handles the registration.
// The given template is used to render the registration page.
// It needs to contain a form with the fields username, password and password2.
func RegisterHandler[S any](sc *sessionCache[S], registerTemp *template.Template) http.HandlerFunc {
	if registerTemp == nil {
		panic("register template is nil")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		if r.Method == http.MethodPost {
			user := r.FormValue("username")
			pass := r.FormValue("password")
			pass2 := r.FormValue("password2")

			var id string
			if id, err = sc.registerUser(user, pass, pass2); err == nil {
				http.SetCookie(w, &http.Cookie{Value: id, Name: "id"})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}
		err = registerTemp.Execute(w, err)
		if err != nil {
			log.Println(err)
		}
	}
}
