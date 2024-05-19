package human

import (
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
)

// HumanInputHandler 人类输入处理器
type HumanInputHandler struct {
	player faction.Player
}

// NewHandler ...
func NewHandler(player faction.Player) *HumanInputHandler {
	return &HumanInputHandler{player: player}
}

var _ controller.InputHandler = (*HumanInputHandler)(nil)

// Handle 处理用户输入，更新指令集
func (h *HumanInputHandler) Handle(misState *state.MissionState) map[string]instr.Instruction {
	instructions := map[string]instr.Instruction{}

	if pos := action.DetectMouseButtonClickOnMap(misState, ebiten.MouseButtonRight); pos != nil {
		selectedShipCount := len(misState.SelectedShips)
		if selectedShipCount != 0 {
			for _, shipUid := range misState.SelectedShips {
				// 如果是多艘战舰，则需要区分下终点位置，不要聚在一起挨揍 TODO 更好的分散策略？
				targetPos := pos.Copy()
				if selectedShipCount > 1 {
					targetPos.AddRx(float64(rand.Intn(7) - 3))
					targetPos.AddRy(float64(rand.Intn(7) - 3))
				}
				// 通过 ShipMove 指令实现移动行为
				instructions[fmt.Sprintf("%s-%s", shipUid, instr.NameShipMove)] = instr.NewShipMove(shipUid, targetPos)
			}
			// 有战舰被选中的情况下，标记目标位置
			misState.GameMarks[obj.MarkTypeTargetPos] = obj.NewMark(obj.MarkTypeTargetPos, *pos)
		}
	}

	// 随机散开，用于战舰重叠的情况（按下 X 键）
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyX) {
		if len(misState.SelectedShips) != 0 {
			for _, shipUid := range misState.SelectedShips {
				// 如果战舰不是静止状态，则散开指令无效
				if misState.Ships[shipUid].CurSpeed != 0 {
					continue
				}
				// 随机散开 [-2, 2] 的范围
				x, y := rand.Intn(5)-2, rand.Intn(5)-2
				// 通过 ShipMove 指令实现散开行为
				instructions[fmt.Sprintf("%s-%s", shipUid, instr.NameShipMove)] = instr.NewShipMove(
					shipUid, obj.NewMapPos(
						misState.Ships[shipUid].CurPos.MX+x,
						misState.Ships[shipUid].CurPos.MY+y,
					),
				)
			}
		}
	}

	// 按下 w 键，如果任意选中战舰任意武器被禁用，则启用所有，否则禁用所有
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyW) {
		if len(misState.SelectedShips) != 0 {
			anyWeaponDisabled := false
			for _, shipUid := range misState.SelectedShips {
				ship := misState.Ships[shipUid]
				if ship.Weapon.GunDisabled || ship.Weapon.TorpedoDisabled {
					anyWeaponDisabled = true
					break
				}
			}
			for _, shipUid := range misState.SelectedShips {
				if anyWeaponDisabled {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameEnableWeapon)
					instructions[instrKey] = instr.NewEnableWeapon(shipUid, obj.WeaponTypeAll)
				} else {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameDisableWeapon)
					instructions[instrKey] = instr.NewDisableWeapon(shipUid, obj.WeaponTypeAll)
				}
			}
		}
	}

	// 按下 g 键，如果任意选中战舰任意火炮被禁用，则启用所有，否则禁用所有
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyG) {
		if len(misState.SelectedShips) != 0 {
			anyGunDisabled := false
			for _, shipUid := range misState.SelectedShips {
				ship := misState.Ships[shipUid]
				if ship.Weapon.GunDisabled {
					anyGunDisabled = true
					break
				}
			}
			for _, shipUid := range misState.SelectedShips {
				if anyGunDisabled {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameEnableWeapon)
					instructions[instrKey] = instr.NewEnableWeapon(shipUid, obj.WeaponTypeGun)
				} else {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameDisableWeapon)
					instructions[instrKey] = instr.NewDisableWeapon(shipUid, obj.WeaponTypeGun)
				}
			}
		}
	}

	// 按下 t 键，如果任意选中战舰任意鱼雷被禁用，则启用所有，否则禁用所有
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyT) {
		if len(misState.SelectedShips) != 0 {
			anyTorpedoDisabled := false
			for _, shipUid := range misState.SelectedShips {
				ship := misState.Ships[shipUid]
				if ship.Weapon.TorpedoDisabled {
					anyTorpedoDisabled = true
					break
				}
			}
			for _, shipUid := range misState.SelectedShips {
				if anyTorpedoDisabled {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameEnableWeapon)
					instructions[instrKey] = instr.NewEnableWeapon(shipUid, obj.WeaponTypeTorpedo)
				} else {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameDisableWeapon)
					instructions[instrKey] = instr.NewDisableWeapon(shipUid, obj.WeaponTypeTorpedo)
				}
			}
		}
	}

	return instructions
}
