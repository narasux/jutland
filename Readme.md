# Jutland

怒海争锋（Jutland）是一款 2D 海战即时策略类游戏，基于 golang 游戏引擎 ebiten 实现。

## 如何开始

### 游戏指南

- 鼠标左键选取某个区域内的所有战舰
- 右键点击地图位置，让选中的战舰前往
- 若选中的战舰处于静止状态，按下 X 键散开
- 按下 `W` 键 `(weapon)`，如果任意选中战舰任意武器被禁用，则启用所有，否则禁用所有
- 按下 `G` 键 `(gun)`，如果任意选中战舰任意火炮被禁用，则启用所有，否则禁用所有
- 按下 `T` 键 `(torpedo)`，如果任意选中战舰任意鱼雷被禁用，则启用所有，否则禁用所有
- 按下 `D` 键 `(display)`，强制展示所有战舰的状态（HP，武器是否启用等）

### 依赖环境

- make
- go 1.22 (CGO required)

### 服务启动命令

```shell
export GOGC=50

make build && ./jutland
```

### 调试模式

```shell
export DEBUG=true
```

## 参考资料

- [Ebiten Engine](https://ebitengine.org/)