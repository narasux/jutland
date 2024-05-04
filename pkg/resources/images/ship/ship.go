package ship

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
)

var (
	ShipDefaultZeroImg *ebiten.Image
)

func init() {
	var err error

	log.Println("loading ship image resources...")

	imgPath := "/ships/default/default_0.png"
	if ShipDefaultZeroImg, err = loader.LoadImage(imgPath); err != nil {
		log.Fatalf("missing %s: %s", imgPath, err)

	}

	log.Println("ship image resources loaded")
}
