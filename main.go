package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	_ "github.com/silbinarywolf/preferdiscretegpu"

	"github.com/narasux/jutland/pkg/game"
)

func init() {
}

func main() {
	ebiten.SetFullscreen(true)
	ebiten.SetWindowTitle("Jutland - Powered by Ebitengine")
	if err := ebiten.RunGame(game.New()); err != nil {
		log.Fatal(err)
	}
}
