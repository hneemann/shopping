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

// Persist is the interface that needs to be implemented to persist the session data
type Persist[D any] interface {
	// Load is called to load the data from the persistant storage
	Load() (*D, error)
	// Save is called to save the data to the persistant storage
	Save(d *D) error
}

// Manager is the interface that needs to be implemented to manage the session data
// D is the type of the data that is stored in the session
// The manager is responsible for creating new users, checking the password and
// creating the persist interface for the user
type Manager[D any] interface {
	// CreateUser is called if a new user needs to be created
	CreateUser(user, pass string) (*D, error)
	// CheckPassword is called to check if the password is correct
	CheckPassword(user, pass string) bool
	// CreatePersist create the persist interface used to
	// restore and persist the user data
	CreatePersist(user, pass string) (Persist[D], error)
}

type sessionCacheEntry[D any] struct {
	mutex      sync.Mutex
	lastAccess time.Time
	user       string
	persist    Persist[D]
	data       *D
}

// Cache is the session cache
type Cache[D any] struct {
	mutex    sync.Mutex
	lifeTime time.Duration
	sessions map[string]*sessionCacheEntry[D]
	sm       Manager[D]
	shutDown chan struct{}
	loginUrl string
	mainUrl  string
}

// NewSessionCache creates a new session cache
// sm is the session manager
// sessionLifeTime is the time a session is valid
func NewSessionCache[S any](sm Manager[S], sessionLifeTime time.Duration) *Cache[S] {
	shutDown := make(chan struct{})
	sc := Cache[S]{
		sessions: make(map[string]*sessionCacheEntry[S]),
		sm:       sm,
		shutDown: shutDown,
		lifeTime: sessionLifeTime,
		loginUrl: "/login",
		mainUrl:  "/",
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

// SetLoginUrl sets the url to redirect to if no session is found
func (s *Cache[S]) SetLoginUrl(url string) *Cache[S] {
	s.loginUrl = url
	return s
}

// SetMainUrl sets the url to redirect to after login
func (s *Cache[S]) SetMainUrl(url string) *Cache[S] {
	s.mainUrl = url
	return s
}

func (s *Cache[S]) getSession(id string) *sessionCacheEntry[S] {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if sce, ok := s.sessions[id]; ok {
		if time.Since(sce.lastAccess) < s.lifeTime {
			return sce
		} else {
			err := sce.persist.Save(sce.data)
			if err != nil {
				log.Println(err)
			}
			delete(s.sessions, id)
		}
	}
	return nil
}

func (s *Cache[S]) CreateSessionId(user string, pass string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sm == nil {
		return "", errors.New("session manager closed")
	}

	if !s.sm.CheckPassword(user, pass) {
		return "", errors.New("wrong password")
	}

	for id, sce := range s.sessions {
		if sce.user == user {
			sce.lastAccess = time.Now()
			log.Println("gained access to an existing session")
			return id, nil
		}
	}

	p, err := s.sm.CreatePersist(user, pass)
	if err != nil {
		return "", err
	}

	data, err := p.Load()
	if err != nil {
		return "", err
	}
	id := createRandomString()

	ses := &sessionCacheEntry[S]{lastAccess: time.Now(), data: data, user: user, persist: p}
	s.sessions[id] = ses

	return id, nil
}

func (s *Cache[S]) registerUser(user, pass, pass2 string) (string, error) {
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
	p, err := s.sm.CreatePersist(user, pass)
	if err != nil {
		return "", err
	}

	id := createRandomString()

	ses := &sessionCacheEntry[S]{lastAccess: time.Now(), data: data, user: user, persist: p}
	s.sessions[id] = ses

	return id, nil
}

func (s *Cache[S]) checkSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sm == nil {
		return
	}

	for id, sce := range s.sessions {
		if time.Since(sce.lastAccess) > s.lifeTime {
			err := sce.persist.Save(sce.data)
			if err != nil {
				log.Println(err)
			}
			delete(s.sessions, id)
		}
	}
}

// Close closes the session cache
// It saves all data and stops the session cache
// This function should be called before the program exits
// to save all the session data. It also stops the go routine
// that periodically checks the session lifetime.
func (s *Cache[S]) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	close(s.shutDown)

	for _, sce := range s.sessions {
		err := sce.persist.Save(sce.data)
		if err != nil {
			log.Println(err)
		}
	}
	s.sm = nil
}

// CallHandlerWithData calls the parent handler with the data from the session.
// The data is stored in the context with the key "data".
// If no session is found it returns false.
func (s *Cache[D]) CallHandlerWithData(w http.ResponseWriter, r *http.Request, parent http.Handler) bool {
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
			log.Println("no matching session found")
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
func (s *Cache[S]) CheckSessionFunc(parent http.HandlerFunc) http.HandlerFunc {
	return s.CheckSession(parent)
}

// CheckSession is a wrapper that redirects to /login if no valid session id is found
func (s *Cache[S]) CheckSession(parent http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok := s.CallHandlerWithData(w, r, parent); !ok {
			http.Redirect(w, r, s.loginUrl, http.StatusFound)
		}
	}
}

// CheckSessionRest is a wrapper that returns a 403 Forbidden if no valid session id is found
func (s *Cache[S]) CheckSessionRest(parent http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok := s.CallHandlerWithData(w, r, parent); !ok {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	}
}

// LoginHandler is a handler that does the login.
// The given template is used to render the login page.
// It needs to contain a form with the fields username and password.
// If the login is successful a cookie with the session id is set and
// the user is redirected to /.
func (s *Cache[S]) LoginHandler(loginTemp *template.Template) http.HandlerFunc {
	if loginTemp == nil {
		panic("login template is nil")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		if r.Method == http.MethodPost {
			user := r.FormValue("username")
			pass := r.FormValue("password")

			var id string
			if id, err = s.CreateSessionId(user, pass); err == nil {
				http.SetCookie(w, &http.Cookie{Value: id, Name: "id"})
				http.Redirect(w, r, s.mainUrl, http.StatusFound)
				return
			}
		}
		err = loginTemp.Execute(w, err)
		if err != nil {
			log.Println(err)
		}
	}
}

// LogoutHandler is a handler that does the logout.
// The given template is used to render the logout confirmation page.
// The cookie with the session id is deleted.
func (s *Cache[S]) LogoutHandler(logoutTemp *template.Template) http.HandlerFunc {
	if logoutTemp == nil {
		panic("logout template is nil")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("id"); err == nil {
			id := c.Value
			if se := s.getSession(id); se != nil {
				se.mutex.Lock()
				defer se.mutex.Unlock()
				err := se.persist.Save(se.data)
				if err != nil {
					log.Println(err)
				}
				delete(s.sessions, id)
			}
			http.SetCookie(w, &http.Cookie{Value: "", Name: "id", Expires: time.Now().Add(-time.Hour)})
		}
		err := logoutTemp.Execute(w, nil)
		if err != nil {
			log.Println(err)
		}
	}
}

// RegisterHandler is th handler to handle the registration.
// The given template is used to render the registration page.
// It needs to contain a form with the fields username, password and password2.
func (s *Cache[S]) RegisterHandler(registerTemp *template.Template) http.HandlerFunc {
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
			if id, err = s.registerUser(user, pass, pass2); err == nil {
				http.SetCookie(w, &http.Cookie{Value: id, Name: "id"})
				http.Redirect(w, r, s.mainUrl, http.StatusFound)
				return
			}
		}
		err = registerTemp.Execute(w, err)
		if err != nil {
			log.Println(err)
		}
	}
}
