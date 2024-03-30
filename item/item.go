package item

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"time"
)

type Category string

type CategoryList []Category

var Categories = CategoryList{
	"Obst/Gemüse", "Kühlregal", "Kuchen", "Brot", "Tee/Kaffee", "Backzutaten", "Cerealien",
	"Konserven", "Fertiggerichte", "Hygiene", "Getränke", "Tiefkühl", "Süßes", "Anderes",
}

var REWE = MapOrder(Categories...)

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
	(*items).Order(REWE)
}

func (items *Items) DeleteItem(id int) {
	index := -1
	for i, item := range *items {
		if item.Id == id {
			index = i
		}
	}
	if index >= 0 {
		*items = append((*items)[:index], (*items)[index+1:]...)
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
			items[i] = edit
		}
	}
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

func (items Items) Save(file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}

	err = json.NewEncoder(f).Encode(items)
	if err != nil {
		return err
	}

	return nil
}

func (items Items) Shopped(id int) {
	if item := items.ItemById(id); item != nil {
		if item.QuantityRequired > 0 {
			item.Basket = !item.Basket
		}
	}
}

func (items Items) Payed() {
	ti := time.Now()
	for _, item := range items {
		if item.QuantityRequired > 0 && item.Basket {
			item.ShopHistory = append(item.ShopHistory, HistoryEntry{
				ShopTime: ti,
				Quantity: item.QuantityRequired,
			})
			item.QuantityRequired = 0
			item.Basket = false
		}
	}
}

func (items Items) Delete(id int) {
	if item := items.ItemById(id); item != nil {
		item.QuantityRequired = 0
		item.Basket = false
	}
}

func (items Items) SetQuantity(id, q int) {
	if item := items.ItemById(id); item != nil {
		if q < 0 {
			q = 0
		}
		item.SetQuantity(float64(q))
	}
}

func (items Items) AddToQuantity(id, q int) {
	if item := items.ItemById(id); item != nil {
		item.QuantityRequired += float64(q)
		if item.QuantityRequired < 0 {
			item.QuantityRequired = 0
		}
		item.Basket = false
	}
}

func (items Items) ModQuantity(id, n int) float64 {
	if item := items.ItemById(id); item != nil {
		if item.Weight == 1 {
			item.QuantityRequired += float64(n) * 50
		} else {
			item.QuantityRequired += float64(n)
		}
		if item.QuantityRequired < 0 {
			item.QuantityRequired = 0
		}
		item.Basket = false
		return item.QuantityRequired
	}
	return 0
}

func (items Items) SomethingHidden() bool {
	for _, item := range items {
		if item.QuantityRequired > 0 && item.Basket {
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

type Item struct {
	Id               int
	Name             string
	QuantityRequired float64
	Basket           bool
	Unit             string
	Weight           int
	WeightStr        string
	Volume           int
	VolumeStr        string
	Category         Category
	ShopHistory      []HistoryEntry
	OutlookState     OutlookState
}

func New(name string, unit string, weight int, weightStr string, volume int, volumeStr string, category Category) *Item {
	return &Item{
		Name:      name,
		Category:  category,
		Unit:      unit,
		Weight:    weight,
		WeightStr: weightStr,
		Volume:    volume,
		VolumeStr: volumeStr,
	}
}

func (i *Item) SetQuantity(quantity float64) {
	i.QuantityRequired = quantity
	i.Basket = false
}

func (i *Item) Less(other *Item, cat func(Category) int) bool {
	if i.Category == other.Category {
		return i.Name < other.Name
	}
	if cat != nil {
		return cat(i.Category) < cat(other.Category)
	}
	return i.Category < other.Category
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

func Load(file string) (*Items, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	items := Items{}
	err = json.NewDecoder(f).Decode(&items)
	if err != nil {
		return nil, err
	}

	return &items, nil
}
