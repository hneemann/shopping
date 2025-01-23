package item

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	historyDays           = 180
	daysShoppingHasToLast = 4
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

type HistoryEntry struct {
	ShopTime time.Time
	Quantity float64
}

type ListData struct {
	Items []*Item
	Notes string
}

type Total struct {
	Weight float64
	Volume float64
}

func (ld *ListData) NotesLines() int {
	if len(ld.Notes) > 0 {
		lines := strings.Split(ld.Notes, "\n")
		return len(lines)
	}
	return 1
}

func (ld *ListData) AddItem(item *Item) {
	id := 0
	for _, it := range ld.Items {
		if it.Id > id {
			id = it.Id
		}
	}
	item.Id = id + 1
	ld.Items = append(ld.Items, item)
	ld.createUniqueNames()
	ld.Order(rewe)
}

func (ld *ListData) DeleteItem(id int) {
	index := -1
	for i, item := range ld.Items {
		if item.Id == id {
			index = i
		}
	}
	if index >= 0 {
		log.Println("finally delete item", ld.Items[index].Name)
		ld.Items = append(ld.Items[:index], ld.Items[index+1:]...)
		ld.createUniqueNames()
		ld.Order(rewe)
	}
}

func (ld *ListData) IdValid(id int) bool {
	for _, item := range ld.Items {
		if item.Id == id {
			return true
		}
	}
	return false
}

func (ld *ListData) ItemById(id int) *Item {
	for _, item := range ld.Items {
		if item.Id == id {
			return item
		}
	}
	return nil
}

func (ld *ListData) Replace(id int, edit *Item) {
	for i, item := range ld.Items {
		if item.Id == id {
			edit.Id = item.Id
			edit.QuantityRequired = item.QuantityRequired
			ld.Items[i] = edit
		}
	}
	ld.createUniqueNames()
	ld.Order(rewe)
}

func (ld *ListData) Total() Total {
	weight := 0.0
	volume := 0.0
	for _, item := range ld.Items {
		q := item.QuantityRequired
		if q > 0 && !item.IsNotAvailable {
			weight += float64(item.Weight) * q
			volume += float64(item.Volume) * q
		}
	}
	return Total{Weight: weight / 1000, Volume: volume / 1000 / 0.87}
}

func (ld *ListData) Save(w io.Writer) error {
	err := json.NewEncoder(w).Encode(ld)
	if err != nil {
		return err
	}

	return nil
}

func (ld *ListData) ToggleInCar(id int) {
	if item := ld.ItemById(id); item != nil {
		if item.QuantityRequired > 0 {
			item.IsInCar = !item.IsInCar
			if item.IsInCar {
				item.IsNotAvailable = false
			}
			log.Println("in car:", item.Name, item.IsInCar)
		}
	}
}

func (ld *ListData) ToggleAvailable(id int) {
	if item := ld.ItemById(id); item != nil {
		if item.QuantityRequired > 0 {
			item.IsNotAvailable = !item.IsNotAvailable
			if item.IsNotAvailable {
				item.IsInCar = false
			}
			log.Println("not available:", item.Name, item.IsInCar)
		}
	}
}

func (ld *ListData) DeleteFromList(id int) {
	if item := ld.ItemById(id); item != nil {
		log.Println("deleted", item.Name)
		item.QuantityRequired = 0
		item.IsInCar = false
		item.IsNotAvailable = false
	}
}

func (ld *ListData) Paid() {
	log.Println("Paid")
	ti := time.Now()
	isItemInCar := false
	for _, item := range ld.Items {
		item.IsNotAvailable = false
		if item.QuantityRequired > 0 {
			if item.IsInCar {
				item.ShopHistory = append(item.ShopHistory, HistoryEntry{
					ShopTime: ti,
					Quantity: item.QuantityRequired,
				})
				item.QuantityRequired = 0
				item.IsInCar = false
				item.suggestedQuantityCalculated = false
			} else {
				isItemInCar = true
			}
		}
	}
	if !isItemInCar {
		ld.Notes = ""
	}
}

func (ld *ListData) SetQuantity(id int, q float64) {
	if item := ld.ItemById(id); item != nil {
		if q < 0 {
			q = 0
		}
		item.SetQuantity(q)
	}
}

func (ld *ListData) ModQuantity(id int, n float64, useUnitIncrement bool) {
	if item := ld.ItemById(id); item != nil {
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
		item.IsNotAvailable = false
	}
}

