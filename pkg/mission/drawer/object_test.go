package drawer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRotatedRectangleCorners(t *testing.T) {
	tests := []struct {
		name     string
		rotation float64
		want     [4][2]float64
	}{
		{
			name:     "zero degrees keeps length on Y axis",
			rotation: 0,
			want: [4][2]float64{
				{9, 16},
				{11, 16},
				{11, 24},
				{9, 24},
			},
		},
		{
			name:     "forty-five degrees rotates clockwise",
			rotation: 45,
			want: [4][2]float64{
				{12.121320343559642, 16.464466094067262},
				{13.535533905932738, 17.878679656440358},
				{7.878679656440358, 23.535533905932738},
				{6.464466094067262, 22.121320343559642},
			},
		},
		{
			name:     "ninety degrees rotates length onto X axis",
			rotation: 90,
			want: [4][2]float64{
				{14, 19},
				{14, 21},
				{6, 21},
				{6, 19},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rotatedRectangleCorners(10, 20, 8, 2, tt.rotation)
			for idx := range got {
				assert.InDelta(t, tt.want[idx][0], got[idx][0], 1e-9)
				assert.InDelta(t, tt.want[idx][1], got[idx][1], 1e-9)
			}
		})
	}
}
