package item

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_shorten(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"empty", args{""}, ""},
		{"simple", args{"kg"}, "kg"},
		{"simples", args{" kg "}, "kg"},
		{"normal", args{"Packung"}, "Pckng"},
		{"normal2", args{"Paket"}, "Pkt"},
		{"space", args{" große Dose "}, "grßDs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shorten(tt.args.s); got != tt.want {
				t.Errorf("shorten() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestItems_removeOldHistory(t *testing.T) {
	n := time.Now()
	tests := []struct {
		name string
		item Item
		size int
	}{
		{"empty", Item{
			ShopHistory: []HistoryEntry{},
		}, 0},
		{"no remove", Item{
			ShopHistory: []HistoryEntry{{n, 1}},
		}, 1},
		{"all", Item{
			ShopHistory: []HistoryEntry{{n.Add(-24 * time.Hour * (historyDays + 1)), 1}},
		}, 0},
		{"one", Item{
			ShopHistory: []HistoryEntry{
				{n.Add(-24 * time.Hour * (historyDays + 1)), 1},
				{n, 1},
			},
		}, 1},
		{"two", Item{
			ShopHistory: []HistoryEntry{
				{n.Add(-24 * time.Hour * (historyDays + 2)), 1},
				{n.Add(-24 * time.Hour * (historyDays + 1)), 1},
				{n, 1},
			},
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var items Items = []*Item{&tt.item}
			items.removeOldHistory()
			assert.EqualValues(t, tt.size, len(items[0].ShopHistory))
		})
	}
}
