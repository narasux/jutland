package ship

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/narasux/jutland/pkg/config"
)

// TestPlaneImageDirectoriesMatchConfigTypes 验证 planes.json5 的每种机型都有独立图片目录，侦察机不再混在 other。
// 场景：planeTypes 列出 fighter/dive_bomber/level_bomber/torpedo_bomber/attacker/scout/other 七个目录；scout 必须有图，other 只保留无战斗行为的运输机 C-47A.png。
// 预期：七个目录全部存在且 scout 非空，other 下除 C-47A.png 外出现任何文件都报错；失败说明资源目录与 planeTypes 脱节，启动期会 log.Fatal 或图鉴缺图。
// 覆盖：用 os.Stat 检查每个机型目录，os.ReadDir 检查 scout 数量并遍历 other 文件名。
func TestPlaneImageDirectoriesMatchConfigTypes(t *testing.T) {
	for _, planeType := range planeTypes {
		if _, err := os.Stat(filepath.Join(config.ImgResBaseDir, "planes", planeType)); err != nil {
			t.Fatalf("planes.json5 的机型 %s 缺少图片目录: %s", planeType, err)
		}
	}

	scouts, err := os.ReadDir(filepath.Join(config.ImgResBaseDir, "planes", "scout"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scouts) == 0 {
		t.Fatal("scout 图片目录是空的")
	}

	others, err := os.ReadDir(filepath.Join(config.ImgResBaseDir, "planes", "other"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range others {
		if entry.Name() != "C-47A.png" {
			t.Errorf("other 目录只应保留运输机 C-47A，发现 %s", entry.Name())
		}
	}
}
