package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

type planeCostRecord struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Nation      string  `json:"nation"`
	CombatPower int     `json:"combatPower"`
	Tonnage     float64 `json:"tonnage"`
}

func main() {
	if len(objUnit.PlaneMap) == 0 {
		fmt.Fprintln(os.Stderr, "PlaneMap is empty")
		os.Exit(1)
	}

	records := make([]planeCostRecord, 0, len(objUnit.PlaneMap))
	for _, plane := range objUnit.PlaneMap {
		records = append(records, planeCostRecord{
			Name:        plane.Name,
			Type:        string(plane.Type),
			Nation:      string(plane.Nation),
			CombatPower: plane.CombatPower.Total,
			Tonnage:     plane.Tonnage,
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
