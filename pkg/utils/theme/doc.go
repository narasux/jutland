// Package theme 集中定义任务运行 HUD 的两套面板（右侧 tactical sidebar 与
// 底部 unit panel）共享的视觉 token：调色板、圆角/间距几何常量与字体档位。
//
// 目的是让两个在不同 package 中手写绘制的面板使用同一套底色、描边、按钮态、
// 把手与进度色，从源头避免"两套皮肤"漂移。包内只暴露 token 与少量通用绘制
// 帮手，不引入大粒度组件抽象。
package theme
