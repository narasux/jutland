# Jutland

[中文](./README.md)

**Jutland** is a 2D real-time naval strategy game implemented using the Golang game engine Ebiten.

> The project is currently being updated irregularly. A release will be released after it is completed. To experience it first, you need to prepare your own Golang development environment.

## How to Start

### Recommended devices

Currently, the screen compatibility has only been tested for 16" / 27" screens. It is recommended to use these two sizes of screens for experience :)

CPU / graphics card requirements are not very high, but too low will probably cause frame jams (because some optimizations are not done well at present orz)

### Game Guide

The mission select screen has three categories: Classic, Tutorial, and Test. New players should start with Tutorial and follow the on-screen key prompts to learn basic controls, formations, submarines, aerial scouting, and reinforcements. See [docs/tutorial/README.md](docs/tutorial/README.md) for the design and config notes.

#### In-game Mode

- Press and hold the left mouse button to drag and select an area, selecting all warships within that area.
- Right-click on a location on the map to move the **currently selected warships** to that location.
- Hold <kbd>Shift</kbd> and right-click on a location on the map to send a scout plane from the **currently selected carrier / airfield** to reconnoiter it (a torpedo or dive bomber fills in when no dedicated scout is available).
- Hold down <kbd>Ctrl</kbd> to enter formation mode, then press numbers <kbd>0-9</kbd> to form a group with the currently selected warships.
- Press numbers <kbd>0-9</kbd> to quickly select an already grouped fleet. If a fleet is already selected, pressing the grouping key again will move the camera to the location of that fleet.
- If the **selected warships** are stationary, press the <kbd>X</kbd> key to disperse them (useful for overlapping ships).
- **Submarines** start submerged. With submarines selected, press the <kbd>D</kbd> key to surface or dive. A submerged boat is fully hidden and can only fire torpedoes; diving and surfacing take 8-12 seconds. You can reverse mid-transition: the last order wins, depth progress carries over and the transition speed is unchanged. The icon above the hull shows the current dive state.
  - State machine, sonar detection, depth-charge trade-offs and AI behaviour: [Submarine design (zh)](docs/submarine/README.md)
- **Ramming damage** can be adjusted from Settings ("Ramming damage"; the default follows the game settings): battleships, cruisers, destroyers and frigates then get a default bow ram scaled by type, tonnage and armor (heavier armor means more damage and less self-damage). Carriers, cargo ships, repair/hospital ships, submarines and torpedo boats do not participate. The Droplet and Giant Swordfish ignore the toggle and always ram (the Droplet can also ram aircraft). Design and values: [Ramming and collision design (zh)](docs/collision/README.md)
- **Hull collision** is on by default and can be disabled separately from Settings ("Hull collision"): surface ships of opposing sides separate by tonnage and lose speed on contact; friendly ships still pass through. It is independent of ramming damage, so hulls can block without taking damage.
- Hold <kbd>V</kbd> (View) to temporarily show the merged sonar / hydrophone coverage outline of the **selected warships**; a single selected ship also shows its detection range label. Release to hide (no light-cyan outline is drawn by default).
- Press the <kbd>B</kbd> key to view the reinforcement point information, consume funds and time, and summon warships to join the battlefield.
- Press the <kbd>M</kbd> key to view the full thumbnail mode of the current level map (including both friendly and enemy warships).
- Press the <kbd>←</kbd> <kbd>→</kbd> <kbd>↓</kbd> <kbd>↑</kbd> keys to move the **currently selected ship** one unit in the corresponding direction.
- Press the <kbd>ESC</kbd> key to pause the game. At this point, press <kbd>Q</kbd> to abandon the mission, or press <kbd>Enter</kbd> to continue the game.

#### Full-screen Map Mode

- Left-click on a location to move the camera center to that location; click the current view center to exit full-screen map mode.

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
