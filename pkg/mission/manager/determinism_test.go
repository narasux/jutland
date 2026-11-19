package manager

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/observer"
	"github.com/narasux/jutland/pkg/mission/state"
)

const determinismBullet = "test/determinism-shell"

// TestHeadlessSameSeedProducesSameEventStream 验证同一 seed 的两次无头运行产生逐事件一致的 observer 流。
// 场景：两艘对射战列舰、固定时钟、SkipCommands、SyncTargeting；这是观察者/有序遍历/RNG 共同作用的端到端检查。
// 预期：两次 Sink/Loss/Shot/Hit 序列 DeepEqual，且 Shot/Hit 非空；失败说明还有 map 顺序或全局随机泄漏。
// 覆盖：fresh state 各跑 600 tick（或战局结束），seed=7；比较 Recorder 四类切片与最终状态。
func TestHeadlessSameSeedProducesSameEventStream(t *testing.T) {
	a, statusA := runDeterminismEvents(t, 7)
	b, statusB := runDeterminismEvents(t, 7)
	require.Equal(t, statusA, statusB)
	require.NotEmpty(t, a.Shots)
	require.NotEmpty(t, a.Hits)
	require.Equal(t, a, b)
}

// TestHeadlessDeterminismAcrossProcesses 验证同 seed 在独立进程中的事件哈希一致，排除进程级 map/seed 状态。
// 场景：父测试启动自身两次，子进程用 JUTLAND_DET_CHILD=1 跑同一 runDeterminismEvents(7) 并打印哈希。
// 预期：两次 DET_RESULT 的 status 与 sha256 完全相同；不一致说明仍有进程级随机/顺序泄漏。
// 覆盖：os.Executable + exec.Command 两次，解析子进程 DET_RESULT 行；子进程只执行本测试。
func TestHeadlessDeterminismAcrossProcesses(t *testing.T) {
	if os.Getenv("JUTLAND_DET_CHILD") == "1" {
		rec, status := runDeterminismEvents(t, 7)
		data, err := json.Marshal(rec)
		require.NoError(t, err)
		sum := sha256.Sum256(data)
		fmt.Printf("DET_RESULT=%s:%s\n", status, hex.EncodeToString(sum[:]))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}
	results := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		args := []string{exe, "-test.run=^TestHeadlessDeterminismAcrossProcesses$"}
		// macOS test binaries initialize Ebiten/GLFW even for headless logic; wake
		// the display for the child so a locked screen does not turn this into a
		// window-server test. Linux/CI runs the binary directly.
		if runtime.GOOS == "darwin" {
			if _, lookErr := exec.LookPath("caffeinate"); lookErr == nil {
				args = append([]string{"-u", "-t", "20"}, args...)
				args = append([]string{"caffeinate"}, args...)
			}
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(), "JUTLAND_DET_CHILD=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child %d failed: %v\n%s", i, err, out)
		}
		line := ""
		for _, candidate := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(candidate, "DET_RESULT=") {
				line = candidate
			}
		}
		if line == "" {
			t.Fatalf("child %d produced no DET_RESULT:\n%s", i, out)
		}
		results = append(results, line)
	}
	if results[0] != results[1] {
		t.Fatalf("cross-process mismatch:\n%s\n%s", results[0], results[1])
	}
}

func runDeterminismEvents(t *testing.T, seed int64) (*observer.Recorder, state.MissionStatus) {
	t.Helper()
	old := config.G
	config.G = config.NewDefaultGameSettings()
	defer func() { config.G = old }()

	objBullet.Map[determinismBullet] = &objBullet.Bullet{
		Name: determinismBullet, Type: objBullet.TypeShell, Diameter: 127,
		Damage: 25, CriticalRate: 0,
	}
	defer delete(objBullet.Map, determinismBullet)

	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(seed)))
	restoreClock := objUnit.SetNowMillis(func() int64 { return 1_700_000_000_000 })
	defer restoreRand()
	defer restoreClock()
	objUnit.SetSimTick(0)

	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"alpha": newDeterminismShip("alpha", faction.HumanAlpha, 10),
				"beta":  newDeterminismShip("beta", faction.ComputerAlpha, 20),
			},
			Planes:            map[string]*objUnit.Plane{},
			ShipUidGenerators: map[faction.Player]*objUnit.ShipUidGenerator{},
		},
		UI: state.MissionUIState{GameMarks: map[objMark.ID]*objMark.Mark{}},
	}
	ms.Stats = state.NewBattleStats()

	m := &MissionManager{state: ms, instructionSet: NewInstructionSet()}
	m.SetSimulationOptions(SimulationOptions{Headless: true, SyncTargeting: true, SkipCommands: true})
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)

	for tick := 0; tick < 600; tick++ {
		status, err := m.UpdateHeadless()
		require.NoError(t, err)
		if status != state.MissionRunning {
			break
		}
	}
	return recorder, ms.Core.MissionStatus
}

func newDeterminismShip(uid string, player faction.Player, x float64) *objUnit.BattleShip {
	gun := &objUnit.Gun{
		Name: "determinism-gun", BulletName: determinismBullet,
		BulletCount: 1, Range: 20, BulletSpread: 5, BulletSpeed: 0.2,
		AntiShip: true, PosPercent: 0,
		LeftFiringArc:  objUnit.FiringArc{Start: 0, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 360},
	}
	return &objUnit.BattleShip{
		Uid: uid, Name: uid, Type: objUnit.ShipTypeBattleShip,
		Length: 100, Width: 20, CurPos: objPos.NewR(x, 10), CurRotation: 0,
		CurHP: 500, TotalHP: 500, BelongPlayer: player,
		Weapon: objUnit.ShipWeapon{
			MainGuns: []*objUnit.Gun{gun}, MaxToShipRange: 20, HasMainGun: true,
		},
	}
}
