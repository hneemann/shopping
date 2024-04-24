package item

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"sort"
	"strings"
	"time"
)

type Category string

type CategoryList []Category

var Categories = CategoryList{
	"Obst/Gemüse", "Kühlregal", "Kuchen", "Brot", "Tee/Kaffee", "Backzutaten", "Cerealien",
	"Konserven", "Fertiggerichte", "Hygiene", "Getränke", "Tiefkühl", "Süßes", "Anderes",
}

var rewe = MapOrder(Categories...)

func (cl CategoryList) Index(category Category) int {
	for i, c := range cl {
		if c == category {
			return i
		}
	}
	return len(cl) - 1
}

func (cl CategoryList) First() Category {
	return (cl)[0]
}

const (
	Cooled Category = "Cooled"
	Bread  Category = "Bread"
	Sweets Category = "Sweets"
	Frozen Category = "Frozen"
)

type OutlookState int

const (
	Off = iota
	Running
)

type HistoryEntry struct {
	ShopTime time.Time
	Quantity float64
}

type Items []*Item

type Total struct {
	Weight float64
	Volume float64
}

func (items *Items) AddItem(item *Item) {
	id := 0
	for _, it := range *items {
		if it.Id > id {
			id = it.Id
		}
	}
	item.Id = id + 1
	*items = append(*items, item)
	items.createUniqueNames()
	items.Order(rewe)
}

func (items *Items) DeleteItem(id int) {
	index := -1
	for i, item := range *items {
		if item.Id == id {
			index = i
		}
	}
	if index >= 0 {
		log.Println("finally delete item", (*items)[index].Name)
		*items = append((*items)[:index], (*items)[index+1:]...)
		items.createUniqueNames()
		items.Order(rewe)
	}
}

func (items Items) IdValid(id int) bool {
	for _, item := range items {
		if item.Id == id {
			return true
		}
	}
	return false
}

func (items *Items) ItemById(id int) *Item {
	for _, item := range *items {
		if item.Id == id {
			return item
		}
	}
	return nil
}

func (items Items) Replace(id int, edit *Item) {
	for i, item := range items {
		if item.Id == id {
			edit.Id = item.Id
			edit.QuantityRequired = item.QuantityRequired
			items[i] = edit
		}
	}
	items.createUniqueNames()
	items.Order(rewe)
}

func (items Items) Total() Total {
	weight := 0.0
	volume := 0.0
	for _, item := range items {
		q := item.QuantityRequired
		if q > 0 {
			weight += float64(item.Weight) * q
			volume += float64(item.Volume) * q
		}
	}
	return Total{Weight: weight / 1000, Volume: volume / 1000 / 0.87}
}

func (items Items) Save(w io.Writer) error {
	err := json.NewEncoder(w).Encode(items)
	if err != nil {
		return err
	}

	return nil
}

func (items Items) ToggleInCar(id int) {
	if item := items.ItemById(id); item != nil {
		if item.QuantityRequired > 0 {
			item.IsInCar = !item.IsInCar
			log.Println("in car:", item.Name, item.IsInCar)
		}
	}
}

func (items Items) DeleteFromList(id int) {
	if item := items.ItemById(id); item != nil {
		log.Println("deleted", item.Name)
		item.QuantityRequired = 0
		item.IsInCar = false
	}
}

func (items Items) Payed() {
	log.Println("Payed")
	ti := time.Now()
	for _, item := range items {
		if item.QuantityRequired > 0 && item.IsInCar {
			item.ShopHistory = append(item.ShopHistory, HistoryEntry{
				ShopTime: ti,
				Quantity: item.QuantityRequired,
			})
			item.QuantityRequired = 0
			item.IsInCar = false
		}
	}
}

func (items Items) SetQuantity(id int, q float64) {
	if item := items.ItemById(id); item != nil {
		if q < 0 {
			q = 0
		}
		item.SetQuantity(q)
	}
}

func (items Items) ModQuantity(id int, n float64, useUnitIncrement bool) {
	if item := items.ItemById(id); item != nil {
		log.Println("mod quantity", item.Name, n)
		f := 1.0
		if useUnitIncrement {
			f = item.Increment()
		}
		item.QuantityRequired += n * f
		if item.QuantityRequired < 0.001 {
			log.Println("negative quantity avoided", item.Name)
			item.QuantityRequired = 0
		}
		item.IsInCar = false
	}
}

func (items Items) SomethingHidden() bool {
	for _, item := range items {
		if item.QuantityRequired > 0 && item.IsInCar {
			return true
		}
	}
	return false
}

func MapOrder(str ...Category) func(Category) int {
	m := make(map[Category]int)
	for i, s := range str {
		m[s] = i
	}
	return func(c Category) int {
		if i, ok := m[c]; ok {
			return i
		}
		return len(m)
	}
}

func (items Items) Order(c func(Category) int) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Less(items[j], c)
	})
}

func (items *Items) Shops() []string {
	shops := make(map[string]struct{})
	for _, item := range *items {
		if item.QuantityRequired > 0 {
			if len(item.Shops) == 0 {
				shops[""] = struct{}{}
			} else {
				for _, s := range item.Shops {
					shops[s] = struct{}{}
				}
			}
		}
	}
	var result []string
	for shop := range shops {
		result = append(result, shop)
	}
	sort.Strings(result)
	return result
}

