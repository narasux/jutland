package human

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/geometry"
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
func (h *HumanInputHandler) Handle(
	_ map[string]instr.Instruction, misState *state.MissionState,
) map[string]instr.Instruction {
	instructions := map[string]instr.Instruction{}

	// 部分场景需要屏蔽用户输入
	if misState.MissionStatus != state.MissionRunning {
		return instructions
	}

	instructions = lo.Assign(instructions, h.handleShipMove(misState))
	instructions = lo.Assign(instructions, h.handleWeapon(misState))

	return instructions
}

func (h *HumanInputHandler) handleShipMove(misState *state.MissionState) map[string]instr.Instruction {
	instructions := map[string]instr.Instruction{}

	// 当前选中的战舰数量
	selectedShipCount := len(misState.SelectedShips)

	// 检查鼠标是否在某个敌方战舰上，需要显示锁定
	var lockOnEnemy *obj.BattleShip
	if selectedShipCount != 0 {
		pos := action.DetectCursorPosOnMap(misState)
		for _, ship := range misState.Ships {
			if ship.BelongPlayer == misState.CurPlayer {
				continue
			}
			if geometry.IsPointInRotatedRectangle(
				pos.RX, pos.RY,
				ship.CurPos.RX, ship.CurPos.RY,
				ship.Length/constants.MapBlockSize,
				ship.Width/constants.MapBlockSize,
				ship.CurRotation,
			) {
				lockOnEnemy = ship

				// 默认为锁定标志
				markID, markImg := obj.MarkIDLockOn, textureImg.LockOnTarget
				// 如果选中战舰中某艘已经设置该战舰为攻击目标，则应显示攻击标志而非锁定标志
				for _, shipUid := range misState.SelectedShips {
					if s, ok := misState.Ships[shipUid]; ok {
						if s.AttackTarget == lockOnEnemy.Uid {
							markID, markImg = obj.MarkIDAttack, textureImg.AttackTarget
							break
						}
					}
				}
				mark := obj.NewImgMark(markID, *pos, markImg, 2)
				misState.GameMarks[mark.ID] = mark
				break
			}
		}
	}

	// 按下鼠标右键，如果有选中战舰，则移动选中战舰到指定位置
	if pos := action.DetectMouseButtonClickOnMap(
		misState, ebiten.MouseButtonRight,
	); pos != nil && selectedShipCount != 0 {
		for _, shipUid := range misState.SelectedShips {
			// 如果是多艘战舰，则需要区分下终点位置，不要聚在一起挨揍
			targetPos := pos.Copy()
			if selectedShipCount > 1 {
				targetPos.AddRx(float64(rand.Intn(5) - 2))
				targetPos.AddRy(float64(rand.Intn(5) - 2))
			}
			// 通过 ShipMovePath 指令实现移动行为
			ship, ok := misState.Ships[shipUid]
			if !ok {
				continue
			}
			// 右键点击前往并攻击指定目标
			if lockOnEnemy != nil {
				ship.Attack(lockOnEnemy.Uid)
			}
			moveInstr := instr.NewShipMovePath(ship.Uid, ship.CurPos, targetPos)
			instructions[moveInstr.Uid()] = moveInstr
		}
		// 有战舰被选中的情况下，标记目标位置
		markID, markImg := obj.MarkIDTarget, textureImg.TargetPos
		if lockOnEnemy != nil {
			markID, markImg = obj.MarkIDAttack, textureImg.AttackTarget
		}
		mark := obj.NewImgMark(markID, *pos, markImg, 20)
		misState.GameMarks[markID] = mark
	}

	// 通过 ShipMove 指令实现移动
	handleMove := func(shipUid string, curPos obj.MapPos, dx, dy int) {
		moveInstr := instr.NewShipMove(
			shipUid,
			obj.NewMapPosR(
				curPos.RX+float64(dx),
				curPos.RY+float64(dy),
			),
		)
		instructions[moveInstr.Uid()] = moveInstr
	}

	// 随机散开，用于战舰重叠的情况（按下 X 键）
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		for _, shipUid := range misState.SelectedShips {
			// 如果战舰不是静止状态，则散开指令无效
			if misState.Ships[shipUid].CurSpeed != 0 {
				continue
			}
			// 随机散开 [-3, 3] 的范围
			dx, dy := rand.Intn(7)-3, rand.Intn(7)-3
			// 通过 ShipMove 指令实现散开行为
			handleMove(shipUid, misState.Ships[shipUid].CurPos, dx, dy)
		}
	}

	// 方向键，让选中的战舰往对应方向移动一个单位
	dx, dy := 0, 0
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		dx, dy = 0, -1
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		dx, dy = 0, 1
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		dx, dy = -1, 0
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		dx, dy = 1, 0
	}
	if dx != 0 || dy != 0 {
		for _, shipUid := range misState.SelectedShips {
			if ship, ok := misState.Ships[shipUid]; ok {
				handleMove(shipUid, ship.CurPos, dx, dy)
			}
		}
	}
	return instructions
}

