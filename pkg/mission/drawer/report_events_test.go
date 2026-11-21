package drawer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// TestBattleReportEventSegmentsColorsCriticalDamage 验证战报伤害数字按暴击档位着色：普通白、三倍黄、十倍红。
// 场景：iowa 以 shell 命中 yamato（Tick=60、Count=1、Damage=123），分别使用 CriticalTypeNone /
// ThreeTimes / TenTimes；伤害颜色由 battleReportDamageColor 决定，与浮动伤害数字保持一致。
// 预期：文本中不残留内部占位符 \x01；含 "123" 的段颜色依次为 battleReportValueClr、colorx.Yellow、colorx.Red。
// 覆盖：battleReportEventSegments 拆段 + battleReportSegmentsText 拼全文，逐档断言伤害段颜色与数字存在。
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

// TestBattleReportPlaneLossRowShowsSide 验证非战斗损失行写出损失方是「我方」还是「敌方」。
// 场景：BattleLogPlaneLost 没有攻击者（ActorName 为空），Tick=60、F4F-3（fighter）、Count=1；
// TargetPlayer 分别取 HumanAlpha（己方视角）与 ComputerAlpha。
// 预期：己方损失行含「我方」且不含无攻击者占位符「—」；敌方损失行含「敌方」。
// 覆盖：BattleLogPlaneLost 走 MsgBattleReportEventLossRow 分支、battleReportSideText 的两种阵营换算。
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

// TestBattleReportDepthChargeSinkShowsWeapon 验证深弹击沉行写出投弹舰与「深水炸弹」，不退化未知武器。
// 场景：中文语言下，gurkha（destroyer）以 TypeDepthCharge 击沉 nautilus（submarine），
// Tick=60、Count=1、单发伤害 1200、TotalDamage=2770（大于单发伤害，追加累计伤害）。
// 预期：文本含 MsgBattleReportWeaponDepthCharge 与 "gurkha"，且不出现 MsgBattleReportWeaponUnknown。
// 覆盖：battleReportWeaponText 的 TypeDepthCharge 分支、带 Actor 的 BattleLogShipSunk 击杀行与 i18n 语言切换 Cleanup。
func TestBattleReportDepthChargeSinkShowsWeapon(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageZhHans))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	entry := state.BattleLogEntry{
		Tick:         60,
		ActorPlayer:  faction.HumanAlpha,
		ActorName:    "gurkha",
		ActorType:    string(objUnit.ShipTypeDestroyer),
		TargetPlayer: faction.ComputerAlpha,
		TargetName:   "nautilus",
		TargetType:   string(objUnit.ShipTypeSubmarine),
		Action:       state.BattleLogShipSunk,
		Count:        1,
		Hit: objUnit.HitInfo{
			WeaponType: string(objBullet.TypeDepthCharge),
			Damage:     1200,
		},
		TotalDamage: 2770,
	}
	text := battleReportSegmentsText(battleReportEventSegments(entry, faction.HumanAlpha))
	require.Contains(t, text, i18n.Text(i18n.MsgBattleReportWeaponDepthCharge))
	require.Contains(t, text, "gurkha")
	require.NotContains(t, text, i18n.Text(i18n.MsgBattleReportWeaponUnknown))
}

// TestBattleReportSelfHitRow 验证撞击反作用自伤在战况详情中单独成行。
// 场景：中文语言下，Yamato 撞击后自伤 450，记录 Action=BattleLogShipSelfHit、
// Hit.WeaponType=ram、Tick=60。
// 预期：行文本含 Yamato、「自伤」与 450，且不出现「未知武器」；失败说明自伤仍只扣血不入详情。
// 覆盖：battleReportEventSegments 的 SelfHit 分支、TypeRam 武器文案与伤害片段渲染。
func TestBattleReportSelfHitRow(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageZhHans))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	entry := state.BattleLogEntry{
		Tick:         60,
		ActorPlayer:  faction.HumanAlpha,
		ActorName:    "Yamato",
		ActorType:    string(objUnit.ShipTypeBattleShip),
		TargetPlayer: faction.HumanAlpha,
		TargetName:   "Yamato",
		TargetType:   string(objUnit.ShipTypeBattleShip),
		Action:       state.BattleLogShipSelfHit,
		Count:        1,
		Hit:          objUnit.HitInfo{WeaponType: string(objBullet.TypeRam), Damage: 450},
	}
	text := battleReportSegmentsText(battleReportEventSegments(entry, faction.HumanAlpha))
	require.Contains(t, text, "Yamato")
	require.Contains(t, text, "自伤")
	require.Contains(t, text, "450")
	require.NotContains(t, text, i18n.Text(i18n.MsgBattleReportWeaponUnknown))
}
