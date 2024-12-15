package main

import (
	"log"
	"runtime/debug"

	"github.com/hajimehoshi/ebiten/v2"
	_ "github.com/silbinarywolf/preferdiscretegpu"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/game"
)

func main() {
	// 设置 Golang GC 阈值，避免使用过多内存
	debug.SetGCPercent(50)

	ebiten.SetTPS(constants.MaxTPS)
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Jutland - Powered by Ebitengine")

	if err := ebiten.RunGame(game.New()); err != nil {
		log.Fatal(err)
	}
}
