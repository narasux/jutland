package drawer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	weaponImg "github.com/narasux/jutland/pkg/resources/images/weapon"
)

// 敌舰尾流要被迷雾藏住，否则顺着尾流就能看出看不见的舰队在哪。
func TestTrailHiddenByFog(t *testing.T) {
	vision := state.NewFactionVision(16, 16)
	vision.Stamp(5, 5, 1)
	ms := &state.MissionState{
		Core: state.MissionCoreState{FogOfWar: true},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			Visions:   map[faction.Player]*state.FactionVision{faction.HumanAlpha: vision},
		},
	}

	assert.True(t,
		trailHiddenByFog(ms, &objTrail.Trail{BelongPlayer: faction.ComputerAlpha, Pos: objPos.New(9, 9)}),
		"unseen enemy wake should be hidden")
	assert.False(t,
		trailHiddenByFog(ms, &objTrail.Trail{BelongPlayer: faction.ComputerAlpha, Pos: objPos.New(5, 5)}),
		"enemy wake inside the sight circle should still draw")
	assert.False(t,
		trailHiddenByFog(ms, &objTrail.Trail{BelongPlayer: faction.HumanAlpha, Pos: objPos.New(9, 9)}),
		"own wake is never hidden")
	assert.False(t, trailHiddenByFog(ms, &objTrail.Trail{Pos: objPos.New(9, 9)}),
		"trail without an owner stays as before")
}

func TestRotatedRectangleCorners(t *testing.T) {
	tests := []struct {
		name     string
		rotation float64
		want     [4][2]float64
	}{
		{
			name:     "zero degrees keeps length on Y axis",
			rotation: 0,
			want: [4][2]float64{
				{9, 16},
				{11, 16},
				{11, 24},
				{9, 24},
			},
		},
		{
			name:     "forty-five degrees rotates clockwise",
			rotation: 45,
			want: [4][2]float64{
				{12.121320343559642, 16.464466094067262},
				{13.535533905932738, 17.878679656440358},
				{7.878679656440358, 23.535533905932738},
				{6.464466094067262, 22.121320343559642},
			},
		},
		{
			name:     "ninety degrees rotates length onto X axis",
			rotation: 90,
			want: [4][2]float64{
				{14, 19},
				{14, 21},
				{6, 21},
				{6, 19},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rotatedRectangleCorners(10, 20, 8, 2, tt.rotation)
			for idx := range got {
				assert.InDelta(t, tt.want[idx][0], got[idx][0], 1e-9)
				assert.InDelta(t, tt.want[idx][1], got[idx][1], 1e-9)
			}
		})
	}
}

// 舰载机图标只表达状态：甲板有备机=绿，甲板空但飞机在空=黄，关闭或全灭=灰。
func TestPlaneIconStatus(t *testing.T) {
	tests := []struct {
		name     string
		aircraft objUnit.ShipAircraft
		airborne int
		want     weaponImg.WeaponStatus
	}{
		{
			name:     "甲板有备机时可以起飞",
			aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "F", CurCount: 3}}},
			want:     weaponImg.WeaponStatusLoaded,
		},
		{
			name:     "甲板已空但飞机在空或返航",
			aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "F", CurCount: 0}}},
			airborne: 6,
			want:     weaponImg.WeaponStatusReloading,
		},
		{
			name:     "甲板已空且飞机全部损失",
			aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "F", CurCount: 0}}},
			want:     weaponImg.WeaponStatusDisabled,
		},
		{
			name: "起飞被关闭时满甲板也显示禁用",
			aircraft: objUnit.ShipAircraft{
				Disable: true,
				Groups:  []objUnit.PlaneGroup{{Name: "F", CurCount: 12}},
			},
			airborne: 4,
			want:     weaponImg.WeaponStatusDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, planeIconStatus(&tt.aircraft, tt.airborne))
		})
	}
}

// 空中飞机数按归属舰船聚合，回收后飞机已从集合里移除，不需要再扣减。
func TestAirbornePlaneCounts(t *testing.T) {
	planes := map[string]*objUnit.Plane{
		"p1": {Uid: "p1", BelongShip: "a"},
		"p2": {Uid: "p2", BelongShip: "a"},
		"p3": {Uid: "p3", BelongShip: "b"},
		"p4": nil,
	}

	assert.Equal(t, map[string]int{"a": 2, "b": 1}, airbornePlaneCounts(planes))
	assert.Empty(t, airbornePlaneCounts(nil))
}
