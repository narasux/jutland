package metadata

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

type rawMissionMetadata struct {
	Name                string                          `json:"name"`
	DisplayName         string                          `json:"displayName"`
	Category            string                          `json:"category"`
	DisplayNameEn       string                          `json:"displayNameEn"`
	DisplayNameRu       string                          `json:"displayNameRu"`
	DisplayNameJa       string                          `json:"displayNameJa"`
	InitFunds           int64                           `json:"initFunds"`
	InitCameraPos       [2]int                          `json:"initCameraPos"`
	MapName             string                          `json:"mapName"`
	MaxShipCount        int                             `json:"maxShipCount"`
	Description         string                          `json:"description"`
	DescriptionEn       string                          `json:"descriptionEn"`
	DescriptionRu       string                          `json:"descriptionRu"`
	DescriptionJa       string                          `json:"descriptionJa"`
	InitShips           []rawInitShipMetadata           `json:"initShips"`
	InitReinforcePoints []rawInitReinforcePointMetadata `json:"initReinforcePoints"`
	InitOilPlatforms    []rawInitOilPlatformMetadata    `json:"initOilPlatforms"`
	InitAirfields       []rawInitAirfieldMetadata       `json:"initAirfields"`
}

func normalizeMissionCategory(raw string) (MissionCategory, error) {
	category := MissionCategory(raw)
	if category == "" {
		return MissionCategoryClassic, nil
	}
	if category != MissionCategoryClassic && category != MissionCategoryTest {
		return "", fmt.Errorf("unknown mission category %q", raw)
	}
	return category, nil
}

type rawInitShipMetadata struct {
	Name         string `json:"name"`
	Pos          [2]int `json:"pos"`
	Rotation     int    `json:"rotation"`
	BelongPlayer string `json:"belongPlayer"`
}

type rawInitReinforcePointMetadata struct {
	Pos               [2]int   `json:"pos"`
	Rotation          int      `json:"rotation"`
	RallyPos          [2]int   `json:"rallyPos"`
	BelongPlayer      string   `json:"belongPlayer"`
	MaxOncomingShip   int      `json:"maxOncomingShip"`
	ProvidedShipNames []string `json:"providedShipNames"`
}

type rawInitOilPlatformMetadata struct {
	Pos    [2]int `json:"pos"`
	Radius int    `json:"radius"`
	Yield  int    `json:"yield"`
}

type rawInitAirfieldMetadata struct {
	// Pos 机场中心浮点格坐标（如 [52.4, 66.3]），对齐地图素材时不再受整数取整限制
	Pos          [2]float64 `json:"pos"`
	Rotation     int        `json:"rotation"`
	RunwayLength float64    `json:"runwayLength"`
	RunwayWidth  float64    `json:"runwayWidth"`
	BelongPlayer string     `json:"belongPlayer"`
	// Enabled 配置级开关，缺省 true；false 时该机场不生成
	Enabled     *bool   `json:"enabled"`
	TakeOffTime float64 `json:"takeOffTime"`
	// TakeoffPoints 跑道起飞点数：<=0 缺省（双点并行），1 = 单机串行起飞，
	// >=2 = 双机并行（缺省值）。用于重轰炸机单架间隔起飞。
	TakeoffPoints int                            `json:"takeoffPoints"`
	PlaneGroups   []rawInitAirfieldGroupMetadata `json:"planeGroups"`
}

type rawInitAirfieldGroupMetadata struct {
	Name     string `json:"name"`
	MaxCount int64  `json:"maxCount"`
	// InitCount 初始停放数量，<=0 时缺省为 maxCount
	InitCount int64 `json:"initCount"`
}

// isAllyPlayer 判定是否为友方玩家
func isAllyPlayer(p faction.Player) bool {
	return p == faction.HumanAlpha || p == faction.HumanBeta
}