func (items *Items) createUniqueNames() {
	names := make(map[string]*[]*Item)
	for _, item := range *items {
		item.uniqueName = ""
		list := names[item.Name]
		if list == nil {
			list = &[]*Item{}
			names[item.Name] = list
		}
		*list = append(*list, item)
	}
	for _, list := range names {
		if len(*list) > 1 {
			for _, i := range *list {
				i.uniqueName = i.Name + ", " + i.UnitSingular()
			}
		}
	}
}

type Item struct {
	Id               int
	Name             string
	uniqueName       string
	Shops            []string
	QuantityRequired float64
	IsInCar          bool   `json:"Basket"`
	UnitDef          string `json:"Unit"`
	unitCreated      bool
	unitSingular     string
	unitPlural       string
	Weight           int
	WeightStr        string
	Volume           int
	VolumeStr        string
	Category         Category
	ShopHistory      []HistoryEntry
	OutlookState     OutlookState
}

func New(name string, unit string, weight int, weightStr string, volume int, volumeStr string, category Category, shops []string) *Item {
	return &Item{
		Name:      name,
		Category:  category,
		Shops:     shops,
		UnitDef:   unit,
		Weight:    weight,
		WeightStr: weightStr,
		Volume:    volume,
		VolumeStr: volumeStr,
	}
}

func (i *Item) UniqueName() string {
	if i.uniqueName != "" {
		return i.uniqueName
	}
	return i.Name
}

func (i *Item) SetQuantity(quantity float64) {
	log.Println("Set quantity", i.Name, quantity)
	i.QuantityRequired = quantity
	i.IsInCar = false
}

func (i *Item) Less(other *Item, cat func(Category) int) bool {
	if i.Category == other.Category {
		if i.Name == other.Name {
			return i.UnitSingular() < other.UnitSingular()
		}
		return i.Name < other.Name
	}
	if cat != nil {
		return cat(i.Category) < cat(other.Category)
	}
	return i.Category < other.Category
}

func (i *Item) ShopMatches(shop string) bool {
	if shop == "" || len(i.Shops) == 0 {
		return true
	}
	for _, s := range i.Shops {
		if s == shop {
			return true
		}
	}
	return false
}

func (i *Item) ShopIs(shop string) bool {
	for _, s := range i.Shops {
		if s == shop {
			return true
		}
	}
	return false
}

func (i *Item) ShopsStr() string {
	str := ""
	for _, s := range i.Shops {
		if len(str) > 0 {
			str += ", "
		}
		str += s
	}
	return str
}

func (i *Item) Suggest() float64 {
	if i.OutlookState == Off || len(i.ShopHistory) < 2 {
		return 0
	}
	count := 0.0
	pending := 0.0
	for _, entry := range i.ShopHistory {
		count += pending
		pending = entry.Quantity
	}
	first := i.ShopHistory[0].ShopTime
	last := i.ShopHistory[len(i.ShopHistory)-1].ShopTime

	timePerItem := last.Sub(first) / time.Duration(count)
	suggestion := math.Round(float64(time.Since(last)/timePerItem) - pending)
	if suggestion < 1 {
		suggestion = 0
	}
	return suggestion
}

func (i *Item) UnitSingular() string {
	i.createUnits()
	return i.unitSingular
}

func (i *Item) UnitPlural() string {
	i.createUnits()
	return i.unitPlural
}

func (i *Item) Unit() string {
	i.createUnits()
	if i.QuantityRequired == 1 {
		return i.unitSingular
	}
	return i.unitPlural
}

var unitPluralMap = map[string]string{
	"Dose":    "Dosen",
	"Packung": "Packungen",
	"Paket":   "Pakete",
	"Tüte":    "Tüten",
	"Glas":    "Gläser",
	"Stange":  "Stangen",
	"Flasche": "Flaschen",
	"Rolle":   "Rollen",
	"Tube":    "Tuben",
	"Sack":    "Säcke",
}

func (i *Item) createUnits() {
	if i.unitCreated {
		return
	}
	i.unitCreated = true

	u := strings.TrimSpace(i.UnitDef)
	if len(u) == 0 {
		i.unitSingular = ""
		i.unitPlural = ""
		return
	}
	p := strings.Index(u, ",")
	if p > 0 {
		i.unitSingular = strings.TrimSpace(u[:p])
		i.unitPlural = strings.TrimSpace(u[p+1:])
	} else {
		i.unitSingular = u
		if up, ok := unitPluralMap[i.unitSingular]; ok {
			i.unitPlural = up
		} else {
			i.unitPlural = i.unitSingular
		}
	}
}

func (i *Item) Increment() float64 {
	f := 1.0
	switch strings.ToLower(i.UnitDef) {
	case "g", "ml":
		f = 50
	case "kg", "l", "kilo":
		f = 0.5
	}
	return f
}

func Load(r io.Reader) (*Items, error) {
	items := Items{}
	err := json.NewDecoder(r).Decode(&items)
	if err != nil {
		return nil, err
	}

	items.createUniqueNames()

	return &items, nil
}
