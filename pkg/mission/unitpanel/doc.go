// Package unitpanel 实现任务运行页底部的战舰信息与控制面板。
//
// 面板只负责展示状态和产生类型化 UI 动作；武器、舰载机与相机等领域操作
// 由 mission manager 统一执行，避免绘制层直接改变战斗对象。
package unitpanel
