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
	CreateUser(user string, pass string) (*D, error)
	CheckPassword(user string, pass string) bool
	RestoreData(user string, pass string) (*D, error)
	PersistData(user string, data *D)
}

type sessionCacheEntry[D any] struct {
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

func NewSessionCache[S any](sm Manager[S], lifeTime time.Duration) *sessionCache[S] {
	shutDown := make(chan struct{})
	sc := sessionCache[S]{
		sessions: make(map[string]*sessionCacheEntry[S]),
		sm:       sm,
		shutDown: shutDown,
		lifeTime: lifeTime,
	}

	go func() {
		for {
			select {
			case <-time.After(lifeTime):
				sc.checkSessions()
			case <-shutDown:
				return
			}
		}
	}()

	return &sc
}

func (s *sessionCache[S]) getSession(id string) *S {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if sce, ok := s.sessions[id]; ok {
		if time.Since(sce.lastAccess) < s.lifeTime {
			sce.lastAccess = time.Now()
			return sce.data
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
		if sce.user == user && s.sm.CheckPassword(user, pass) {
			sce.lastAccess = time.Now()
			return id, nil
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

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func createRandomString() string {
	b := make([]byte, 20)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func CheckSessionFunc[S any](sc *sessionCache[S], parent http.HandlerFunc) http.HandlerFunc {
	return CheckSession(sc, parent)
}

func CheckSession[S any](sc *sessionCache[S], parent http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("id"); err == nil {
			id := c.Value
			if s := sc.getSession(id); s != nil {
				nc := context.WithValue(r.Context(), "data", s)
				parent.ServeHTTP(w, r.WithContext(nc))
				return
			}
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func CheckSessionRest[S any](sc *sessionCache[S], parent http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("id"); err == nil {
			id := c.Value
			if s := sc.getSession(id); s != nil {
				nc := context.WithValue(r.Context(), "data", s)
				parent.ServeHTTP(w, r.WithContext(nc))
				return
			}
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

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
				http.SetCookie(w, &http.Cookie{Value: id, Name: "id", Expires: time.Now().Add(sc.lifeTime)})
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
				http.SetCookie(w, &http.Cookie{Value: id, Name: "id", Expires: time.Now().Add(sc.lifeTime)})
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
