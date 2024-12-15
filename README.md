# Jutland

[English](./README_EN.md)

怒海激战（Jutland）是一款 2D 海战即时策略类游戏，基于 golang 游戏引擎 ebiten 实现。

> 目前该项目处于不定期更新状态，待完成度较高后会有 Release，抢先体验需要自备 Golang 开发环境~

## 如何开始

### 推荐设备

目前画面兼容仅完成 16" / 27" 屏幕的测试，推荐使用这两尺寸的屏幕进行体验 :)

CPU / 显卡要求木有很高，但过低估计会卡帧（因为目前有些优化做得不太好 orz）

### 游戏指南

#### 游戏模式下

- 鼠标左键按下拖动选取某个区域，可选中该区域内的所有战舰
- 鼠标右键点击地图位置，让 **当前选中的战舰** 前往该位置
- 持续按下 <kbd>Ctrl</kbd> 进入编队模式，再按下数字 <kbd>0-9</kbd> 将当前选中的战舰进行编队
- 按下数字 <kbd>0-9</kbd> 快速选中已经编组的舰队，若某支舰队已被选中，按下编队键会移动相机到舰队位置
- 若 **选中的战舰** 处于静止状态，按下 <kbd>X</kbd> 键散开（适用于战舰重叠的情况）
- 按下 <kbd>Q</kbd> 键，如果任意选中战舰任意武器被禁用，则启用所有，否则禁用所有
- 按下 <kbd>W</kbd> 键，如果任意选中战舰任意 **主炮** 被禁用，则启用所有，否则禁用所有
- 按下 <kbd>E</kbd> 键，如果任意选中战舰任意 **副炮** 被禁用，则启用所有，否则禁用所有
- 按下 <kbd>R</kbd> 键，如果任意选中战舰任意 **防空炮** 被禁用，则启用所有，否则禁用所有
- 按下 <kbd>T</kbd> 键，如果任意选中战舰任意 **鱼雷** 被禁用，则启用所有，否则禁用所有
- 按下 <kbd>D</kbd> 键，强制展示所有战舰的状态（HP，武器是否启用等）
- 按下 <kbd>X</kbd> 键，让 **当前选中的战舰** 往随机方向移动若干单位（分散）
- 按下 <kbd>B</kbd> 键，查看增援点信息，消耗资金与时间，召唤战舰加入战场
- 按下 <kbd>N</kbd> 键，展示弹药命中造成的伤害数值（白/黄/红：标准/三倍/十倍暴击）
- 按下 <kbd>M</kbd> 键，查看当前关卡地图的全缩略图模式（含敌我战舰对象）
- 按下 <kbd>←</kbd> <kbd>→</kbd> <kbd>↓</kbd> <kbd>↑</kbd> 键，让 **当前选中的战舰** 往对应方向移动一个单位
- 按下 <kbd>ESC</kbd> 键暂停游戏，此时按下 <kbd>Q</kbd> 退出游戏，按下 <kbd>Enter</kbd> 继续游戏

#### 全屏地图模式下

- 鼠标左键点击某个位置，可将相机中心点移动到该位置（双击可退出全屏地图模式）

#### Hacker

电脑实力太强怎么办？

1. <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>\`</kbd> 进入 Terminal
2. `help` 查看秘籍，输入 & <kbd>Enter</kbd> 教 TA 做人

是的，这个游戏内置外挂 :D

## 特别说明

本项目仅供学习，不得用于任何商业用途！

地图素材来自单机游戏《偷袭珍珠港》（2008）

声音素材来自单机游戏《钢铁的咆哮3》（2004）

战舰图片素材来自 [Tzoli](https://www.deviantart.com/tzoli/gallery)，[midnike](https://www.deviantart.com/midnike/gallery)，[shipbucket](https://www.deviantart.com/shipbucket/gallery)，[pinterest](https://jp.pinterest.com/FCZ_NN/pins/)，[bilibili](https://space.bilibili.com/650338906) 等

以上素材中如有侵权烦请联系我删除，万分感谢！

## 开发指南

### 依赖环境

- make
- go 1.22 (CGO required)

### 启动命令

```shell
make build && ./jutland
```

## 参考资料

- [Ebiten Engine](https://ebitengine.org/)