// TODO 这个函数应该拆分一下，然后重复代码有点多，可以考虑复用一下
func (h *HumanInputHandler) handleWeapon(misState *state.MissionState) map[string]instr.Instruction {
	instructions := map[string]instr.Instruction{}

	// 按下 w 键，如果任意选中战舰任意武器被禁用，则启用所有，否则禁用所有
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if len(misState.SelectedShips) != 0 {
			anyWeaponDisabled := false
			for _, shipUid := range misState.SelectedShips {
				ship := misState.Ships[shipUid]
				if ship.Weapon.MainGunDisabled || ship.Weapon.SecondaryGunDisabled || ship.Weapon.TorpedoDisabled {
					anyWeaponDisabled = true
					break
				}
			}
			for _, shipUid := range misState.SelectedShips {
				if anyWeaponDisabled {
					enableWeaponInstr := instr.NewEnableWeapon(shipUid, obj.WeaponTypeAll)
					instructions[enableWeaponInstr.Uid()] = enableWeaponInstr
				} else {
					disableWeaponInstr := instr.NewDisableWeapon(shipUid, obj.WeaponTypeAll)
					instructions[disableWeaponInstr.Uid()] = disableWeaponInstr
				}
			}
		}
	}

	// 按下 e 键，如果任意选中战舰任意主炮被禁用，则启用所有，否则禁用所有
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		if len(misState.SelectedShips) != 0 {
			anyGunDisabled := false
			for _, shipUid := range misState.SelectedShips {
				ship := misState.Ships[shipUid]
				if ship.Weapon.MainGunDisabled {
					anyGunDisabled = true
					break
				}
			}
			for _, shipUid := range misState.SelectedShips {
				if anyGunDisabled {
					enableWeaponInstr := instr.NewEnableWeapon(shipUid, obj.WeaponTypeMainGun)
					instructions[enableWeaponInstr.Uid()] = enableWeaponInstr
				} else {
					disableWeaponInstr := instr.NewDisableWeapon(shipUid, obj.WeaponTypeMainGun)
					instructions[disableWeaponInstr.Uid()] = disableWeaponInstr
				}
			}
		}
	}

	// 按下 r 键，如果任意选中战舰任意副炮被禁用，则启用所有，否则禁用所有
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if len(misState.SelectedShips) != 0 {
			anyGunDisabled := false
			for _, shipUid := range misState.SelectedShips {
				ship := misState.Ships[shipUid]
				if ship.Weapon.SecondaryGunDisabled {
					anyGunDisabled = true
					break
				}
			}
			for _, shipUid := range misState.SelectedShips {
				if anyGunDisabled {
					enableWeaponInstr := instr.NewEnableWeapon(shipUid, obj.WeaponTypeSecondaryGun)
					instructions[enableWeaponInstr.Uid()] = enableWeaponInstr
				} else {
					disableWeaponInstr := instr.NewDisableWeapon(shipUid, obj.WeaponTypeSecondaryGun)
					instructions[disableWeaponInstr.Uid()] = disableWeaponInstr
				}
			}
		}
	}

	// 按下 t 键，如果任意选中战舰任意鱼雷被禁用，则启用所有，否则禁用所有
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
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
					enableWeaponInstr := instr.NewEnableWeapon(shipUid, obj.WeaponTypeTorpedo)
					instructions[enableWeaponInstr.Uid()] = enableWeaponInstr
				} else {
					disableWeaponInstr := instr.NewDisableWeapon(shipUid, obj.WeaponTypeTorpedo)
					instructions[disableWeaponInstr.Uid()] = disableWeaponInstr
				}
			}
		}
	}

	return instructions
}
