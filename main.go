package main

import (
	"context"
	"flag"
	"github.com/hneemann/shopping/item"
	"github.com/hneemann/shopping/server"
	"github.com/hneemann/shopping/session"
	"github.com/hneemann/shopping/session/fileSys"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type persist struct{}

func (p persist) Load(f fileSys.FileSystem) (*item.Items, error) {
	r, err := f.Reader("data.json")
	if err != nil {
		return nil, err
	}
	defer fileSys.CloseLog(r)
	return item.Load(r)
}

func (p persist) Save(f fileSys.FileSystem, items *item.Items) error {
	w, err := f.Writer("data.json")
	if err != nil {
		return err
	}
	defer fileSys.CloseLog(w)
	return items.Save(w)
}

func main() {
	dataFolder := flag.String("folder", "data", "data folder")
	port := flag.Int("port", 8090, "port")
	cert := flag.String("cert", "cert.pem", "certificate")
	key := flag.String("key", "cert.key", "certificate")
	debug := flag.Bool("debug", false, "starts server in debug mode")
	flag.Parse()

	sc := session.NewSessionCache[item.Items](
		session.NewDataManager[item.Items](
			session.NewFileSystemFactory(*dataFolder),
			persist{}),
		30*time.Minute)
	defer sc.Close()

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
