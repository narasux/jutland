package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	_ "github.com/silbinarywolf/preferdiscretegpu"

	"github.com/narasux/jutland/pkg/game"
)

func init() {
	// 预设环境变量
	err := os.Setenv("GOGC", "50")
	if err != nil {
		log.Fatal("failed to set GOGC to 50: ", err)
	}
}

func main() {
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Jutland - Powered by Ebitengine")
	if err := ebiten.RunGame(game.New()); err != nil {
		log.Fatal(err)
	}
}
