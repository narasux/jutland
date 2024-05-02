package pathx

import (
	"path/filepath"
	"runtime"
)

// GetCurPKGPath 获取当前包的目录
func GetCurPKGPath() string {
	// skip == 1 表示获取上一层函数位置
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("get current pkg's pathx failed")
	}
	return filepath.Dir(file)
}
