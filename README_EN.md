# Jutland

[中文](./README.md)

**Jutland** is a 2D real-time naval strategy game implemented using the Golang game engine Ebiten.

## How to Start

### Game Guide

#### In-game Mode

- Press and hold the left mouse button to drag and select an area, selecting all warships within that area.
- Right-click on a location on the map to move the **currently selected warships** to that location.
- Hold down <kbd>Ctrl</kbd> to enter formation mode, then press numbers <kbd>0-9</kbd> to form a group with the currently selected warships.
- Press numbers <kbd>0-9</kbd> to quickly select an already grouped fleet. If a fleet is already selected, pressing the grouping key again will move the camera to the location of that fleet.
- If the **selected warships** are stationary, press the <kbd>X</kbd> key to disperse them (useful for overlapping ships).
- Press the <kbd>W</kbd> key. If any weapon of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>E</kbd> key. If any **main gun** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>R</kbd> key. If any **secondary gun** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>T</kbd> key. If any **torpedo** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>D</kbd> key to forcefully display the status of all warships (HP, whether weapons are enabled, etc.).
- Press the <kbd>N</kbd> key to display damage values caused by ammunition hits (white/yellow/red: standard/triple/tenfold critical hits).
- Press the <kbd>M</kbd> key to view the full thumbnail mode of the current level map (including both friendly and enemy warships).
- Press the <kbd>ESC</kbd> key to pause the game. At this point, press <kbd>Q</kbd> to exit the game, or press <kbd>Enter</kbd> to continue the game.

#### Full-screen Map Mode

- Left-click on a location to move the camera center to that location (double-click to exit full-screen map mode).

## Development Guide

### Dependencies

- make
- go 1.22 (CGO required)

### Startup Command

```shell
make build && ./jutland
```

## References

- [Ebiten Engine](https://ebitengine.org/)
