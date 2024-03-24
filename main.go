package main

import (
	"context"
	"errors"
	"flag"
	"github.com/hneemann/shopping/item"
	"github.com/hneemann/shopping/server"
	"github.com/hneemann/shopping/session"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"
)

type DataManager struct {
	folder string
}

func (s *DataManager) CreateUser(user string, pass string) (*item.Items, error) {
	folder := filepath.Join(s.folder, user)
	if _, err := os.Stat(folder); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(folder, 0755)
			if err != nil {
				return nil, err
			}

			bycryptPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}

			userId := filepath.Join(folder, "id")

			f, err := os.Create(userId)
			if err != nil {
				return nil, err
			}

			_, err = f.Write(bycryptPass)
			if err != nil {
				return nil, err
			}
			items := item.Items{}
			return &items, nil
		} else {
			return nil, err
		}
	}
	return nil, errors.New("user already exists")
}

func (s *DataManager) CheckPassword(user string, pass string) bool {
	id := filepath.Join(s.folder, user, "id")
	f, err := os.Open(id)
	if err != nil {
		return false
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword(b, []byte(pass))
	if err != nil {
		return false
	}
	return true
}

func (s *DataManager) RestoreData(user string, pass string) (*item.Items, error) {
	if !s.CheckPassword(user, pass) {
		return nil, errors.New("wrong password")
	}

	file := filepath.Join(s.folder, user, "data.json")
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			items := item.Items{
				{Name: "Milch", QuantityRequired: 1, Unit: "l", Weight: 1000, Volume: 1000, Category: item.Cooled},
				{Name: "Butter", QuantityRequired: 1, Unit: "Stück", Weight: 250, Volume: 250, Category: item.Cooled},
				{Name: "Graubrot", QuantityRequired: 1, Unit: "Paket", Weight: 500, Volume: int(math.Round(math.Pi * 9 * 9 / 2 * 8)), Category: item.Bread},
				{Name: "Toastbrot", QuantityRequired: 1, Unit: "Paket", Weight: 500, Volume: 10 * 10 * 25, Category: item.Bread},
				{Name: "Katzenpfötchen", QuantityRequired: 1, Unit: "Paket", Weight: 175, Volume: 13 * 13 * 2, Category: item.Sweets},
				{Name: "Mini-Pizzen", QuantityRequired: 1, Unit: "Stück", Weight: 432, Volume: int(math.Round(2.8 * 28 * 23)), Category: item.Frozen},
			}
			items.Order(item.REWE)
			return &items, nil
		}
		return nil, err
	}

	return item.Load(file)
}

func (s *DataManager) PersistData(user string, items *item.Items) {
	file := filepath.Join(s.folder, user, "data.json")
	items.Save(file)
}

func main() {
	folder := flag.String("folder", "data", "data folder")
	port := flag.Int("port", 8090, "port")
	cert := flag.String("cert", "cert.pem", "certificate")
	key := flag.String("key", "key.pem", "certificate")
	debug := flag.Bool("debug", false, "starts server in debug mode")
	flag.Parse()

	sc := session.NewSessionCache[item.Items](
		&DataManager{folder: *folder},
		30*time.Minute,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/login", session.LoginHandler(sc, server.Templates.Lookup("login.html")))
	mux.HandleFunc("/register", session.RegisterHandler(sc, server.Templates.Lookup("register.html")))
	mux.HandleFunc("/", session.CheckSessionFunc(sc, server.MainHandler))
	mux.HandleFunc("/add/", session.CheckSessionFunc(sc, server.AddHandler))

	mux.HandleFunc("/listAll", session.CheckSessionFunc(sc, server.ListAllHandler))
	mux.HandleFunc("/edit/", session.CheckSessionFunc(sc, server.EditHandler))

	assetServer := http.FileServer(http.FS(server.AssetFS))
	if *debug {
		log.Println("Starting in debug mode!")
	} else {
		assetServer = Cache(assetServer)
	}
	mux.Handle("/assets/", assetServer)

	serv := &http.Server{Addr: ":" + strconv.Itoa(*port), Handler: mux}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Print("terminated")

		sc.Close()

		err := serv.Shutdown(context.Background())
		if err != nil {
			log.Println(err)
		}
		for {
			<-c
		}
	}()

	err := serv.ListenAndServeTLS(*cert, *key)
	if err != nil {
		log.Println(err)
	}
}

func Cache(parent http.Handler) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Cache-Control", "public, max-age=28800") // 8h
		parent.ServeHTTP(writer, request)
	}
}