// validateAirfieldMetadata 校验机场配置：位置必须落在陆地格、跑道长度为正、
// 机队配置合法且跑道两端不越出地图边界；校验失败直接报错退出（与甲板模板校验一致）。
func validateAirfieldMetadata(mapCfg *mapcfg.MapCfg, afMD rawInitAirfieldMetadata) {
	if mapCfg == nil {
		log.Fatalf("airfield pos %v: mission map not found", afMD.Pos)
	}
	// 浮点坐标落格判断取包含该点的格（向下取整）
	cellX, cellY := int(math.Floor(afMD.Pos[0])), int(math.Floor(afMD.Pos[1]))
	if !mapCfg.Map.IsLand(cellX, cellY) {
		log.Fatalf(
			"airfield pos (%.2f, %.2f) is not on land in map %s",
			afMD.Pos[0], afMD.Pos[1], mapCfg.Name,
		)
	}
	if afMD.RunwayLength <= 0 {
		log.Fatalf("airfield runwayLength must be positive, got %f", afMD.RunwayLength)
	}
	if afMD.RunwayWidth <= 0 {
		log.Fatalf("airfield runwayWidth must be positive, got %f", afMD.RunwayWidth)
	}
	if afMD.TakeoffPoints < 0 {
		log.Fatalf("airfield takeoffPoints must be >= 0, got %d", afMD.TakeoffPoints)
	}
	if len(afMD.PlaneGroups) == 0 {
		log.Fatalf("airfield at %v must have at least one plane group", afMD.Pos)
	}
	for _, g := range afMD.PlaneGroups {
		if g.Name == "" {
			log.Fatalf("airfield at %v has a plane group with empty name", afMD.Pos)
		}
		if g.MaxCount <= 0 {
			log.Fatalf("airfield plane group %s maxCount must be positive", g.Name)
		}
		if g.InitCount > g.MaxCount {
			log.Fatalf(
				"airfield plane group %s initCount %d exceeds maxCount %d",
				g.Name, g.InitCount, g.MaxCount,
			)
		}
	}
	// 跑道两端不得越出地图边界（网格坐标范围 [0, width-1] x [0, height-1]）
	halfLength := afMD.RunwayLength / 2
	radians := float64(afMD.Rotation) * math.Pi / 180
	for _, sign := range [][2]float64{{1, 1}, {-1, -1}} {
		endX := afMD.Pos[0] + math.Sin(radians)*halfLength*sign[0]
		endY := afMD.Pos[1] - math.Cos(radians)*halfLength*sign[1]
		if endX < 0 || endY < 0 ||
			endX > float64(mapCfg.Width-1) || endY > float64(mapCfg.Height-1) {
			log.Fatalf(
				"airfield runway at %v (rotation %d, length %.2f) is out of map bounds",
				afMD.Pos, afMD.Rotation, afMD.RunwayLength,
			)
		}
	}
}

