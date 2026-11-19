package sim

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// SVGSeries is one line in a static SVG fallback chart.
type SVGSeries struct {
	Name   string
	Color  string
	Values []float64
}

const (
	svgAPColor    = "#2d6cdf"
	svgHEColor    = "#e07a1f"
	svgArmorColor = "#b23a48"
)

// lineChartSVG renders a minimal, dependency-free line chart.
func lineChartSVG(title string, xLabels []string, series []SVGSeries, width, height int, yMax float64) string {
	if width <= 0 {
		width = 720
	}
	if height <= 0 {
		height = 280
	}
	const left, right, top, bottom = 58.0, 24.0, 42.0, 46.0
	plotW := float64(width) - left - right
	plotH := float64(height) - top - bottom
	if yMax <= 0 {
		for _, s := range series {
			for _, v := range s.Values {
				if v > yMax {
					yMax = v
				}
			}
		}
		yMax *= 1.1
	}
	if yMax <= 0 {
		yMax = 1
	}
	var b strings.Builder
	fmt.Fprintf(
		&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-label="%s">`,
		width,
		height,
		html.EscapeString(title),
	)
	fmt.Fprintf(&b, `<title>%s</title>`, html.EscapeString(title))
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/>`, width, height)
	// plot area
	fmt.Fprintf(
		&b,
		`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#fafbfc" stroke="#d8dee6"/>`,
		left,
		top,
		plotW,
		plotH,
	)
	// y grid + labels
	for i := 0; i <= 4; i++ {
		y := top + plotH*float64(i)/4
		value := yMax * (1 - float64(i)/4)
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#e6ebf2"/>`, left, y, left+plotW, y)
		fmt.Fprintf(
			&b,
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="11" fill="#5b6675">%s</text>`,
			left-8,
			y+4,
			formatTick(value),
		)
	}
	// x labels
	if len(xLabels) > 0 {
		step := max(1, len(xLabels)/6)
		for i, label := range xLabels {
			if i%step != 0 && i != len(xLabels)-1 {
				continue
			}
			x := left
			if len(xLabels) > 1 {
				x = left + plotW*float64(i)/float64(len(xLabels)-1)
			}
			fmt.Fprintf(
				&b,
				`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" fill="#5b6675">%s</text>`,
				x,
				top+plotH+18,
				html.EscapeString(label),
			)
		}
	}
	// series
	for _, s := range series {
		if len(s.Values) == 0 {
			continue
		}
		points := make([]string, 0, len(s.Values))
		for i, v := range s.Values {
			x := left
			if len(s.Values) > 1 {
				x = left + plotW*float64(i)/float64(len(s.Values)-1)
			}
			ratio := math.Max(0, math.Min(1, v/yMax))
			y := top + plotH*(1-ratio)
			points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		fmt.Fprintf(
			&b,
			`<polyline fill="none" stroke="%s" stroke-width="2.2" points="%s"/>`,
			s.Color,
			strings.Join(points, " "),
		)
		for i, v := range s.Values {
			x := left
			if len(s.Values) > 1 {
				x = left + plotW*float64(i)/float64(len(s.Values)-1)
			}
			ratio := math.Max(0, math.Min(1, v/yMax))
			y := top + plotH*(1-ratio)
			fmt.Fprintf(
				&b,
				`<circle cx="%.1f" cy="%.1f" r="2.4" fill="%s"><title>%s: %s</title></circle>`,
				x,
				y,
				s.Color,
				html.EscapeString(s.Name),
				formatTick(v),
			)
		}
	}
	// legend
	x := left
	for _, s := range series {
		fmt.Fprintf(&b, `<line x1="%.1f" y1="20" x2="%.1f" y2="20" stroke="%s" stroke-width="3"/>`, x, x+18, s.Color)
		fmt.Fprintf(
			&b,
			`<text x="%.1f" y="24" font-size="12" fill="#334155">%s</text>`,
			x+24,
			html.EscapeString(s.Name),
		)
		x += 24 + float64(len([]rune(s.Name)))*8 + 24
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// heatmapSVG renders a class-vs-class exchange matrix fallback.
func heatmapSVG(title string, rows, cols []string, values map[string]float64, width, height int) string {
	if width <= 0 {
		width = 720
	}
	if height <= 0 {
		height = 320
	}
	const left, top, right, bottom = 62.0, 42.0, 24.0, 34.0
	cellW := (float64(width) - left - right) / float64(max(1, len(cols)))
	cellH := (float64(height) - top - bottom) / float64(max(1, len(rows)))
	var b strings.Builder
	fmt.Fprintf(
		&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-label="%s">`,
		width,
		height,
		html.EscapeString(title),
	)
	fmt.Fprintf(&b, `<title>%s</title>`, html.EscapeString(title))
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/>`, width, height)
	for i, col := range cols {
		x := left + cellW*(float64(i)+0.5)
		fmt.Fprintf(
			&b,
			`<text x="%.1f" y="24" text-anchor="middle" font-size="12" fill="#334155">%s</text>`,
			x,
			html.EscapeString(col),
		)
	}
	for j, row := range rows {
		y := top + cellH*(float64(j)+0.5) + 4
		fmt.Fprintf(
			&b,
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="12" fill="#334155">%s</text>`,
			left-8,
			y,
			html.EscapeString(row),
		)
		for i, col := range cols {
			v := values[row+"|"+col]
			x := left + cellW*float64(i)
			yy := top + cellH*float64(j)
			fmt.Fprintf(
				&b,
				`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" stroke="#ffffff" stroke-width="2"><title>%s vs %s: %.2f</title></rect>`,
				x,
				yy,
				cellW,
				cellH,
				exchangeColor(v),
				html.EscapeString(row),
				html.EscapeString(col),
				v,
			)
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" fill="#1f2937">%.2f</text>`,
				x+cellW/2, yy+cellH/2+4, v)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func exchangeColor(v float64) string {
	// 1.0 is green; deviation towards 0.6 or 1.4 goes red.
	d := math.Min(1, math.Abs(v-1)/0.4)
	r := int(46 + d*190)
	g := int(160 - d*100)
	bl := int(80 - d*40)
	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}

func formatTick(v float64) string {
	if math.Abs(v) >= 1000 {
		return fmt.Sprintf("%.0f", v)
	}
	if math.Abs(v) >= 10 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}
