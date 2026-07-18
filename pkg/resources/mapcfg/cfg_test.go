package mapcfg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapDataGetSupportsRectangularMaps(t *testing.T) {
	tests := []struct {
		name string
		data MapData
		x    int
		y    int
		want rune
	}{
		{name: "wide map in bounds", data: MapData{"abcd", "efgh"}, x: 3, y: 1, want: 'h'},
		{name: "wide map x out of bounds", data: MapData{"abcd", "efgh"}, x: 4, y: 1, want: ' '},
		{name: "tall map in bounds", data: MapData{"ab", "cd", "ef"}, x: 1, y: 2, want: 'f'},
		{name: "tall map x out of bounds", data: MapData{"ab", "cd", "ef"}, x: 2, y: 1, want: ' '},
		{name: "y out of bounds", data: MapData{"ab", "cd", "ef"}, x: 0, y: 3, want: ' '},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.data.Get(tt.x, tt.y))
		})
	}
}
