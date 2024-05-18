# Jutland

怒海争锋（Jutland）是一款 2D 海战即时策略类游戏，基于 golang 游戏引擎 ebiten 实现。

## 如何开始

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