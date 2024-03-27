package server

import (
	"embed"
	"fmt"
	"github.com/hneemann/shopping/item"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed assets/*
var AssetFS embed.FS

var Templates = template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))

var mainTemp = Templates.Lookup("main.html")
var tableTemp = Templates.Lookup("table.html")
var addTemp = Templates.Lookup("add.html")

type mainData struct {
	Items            *item.Items
	HideCart         bool
	Categories       item.CategoryList
	CategorySelected item.Category
}

func MainHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		categorySelected := item.Categories[0]
		err := mainTemp.Execute(w, mainData{
			Items:            data,
			HideCart:         false,
			Categories:       item.Categories,
			CategorySelected: categorySelected,
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func TableHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		query := r.URL.Query()
		idStr := query.Get("id")
		if idStr != "" {
			id := toInt(idStr)
			mode := query.Get("mode")
			if id >= 0 && id < len(*data) {
				switch mode {
				case "car":
					(*data).Shopped(id)
				case "del":
					(*data).Delete(id)
				case "set":
					(*data).SetQuantity(id, toInt(query.Get("q")))
				case "add":
					(*data).AddToQuantity(id, toInt(query.Get("q")))
				}
			}
		} else {
			action := query.Get("a")
			switch action {
			case "payed":
				data.Payed()
			}
		}

		err := tableTemp.Execute(w, mainData{
			Items:      data,
			HideCart:   query.Get("h") != "0",
			Categories: item.Categories,
		})
		if err != nil {
			log.Println(err)
		}
	}
}

func ListAllModHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		query := r.URL.Query()
		idStr := query.Get("id")
		if idStr != "" {
			id := toInt(idStr)
			if id >= 0 && id < len(*data) {
				q := (*data).ModQuantity(id, toInt(query.Get("n")))
				w.Write([]byte(fmt.Sprintf("%1.0f", q)))
				return
			}
		}
	}
	w.Write([]byte("-"))
}

func toFloat(str string) float64 {
	str = strings.TrimSpace(str)
	if len(str) == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(str, ",", "."), 64)
	if err != nil {
		return 0
	}
	return f
}

func toInt(str string) int {
	str = strings.TrimSpace(str)
	if len(str) == 0 {
		return 0
	}
	f, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return f
}

type addData struct {
	Name       string
	Quantity   float64
	QHidden    bool
	Categories []item.Category
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		if r.Method == http.MethodPost {
			itemName := r.FormValue("name")
			itemUnit := r.FormValue("unit")
			category := r.FormValue("category")
			quantity := toFloat(r.FormValue("quantity"))
			weight := toInt(r.FormValue("weight"))
			volume := toInt(r.FormValue("volume"))

			if len(itemName) > 0 {
				found := false
				for _, e := range *data {
					if e.Name == itemName {
						e.SetRequired(quantity)
						found = true
						break
					}
				}
				if !found {
					i := item.New(itemName, itemUnit, weight, volume, item.Category(category))
					i.SetRequired(quantity)
					*data = append(*data, i)
					(*data).Order(item.REWE)
				}
			}
		} else {
			err := addTemp.Execute(w, addData{
				Name:       "",
				Quantity:   1,
				QHidden:    false,
				Categories: item.Categories,
			})
			if err != nil {
				log.Println(err)
			}
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

var listAllTemp = Templates.Lookup("listAll.html")

func ListAllHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		err := listAllTemp.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
	}
}

var editTemp = Templates.Lookup("edit.html")

func EditHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		if r.Method == http.MethodPost {
			itemName := r.FormValue("name")
			itemUnit := r.FormValue("unit")
			category := r.FormValue("category")
			weight := toInt(r.FormValue("weight"))
			volume := toInt(r.FormValue("volume"))
			idStr := r.FormValue("id")

			id, err := strconv.Atoi(idStr)
			if err == nil || id >= 0 || id < len(*data) {
				it := (*data)[id]
				it.Name = itemName
				it.Unit = itemUnit
				it.Category = item.Category(category)
				it.Weight = weight
				it.Volume = volume
			}

			(*data).Order(item.REWE)

			http.Redirect(w, r, "/listAll", http.StatusFound)
			return
		} else {
			itemStr := r.URL.Query().Get("item")
			itemId, err := strconv.Atoi(itemStr)
			if err != nil || itemId < 0 || itemId >= len(*data) {
				http.Redirect(w, r, "/listAll", http.StatusFound)
				return
			}

			var d = struct {
				Item       *item.Item
				Id         int
				Categories []item.Category
			}{
				Item:       (*data)[itemId],
				Id:         itemId,
				Categories: item.Categories,
			}

			err = editTemp.Execute(w, d)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
