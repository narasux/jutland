package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

type shipCostRecord struct {
	Name        string  `json:"name"`
	Nation      string  `json:"nation"`
	Type        string  `json:"type"`
	Year        int     `json:"year"`
	FundsCost   int64   `json:"fundsCost"`
	TimeCost    int64   `json:"timeCost"`
	HullPower   int     `json:"hullPower"`
	Aviation    int     `json:"aviation"`
	Burst       int     `json:"burst"`
	Projection  int     `json:"projection"`
	EffectiveHP float64 `json:"effectiveHP"`
}

func main() {
	if len(objUnit.ShipMap) == 0 {
		fmt.Fprintln(os.Stderr, "ShipMap is empty")
		os.Exit(1)
	}

	records := make([]shipCostRecord, 0, len(objUnit.ShipMap))
	for _, ship := range objUnit.ShipMap {
		records = append(records, shipCostRecord{
			Name:        ship.Name,
			Nation:      string(ship.Nation),
			Type:        string(ship.Type),
			Year:        ship.Year,
			FundsCost:   ship.FundsCost,
			TimeCost:    ship.TimeCost,
			HullPower:   ship.CombatPower.Hull,
			Aviation:    ship.CombatPower.Aviation,
			Burst:       ship.CombatPower.Burst,
			Projection:  ship.CombatPower.Projection,
			EffectiveHP: ship.CombatPower.Details.EffectiveHP,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})

	if err := json.NewEncoder(os.Stdout).Encode(records); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
