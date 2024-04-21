package session

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path/filepath"
	"time"
	"unicode"
)

type PersistFactory[D any] func(folder, pass string) (Persist[D], error)

func NewDataManager[D any](folder string, persist PersistFactory[D]) *DataManager[D] {
	return &DataManager[D]{folder: folder, persistFactory: persist}
}

type DataManager[D any] struct {
	folder         string
	persistFactory PersistFactory[D]
}

var _ Manager[int] = &DataManager[int]{}

func (s *DataManager[D]) CreateUser(user string, pass string) (*D, error) {
	for _, r := range user {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return nil, errors.New("not a valid username")
		}
	}

	folder := filepath.Join(s.folder, user)
	if _, err := os.Stat(folder); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(folder, 0755)
			if err != nil {
				return nil, err
			}

			bcryptPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			userId := filepath.Join(folder, "id")
			err = os.WriteFile(userId, bcryptPass, 0666)
			if err != nil {
				return nil, err
			}
			var items D
			return &items, nil
		} else {
			return nil, err
		}
	}
	return nil, errors.New("user already exists")
}

func (s *DataManager[D]) CheckPassword(user string, pass string) bool {
	id := filepath.Join(s.folder, user, "id")
	b, err := os.ReadFile(id)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword(b, []byte(pass))
	if err != nil {
		return false
	}
	return true
}

func (s *DataManager[D]) CreatePersist(user, pass string) (Persist[D], error) {
	return s.persistFactory(filepath.Join(s.folder, user), pass)
}

func NewPersistSessionCache[S any](folder string, p PersistFactory[S], sessionLifeTime time.Duration) *Cache[S] {
	return NewSessionCache[S](NewDataManager(folder, p), sessionLifeTime)
}
