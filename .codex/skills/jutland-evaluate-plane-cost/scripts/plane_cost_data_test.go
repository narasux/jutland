package initialize

import (
	"encoding/json"
	"sort"
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// planeCostRecord 是 TestPlaneCostData 输出的单条飞机数据。
type planeCostRecord struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Nation      string  `json:"nation"`
	CombatPower int     `json:"combatPower"`
	Tonnage     float64 `json:"tonnage"`
}

// TestPlaneCostData 输出所有飞机的战力数据，供 evaluate_plane_costs.sh 解析。
// 该测试依赖 init() 中完成的资源加载与战力计算，不需要额外初始化。
func TestPlaneCostData(t *testing.T) {
	if len(objUnit.PlaneMap) == 0 {
		t.Fatal("PlaneMap is empty — init() may have failed")
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

	// 按名称排序保证输出稳定
	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})

	for _, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			t.Fatalf("failed to marshal record for %s: %v", rec.Name, err)
		}
		t.Log(string(data))
	}
}
