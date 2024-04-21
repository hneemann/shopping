package main

import (
	"context"
	"flag"
	"github.com/hneemann/shopping/item"
	"github.com/hneemann/shopping/server"
	"github.com/hneemann/shopping/session"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"
)

func CreatePersist(folder, pass string) (session.Persist[item.Items], error) {
	return &persist{file: filepath.Join(folder, "data.json"), base: filepath.Base(folder)}, nil
}

type persist struct {
	file string
	base string
}

func (p *persist) Load() (*item.Items, error) {
	log.Println("load data:", p.base)
	return item.Load(p.file)
}

func (p *persist) Save(items *item.Items) error {
	log.Println("write data:", p.base)
	return items.Save(p.file)
}

func main() {
	folder := flag.String("folder", "data", "data folder")
	port := flag.Int("port", 8090, "port")
	cert := flag.String("cert", "cert.pem", "certificate")
	key := flag.String("key", "cert.key", "certificate")
	debug := flag.Bool("debug", false, "starts server in debug mode")
	flag.Parse()

	sc := session.NewPersistSessionCache[item.Items](*folder, CreatePersist, 30*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("/login", sc.LoginHandler(server.Templates.Lookup("login.html")))
	mux.HandleFunc("/logout", sc.LogoutHandler(server.Templates.Lookup("logout.html")))
	mux.HandleFunc("/register", sc.RegisterHandler(server.Templates.Lookup("register.html")))
	mux.HandleFunc("/", sc.CheckSessionFunc(server.MainHandler))
	mux.HandleFunc("/table/", sc.CheckSessionRest(http.HandlerFunc(server.TableHandler)))
	mux.HandleFunc("/add/", sc.CheckSessionFunc(server.AddHandler))

	mux.HandleFunc("/listAll", sc.CheckSessionFunc(server.ListAllHandler))
	mux.HandleFunc("/listAllMod/", sc.CheckSessionRest(http.HandlerFunc(server.ListAllModHandler)))
	mux.HandleFunc("/edit/", sc.CheckSessionFunc(server.EditHandler))

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
		log.Print("interrupted")

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
