package drawer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 战报详情里的伤害数字要跟随暴击档位着色：普通白、三倍黄、十倍红。
func TestBattleReportEventSegmentsColorsCriticalDamage(t *testing.T) {
	cases := []struct {
		name     string
		critical objBullet.CriticalType
		want     any
	}{
		{name: "normal", critical: objBullet.CriticalTypeNone, want: battleReportValueClr},
		{name: "threeTimes", critical: objBullet.CriticalTypeThreeTimes, want: colorx.Yellow},
		{name: "tenTimes", critical: objBullet.CriticalTypeTenTimes, want: colorx.Red},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := state.BattleLogEntry{
				Tick:      60,
				ActorName: "iowa", ActorType: "battleship",
				TargetName: "yamato", TargetType: "battleship",
				Action: state.BattleLogShipHit,
				Count:  1,
				Hit: objUnit.HitInfo{
					WeaponType: "shell",
					Damage:     123,
					Critical:   tc.critical,
				},
			}
			segments := battleReportEventSegments(entry, faction.HumanAlpha)
			require.NotContains(t, battleReportSegmentsText(segments), battleReportDamageMarker)

			found := false
			for _, segment := range segments {
				if !strings.Contains(segment.Text, "123") {
					continue
				}
				require.Equal(t, tc.want, segment.Color)
				found = true
			}
			require.True(t, found, "伤害数字应出现在某一段里")
		})
	}
}

// 非战斗损失行要写清是哪一方的飞机，而不是只留一个「—」。
func TestBattleReportPlaneLossRowShowsSide(t *testing.T) {
	entry := state.BattleLogEntry{
		Tick:          60,
		TargetPlayer:  faction.HumanAlpha,
		TargetName:    "F4F-3",
		TargetType:    "fighter",
		TargetIsPlane: true,
		Action:        state.BattleLogPlaneLost,
		Count:         1,
	}
	own := battleReportEventSegments(entry, faction.HumanAlpha)
	ownText := battleReportSegmentsText(own)
	require.Contains(t, ownText, "我方")
	require.NotContains(t, ownText, "—")

	entry.TargetPlayer = faction.ComputerAlpha
	enemyText := battleReportSegmentsText(battleReportEventSegments(entry, faction.HumanAlpha))
	require.Contains(t, enemyText, "敌方")
}
