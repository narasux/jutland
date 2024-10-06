package game

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/narasux/jutland/pkg/config"
)

// 捕获 panic 并记录日志，然后退出进程，方便开发者排查问题
// 注：ebitengine 中，Update，Draw 是运行在独立的 goroutine 中的，因此不能在 main 中捕获 panic
func recoverAndLogThenExit() {
	if r := recover(); r != nil {
		errMsgAndStack := fmt.Sprintf("%s\n%s", r, string(debug.Stack()))
		log.Println(errMsgAndStack)
		_ = os.WriteFile(filepath.Join(config.BaseDir, "jutland.log"), []byte(errMsgAndStack), 0o644)
		os.Exit(1)
	}
}