func (ld *ListData) SomethingHidden() bool {
	for _, item := range ld.Items {
		if item.QuantityRequired > 0 && (item.IsInCar || item.IsNotAvailable) {
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

func (ld *ListData) Order(c func(Category) int) {
	sort.Slice(ld.Items, func(i, j int) bool {
		return ld.Items[i].Less(ld.Items[j], c)
	})
}

func (ld *ListData) Shops() []string {
	shops := make(map[string]struct{})
	for _, item := range ld.Items {
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

func (ld *ListData) createUniqueNames() {
	names := make(map[string]*[]*Item)
	for _, item := range ld.Items {
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

func (ld *ListData) removeOldHistory() {
	cutTime := time.Now().Add(-time.Hour * 24 * historyDays)
	for _, item := range ld.Items {
		removed := 0
		for len(item.ShopHistory) > 0 {
			if item.ShopHistory[0].ShopTime.Before(cutTime) {
				item.ShopHistory = item.ShopHistory[1:]
				removed++
			} else {
				break
			}
		}
		if removed > 0 {
			log.Println("removed", removed, "old entries from", item.Name)
		}
	}
}

type Item struct {
	Id                          int
	Name                        string
	uniqueName                  string
	Shops                       []string
	QuantityRequired            float64
	IsInCar                     bool `json:"Basket"`
	IsNotAvailable              bool
	UnitDef                     string `json:"Unit"`
	unitCreated                 bool
	unitSingular                string
	unitPlural                  string
	unitSingularShort           string
	unitPluralShort             string
	Weight                      int
	WeightStr                   string
	Volume                      int
	VolumeStr                   string
	Category                    Category
	ShopHistory                 []HistoryEntry
	suggestedQuantityCalculated bool
	suggestedQuantityRequired   float64
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
	i.IsNotAvailable = false
}

func (i *Item) Less(other *Item, cat func(Category) int) bool {
	if i.Category == other.Category {
		if i.Name == other.Name {
			return i.UnitSingular() < other.UnitSingular()
		}
		return strings.ToLower(i.Name) < strings.ToLower(other.Name)
	}
	if cat != nil {
		return cat(i.Category) < cat(other.Category)
	}
	return strings.ToLower(string(i.Category)) < strings.ToLower(string(other.Category))
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

func (i *Item) IsHidden() bool {
	return i.IsInCar || i.IsNotAvailable
}

func (i *Item) Suggest() float64 {
	if !i.suggestedQuantityCalculated {
		i.suggestedQuantityCalculated = true
		if len(i.ShopHistory) > 2 {
			maxEverAdded := 0.0
			count := 0.0
			lastCount := 0.0
			for _, entry := range i.ShopHistory {
				count += lastCount
				lastCount = entry.Quantity
				if lastCount > maxEverAdded {
					maxEverAdded = lastCount
				}
			}
			first := i.ShopHistory[0].ShopTime
			last := i.ShopHistory[len(i.ShopHistory)-1].ShopTime

			timePerItem := last.Sub(first) / time.Duration(count)
			timeToPlan := time.Since(last) + time.Hour*24*daysShoppingHasToLast
			suggestion := math.Round(timeToPlan.Hours()/timePerItem.Hours() - lastCount)
			if suggestion < 0 {
				suggestion = 0
			} else if suggestion > maxEverAdded+1 {
				suggestion = maxEverAdded + 1
			}
			i.suggestedQuantityRequired = suggestion
		} else {
			i.suggestedQuantityRequired = 0
		}
	}
	return i.suggestedQuantityRequired
}

type HistoryDescription struct {
	Count int
	Last  time.Time
}

func (hd HistoryDescription) Empty() bool {
	return hd.Count == 0
}

func (i *Item) HistoryDescription() HistoryDescription {
	if i == nil || len(i.ShopHistory) == 0 {
		return HistoryDescription{}
	}
	return HistoryDescription{
		Count: len(i.ShopHistory),
		Last:  i.ShopHistory[len(i.ShopHistory)-1].ShopTime,
	}
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

func (i *Item) ShortUnit() string {
	i.createUnits()
	if i.QuantityRequired == 1 {
		return i.unitSingularShort
	}
	return i.unitPluralShort
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
	"Box":     "Boxen",
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
		i.unitSingularShort = ""
		i.unitPluralShort = ""
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
	i.unitSingularShort = shorten(i.unitSingular)
	i.unitPluralShort = shorten(i.unitPlural)
}

func shorten(s string) string {
	ts := strings.TrimSpace(s)
	if utf8.RuneCountInString(ts) <= 4 {
		return ts
	}

	lastSpace := true
	var b bytes.Buffer
	for _, r := range s {
		if unicode.IsSpace(r) {
			lastSpace = true
		} else {
			if lastSpace {
				if b.Len() > 0 {
					b.WriteRune(unicode.ToUpper(r))
				} else {
					b.WriteRune(r)
				}
			} else {
				if !(r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u' || r == 'ä' || r == 'ö' || r == 'ü') {
					b.WriteRune(r)
				}
			}
			lastSpace = false
		}
	}
	return b.String()
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

func Load(r io.Reader) (*ListData, error) {
	items := ListData{}
	err := json.NewDecoder(r).Decode(&items)
	if err != nil {
		return nil, err
	}

	items.removeOldHistory()
	items.createUniqueNames()

	return &items, nil
}
