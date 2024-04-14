package server

import (
	"embed"
	"fmt"
	"github.com/hneemann/parser2"
	"github.com/hneemann/parser2/funcGen"
	"github.com/hneemann/shopping/item"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed assets/*
var AssetFS embed.FS

const eps = 1e-6

var Templates = template.Must(template.New("").Funcs(map[string]any{
	"niceToStr": func(v float64) string {
		if math.Abs(math.Round(v)-v) < eps {
			return fmt.Sprintf("%d", int(v))
		}
		if math.Abs(math.Round(v*10)-v*10) < eps {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%.2f", v)
	},
}).ParseFS(templateFS, "templates/*.html"))

var mainTemp = Templates.Lookup("main.html")
var tableTemp = Templates.Lookup("table.html")
var addTemp = Templates.Lookup("add.html")

type mainData struct {
	Items            *item.Items
	HideCart         bool
	Categories       item.CategoryList
	CategorySelected item.Category
	Shop             string
	Shops            []string
}

func MainHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		categorySelected := item.Categories[0]
		err := mainTemp.Execute(w, mainData{
			Items:            data,
			HideCart:         false,
			Categories:       item.Categories,
			Shops:            data.Shops(),
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
		shop := query.Get("s")
		idStr := query.Get("id")
		if idStr != "" {
			id := toInt(idStr)
			mode := query.Get("mode")
			if data.IdValid(id) {
				switch mode {
				case "car":
					(*data).ToggleInCar(id)
				case "del":
					(*data).DeleteFromList(id)
				case "set":
					(*data).SetQuantity(id, toFloat(query.Get("q")))
				case "add":
					(*data).ModQuantity(id, toFloat(query.Get("q")), false)
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
			Shop:       shop,
			HideCart:   query.Get("h") != "0",
			Categories: item.Categories,
			Shops:      data.Shops(),
		})
		if err != nil {
			log.Println(err)
		}
	}
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

var simpleParser = funcGen.New[float64]().
	AddConstant("pi", math.Pi).
	AddSimpleOp("=", true, func(a, b float64) (float64, error) { return fromBool(a == b), nil }).
	AddSimpleOp("<", false, func(a, b float64) (float64, error) { return fromBool(a < b), nil }).
	AddSimpleOp(">", false, func(a, b float64) (float64, error) { return fromBool(a > b), nil }).
	AddSimpleOp("+", true, func(a, b float64) (float64, error) { return a + b, nil }).
	AddSimpleOp("-", false, func(a, b float64) (float64, error) { return a - b, nil }).
	AddSimpleOp("*", true, func(a, b float64) (float64, error) { return a * b, nil }).
	AddSimpleOp("/", false, func(a, b float64) (float64, error) { return a / b, nil }).
	AddSimpleOp("^", false, func(a, b float64) (float64, error) { return math.Pow(a, b), nil }).
	AddUnary("-", func(a float64) (float64, error) { return -a, nil }).
	AddSimpleFunction("sin", math.Sin).
	AddSimpleFunction("cos", math.Cos).
	AddSimpleFunction("tan", math.Tan).
	AddSimpleFunction("exp", math.Exp).
	AddSimpleFunction("ln", math.Log).
	AddSimpleFunction("sqrt", math.Sqrt).
	AddSimpleFunction("sqr", func(x float64) float64 {
		return x * x
	}).
	SetToBool(func(c float64) (bool, bool) { return c != 0, true }).
	SetNumberParser(
		parser2.NumberParserFunc[float64](
			func(n string) (float64, error) {
				return strconv.ParseFloat(n, 64)
			},
		),
	)

func fromBool(b bool) float64 {
	if b {
		return 1
	} else {
		return 0
	}
}

func toIntCalc(str string) (int, string, error) {
	if str == "" {
		return 0, "", nil
	}

	f, err := simpleParser.Generate(str)
	const format = "Fehler im Ausdruck '%s': %w"
	if err != nil {
		return 0, str, fmt.Errorf(format, str, err)
	}
	res, err := f.Eval()
	if err != nil {
		return 0, str, fmt.Errorf(format, str, err)
	}
	return int(res), str, nil
}

type addData struct {
	Name       string
	Unit       string
	Quantity   float64
	Category   string
	Weight     string
	Volume     string
	QHidden    bool
	Categories []item.Category
	Shops      []string
	Error      error
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		var itemName, itemUnit, category, shop string
		var quantity float64 = 1
		var volumeStr string
		var weightStr string
		var err error
		if r.Method == http.MethodPost {
			itemName = strings.TrimSpace(r.FormValue("name"))
			itemUnit = strings.TrimSpace(r.FormValue("unit"))
			shop = r.FormValue("shop")
			category = strings.TrimSpace(r.FormValue("category"))
			quantity = toFloat(r.FormValue("quantity"))
			var weight int
			weight, weightStr, err = toIntCalc(r.FormValue("weight"))
			if err == nil {
				var volume int
				volume, volumeStr, err = toIntCalc(r.FormValue("volume"))
				if err == nil {
					if len(itemName) > 0 {
						found := false
						for _, e := range *data {
							if e.Name == itemName && e.UnitSingular() == itemUnit {
								e.SetQuantity(quantity)
								found = true
								break
							}
						}
						if !found {
							i := item.New(itemName, itemUnit, weight, weightStr, volume, volumeStr, item.Category(category), splitShop(shop))
							i.SetQuantity(quantity)
							data.AddItem(i)
						}
						http.Redirect(w, r, "/", http.StatusFound)
						return
					}
				}
			}
		} else {
			category = r.URL.Query().Get("c")
			if category == "" {
				category = string(item.Categories[0])
			}
		}
		err = addTemp.Execute(w, addData{
			Name:       itemName,
			Unit:       itemUnit,
			Category:   category,
			Quantity:   quantity,
			Weight:     weightStr,
			Volume:     volumeStr,
			QHidden:    false,
			Categories: item.Categories,
			Shops:      data.Shops(),
			Error:      err,
		})
		if err != nil {
			log.Println(err)
		}
		return
	}
}

func splitShop(shop string) []string {
	var sl []string
	for _, s := range strings.Split(shop, ",") {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			sl = append(sl, s)
		}
	}
	return sl
}

var listAllTemp = Templates.Lookup("listAll.html")

func ListAllHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		idStr := r.URL.Query().Get("del")
		if idStr != "" {
			id, idErr := strconv.Atoi(idStr)
			if idErr == nil || id >= 0 || id < len(*data) {
				data.DeleteItem(id)
			}
		}
		err := listAllTemp.Execute(w, data)
		if err != nil {
			log.Println(err)
		}
	}
}

var listAllRowTemp = Templates.Lookup("listAllRow.html")

func ListAllModHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		query := r.URL.Query()
		idStr := query.Get("id")
		if idStr != "" {
			id := toInt(idStr)
			if data.IdValid(id) {
				data.ModQuantity(id, toFloat(query.Get("n")), true)
				err := listAllRowTemp.Execute(w, data.ItemById(id))
				if err != nil {
					log.Println(err)
				}
				return
			}
		}
	}
	w.Write([]byte("-"))
}

var editTemp = Templates.Lookup("edit.html")

func EditHandler(w http.ResponseWriter, r *http.Request) {
	if data, ok := r.Context().Value("data").(*item.Items); ok {
		var err error
		var id int
		var itemToEdit *item.Item
		if r.Method == http.MethodPost {
			idStr := r.FormValue("id")
			var idErr error
			id, idErr = strconv.Atoi(idStr)
			if idErr != nil || !data.IdValid(id) {
				http.Redirect(w, r, "/listAll", http.StatusFound)
				return
			}

			itemToEdit = &item.Item{
				Name:     strings.TrimSpace(r.FormValue("name")),
				Shops:    splitShop(r.FormValue("shop")),
				UnitDef:  strings.TrimSpace(r.FormValue("unit")),
				Category: item.Category(r.FormValue("category")),
			}

			itemToEdit.Weight, itemToEdit.WeightStr, err = toIntCalc(r.FormValue("weight"))
			if err == nil {
				itemToEdit.Volume, itemToEdit.VolumeStr, err = toIntCalc(r.FormValue("volume"))
				if err == nil {
					data.Replace(id, itemToEdit)
					http.Redirect(w, r, "/listAll#q"+strconv.Itoa(id), http.StatusFound)
					return
				}
			}
		} else {
			idStr := r.URL.Query().Get("item")
			var idErr error
			id, idErr = strconv.Atoi(idStr)
			if idErr != nil {
				http.Redirect(w, r, "/listAll", http.StatusFound)
				return
			}
			itemToEdit = data.ItemById(id)
			if itemToEdit == nil {
				http.Redirect(w, r, "/listAll", http.StatusFound)
				return
			}
		}

		if itemToEdit.WeightStr == "" && itemToEdit.Weight > 0 {
			itemToEdit.WeightStr = strconv.Itoa(itemToEdit.Weight)
		}
		if itemToEdit.VolumeStr == "" && itemToEdit.Volume > 0 {
			itemToEdit.VolumeStr = strconv.Itoa(itemToEdit.Volume)
		}
		var d = struct {
			Item       *item.Item
			Id         int
			Categories []item.Category
			Shops      []string
			Error      error
		}{
			Item:       itemToEdit,
			Id:         id,
			Categories: item.Categories,
			Shops:      data.Shops(),
			Error:      err,
		}

		err = editTemp.Execute(w, d)
		if err != nil {
			log.Println(err)
		}
	}
}
