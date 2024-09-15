package instruction

import "fmt"

// GenInstrUid 生成指令 UID（不需要实际初始化指令）
func GenInstrUid(instrName, objUid string) string {
	return fmt.Sprintf("%s-%s", instrName, objUid)
}
