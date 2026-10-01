package drawer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	"github.com/narasux/jutland/pkg/mission/state"
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
