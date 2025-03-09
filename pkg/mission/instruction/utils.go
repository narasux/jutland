package instruction

import "fmt"

// GenInstrUid 生成指令 UID（不需要实际初始化指令）
func GenInstrUid(instrName, objUid string) string {
	// FIXME ShipMove & ShipMovePath 目前需要共享一个 ID，否则会冲突（一艘战舰两个移动指令）
	//  不过这个其实在接收方（instructions）那边处理会更好，这里只是临时处理
	if instrName == NameShipMovePath {
		instrName = NameShipMove
	} else if instrName == NamePlaneMovePath {
		instrName = NamePlaneMove
	}

	return fmt.Sprintf("%s-%s", instrName, objUid)
}
