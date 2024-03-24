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
	"Obst/Gemüse", "Kühlregal", "Brot", "Backzutaten", "Cerealien", "Konserven", "Fertiggerichte", "Hygiene", "Getränke", "Tiefkühl", "Süßes", "Anderes",
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

func (items Items) Carry() (weight, volume float64) {
	for _, item := range items {
		q := float64(item.QuantityRequired)
		if q > 0 {
			weight += item.Weight * q
			volume += item.Volume * q
		}
	}
	return
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
	if id < 0 || id >= len(items) {
		return
	}
	item := items[id]
	if item.QuantityRequired > 0 {
		item.ShopHistory = append(item.ShopHistory, HistoryEntry{
			ShopTime: time.Now(),
			Quantity: item.QuantityRequired,
		})
		item.QuantityRequired = 0
	}
}

func (items Items) Delete(id int) {
	if id < 0 || id >= len(items) {
		return
	}
	items[id].QuantityRequired = 0
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
	Name             string
	QuantityRequired float64
	Unit             string
	Weight           float64
	Volume           float64
	Category         Category
	ShopHistory      []HistoryEntry
	OutlookState     OutlookState
}

func New(name string, unit string, weight float64, volume float64, category Category) *Item {
	return &Item{
		Name:     name,
		Category: category,
		Unit:     unit,
		Weight:   weight,
		Volume:   volume,
	}
}

func (i *Item) SetRequired(quantity float64) {
	i.QuantityRequired = quantity
}

func (i *Item) Shopped() {
	i.ShopHistory = append(i.ShopHistory, HistoryEntry{
		ShopTime: time.Now(),
		Quantity: i.QuantityRequired,
	})
	i.QuantityRequired = 0
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
