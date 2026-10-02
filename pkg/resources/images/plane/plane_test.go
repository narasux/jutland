package ship

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/narasux/jutland/pkg/config"
)

// 侦察机要有自己的图片目录，不能再混在 other 里；other 只留运输机等无战斗行为机型。
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
