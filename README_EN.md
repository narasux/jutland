# Jutland

[中文](./README.md)

**Jutland** is a 2D real-time naval strategy game implemented using the Golang game engine Ebiten.

> The project is currently being updated irregularly. A release will be released after it is completed. To experience it first, you need to prepare your own Golang development environment.

## How to Start

### Recommended devices

Currently, the screen compatibility has only been tested for 16" / 27" screens. It is recommended to use these two sizes of screens for experience :)

CPU / graphics card requirements are not very high, but too low will probably cause frame jams (because some optimizations are not done well at present orz)

### Game Guide

#### In-game Mode

- Press and hold the left mouse button to drag and select an area, selecting all warships within that area.
- Right-click on a location on the map to move the **currently selected warships** to that location.
- Hold down <kbd>Ctrl</kbd> to enter formation mode, then press numbers <kbd>0-9</kbd> to form a group with the currently selected warships.
- Press numbers <kbd>0-9</kbd> to quickly select an already grouped fleet. If a fleet is already selected, pressing the grouping key again will move the camera to the location of that fleet.
- If the **selected warships** are stationary, press the <kbd>X</kbd> key to disperse them (useful for overlapping ships).
- Press the <kbd>Q</kbd> key. If any weapon of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>W</kbd> key. If any **main gun** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>E</kbd> key. If any **secondary gun** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>R</kbd> key. If any **anti-aircraft gun** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>T</kbd> key. If any **torpedo** of any selected warship is disabled, all will be enabled; otherwise, all will be disabled.
- Press the <kbd>D</kbd> key to forcefully display the status of all warships (HP, whether weapons are enabled, etc.).
- Press the <kbd>X</kbd> key to move the **currently selected ship** to a random direction by a certain number of units (disperse).
- Press the <kbd>B</kbd> key to view the reinforcement point information, consume funds and time, and summon warships to join the battlefield.
- Press the <kbd>N</kbd> key to display damage values caused by ammunition hits (white/yellow/red: standard/triple/tenfold critical hits).
- Press the <kbd>M</kbd> key to view the full thumbnail mode of the current level map (including both friendly and enemy warships).
- Press the <kbd>←</kbd> <kbd>→</kbd> <kbd>↓</kbd> <kbd>↑</kbd> keys to move the **currently selected ship** one unit in the corresponding direction.
- Press the <kbd>ESC</kbd> key to pause the game. At this point, press <kbd>Q</kbd> to exit the game, or press <kbd>Enter</kbd> to continue the game.

#### Full-screen Map Mode

- Left-click on a location to move the camera center to that location (double-click to exit full-screen map mode).

#### Hacker

What to do if the computer is too powerful?

1. <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>\`</kbd> to enter Terminal
2. `help` to view cheats, input & <kbd>Enter</kbd> to teach TA how to behave

Yes, this game has built-in plug-ins :D

## Special Notice

This project is for educational purposes only and must not be used for any commercial purposes!

The map materials are sourced from the single-player game "Attack On Pearl Harbor" (2008).

The sound materials are sourced from the single-player game "Kurogane No Houkou 3" (2004).

The battleship images are sourced from [Tzoli](https://www.deviantart.com/tzoli/gallery), [midnike](https://www.deviantart.com/midnike/gallery), [shipbucket](https://www.deviantart.com/shipbucket/gallery), [pinterest](https://jp.pinterest.com/FCZ_NN/pins/), [bilibili](https://space.bilibili.com/650338906), and others.

If any of the above materials infringe upon your rights, please contact me for removal. Thank you very much!

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