func init() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "missions.json5"))
	if err != nil {
		log.Fatal("failed to open missions.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var misMDs []rawMissionMetadata
	if err = json5.Unmarshal(bytes, &misMDs); err != nil {
		log.Fatal("failed to unmarshal missions.json5: ", err)
	}

	missionMetadata = make(map[string]MissionMetadata)
	missionOrder = make([]string, 0, len(misMDs))

	for _, md := range misMDs {
		category, categoryErr := normalizeMissionCategory(md.Category)
		if categoryErr != nil {
			log.Fatalf("invalid category for mission %q: %v", md.Name, categoryErr)
		}
		// 战舰
		initShips := []InitShipMetadata{}
		for _, shipMD := range md.InitShips {
			initShips = append(initShips, InitShipMetadata{
				ShipName:     shipMD.Name,
				Pos:          objPos.New(shipMD.Pos[0], shipMD.Pos[1]),
				Rotation:     float64(shipMD.Rotation),
				BelongPlayer: faction.Player(shipMD.BelongPlayer),
			})
		}
		// 增援点
		initReinforcePoints := []InitReinforcePointMetadata{}
		for _, rpMD := range md.InitReinforcePoints {
			initReinforcePoints = append(initReinforcePoints, InitReinforcePointMetadata{
				Pos:               objPos.New(rpMD.Pos[0], rpMD.Pos[1]),
				Rotation:          float64(rpMD.Rotation),
				RallyPos:          objPos.New(rpMD.RallyPos[0], rpMD.RallyPos[1]),
				BelongPlayer:      faction.Player(rpMD.BelongPlayer),
				MaxOncomingShip:   rpMD.MaxOncomingShip,
				ProvidedShipNames: rpMD.ProvidedShipNames,
			})
		}
		// 油井
		initOilPlatforms := []InitOilPlatformMetadata{}
		for _, opMD := range md.InitOilPlatforms {
			initOilPlatforms = append(initOilPlatforms, InitOilPlatformMetadata{
				Pos:    objPos.New(opMD.Pos[0], opMD.Pos[1]),
				Radius: opMD.Radius,
				Yield:  opMD.Yield,
			})
		}
		// 陆地机场（配置级开关 enabled 缺省 true，false 时不生成）
		initAirfields := []InitAirfieldMetadata{}
		mapCfg := mapcfg.GetByName(md.MapName)
		for _, afMD := range md.InitAirfields {
			if afMD.Enabled != nil && !*afMD.Enabled {
				continue
			}
			validateAirfieldMetadata(mapCfg, afMD)
			groups := []InitAirfieldGroupMetadata{}
			for _, gMD := range afMD.PlaneGroups {
				initCount := gMD.InitCount
				if initCount <= 0 {
					initCount = gMD.MaxCount
				}
				groups = append(groups, InitAirfieldGroupMetadata{
					Name: gMD.Name, MaxCount: gMD.MaxCount, InitCount: initCount,
				})
			}
			initAirfields = append(initAirfields, InitAirfieldMetadata{
				Pos:           objPos.NewR(afMD.Pos[0], afMD.Pos[1]),
				Rotation:      float64(afMD.Rotation),
				RunwayLength:  afMD.RunwayLength,
				RunwayWidth:   afMD.RunwayWidth,
				BelongPlayer:  faction.Player(afMD.BelongPlayer),
				TakeOffTime:   afMD.TakeOffTime,
				TakeoffPoints: afMD.TakeoffPoints,
				PlaneGroups:   groups,
			})
		}
		// 统计计算
		allyShips, enemyShips := 0, 0
		for _, s := range initShips {
			if isAllyPlayer(s.BelongPlayer) {
				allyShips++
			} else {
				enemyShips++
			}
		}
		allyReinforce, enemyReinforce, totalSlots := 0, 0, 0
		for _, rp := range initReinforcePoints {
			if isAllyPlayer(rp.BelongPlayer) {
				allyReinforce++
			} else {
				enemyReinforce++
			}
			totalSlots += rp.MaxOncomingShip
		}
		// 元数据
		missionMetadata[md.Name] = MissionMetadata{
			Name:        md.Name,
			DisplayName: md.DisplayName,
			Category:    category,
			displayNames: map[i18n.Language]string{
				i18n.LanguageZhHans:   md.DisplayName,
				i18n.LanguageEnglish:  md.DisplayNameEn,
				i18n.LanguageRussian:  md.DisplayNameRu,
				i18n.LanguageJapanese: md.DisplayNameJa,
			},
			MaxShipCount:  md.MaxShipCount,
			InitFunds:     md.InitFunds,
			InitCameraPos: objPos.New(md.InitCameraPos[0], md.InitCameraPos[1]),
			MapCfg:        mapCfg,
			Description:   md.Description,
			descriptions: map[i18n.Language]string{
				i18n.LanguageZhHans:   md.Description,
				i18n.LanguageEnglish:  md.DescriptionEn,
				i18n.LanguageRussian:  md.DescriptionRu,
				i18n.LanguageJapanese: md.DescriptionJa,
			},
			AllyShipCount:       allyShips,
			EnemyShipCount:      enemyShips,
			AllyReinforceCount:  allyReinforce,
			EnemyReinforceCount: enemyReinforce,
			OilPlatformCount:    len(initOilPlatforms),
			TotalReinforceSlots: totalSlots,
			InitShips:           initShips,
			InitReinforcePoints: initReinforcePoints,
			InitOilPlatforms:    initOilPlatforms,
			InitAirfields:       initAirfields,
		}
		missionOrder = append(missionOrder, md.Name)
	}
}
