package ui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Braille dot bit per (sub-column 0..1, sub-row 0..3 top→bottom). A cell packs a
// 2×4 dot grid, giving the chart 2× horizontal and 4× vertical resolution over
// block characters — smooth curves from the same underlying counts.
var brailleDots = [2][4]uint8{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

type tlBucket struct {
	count int
	sev   int // worst severity in the bucket (0 normal, 1 warn, 2 error)
}

// buildBuckets bins dated log lines into `cols` time buckets spanning the log's
// own time range. ok is false when no line carries a parseable date.
func buildBuckets(content string, cols int) (buckets []tlBucket, start, end time.Time, ok bool) {
	type pt struct {
		t   time.Time
		sev int
	}
	var pts []pt
	for _, line := range strings.Split(content, "\n") {
		if t, has := parseLogTime(line); has {
			pts = append(pts, pt{t, severityClass(line)})
		}
	}
	if len(pts) == 0 || cols < 1 {
		return nil, time.Time{}, time.Time{}, false
	}

	start, end = pts[0].t, pts[0].t
	for _, p := range pts {
		if p.t.Before(start) {
			start = p.t
		}
		if p.t.After(end) {
			end = p.t
		}
	}
	span := end.Sub(start)
	if span <= 0 {
		span = time.Minute
	}

	buckets = make([]tlBucket, cols)
	for _, p := range pts {
		idx := int(float64(p.t.Sub(start)) / float64(span) * float64(cols))
		if idx < 0 {
			idx = 0
		}
		if idx >= cols {
			idx = cols - 1
		}
		buckets[idx].count++
		if p.sev > buckets[idx].sev {
			buckets[idx].sev = p.sev
		}
	}
	return buckets, start, end, true
}

func (s Styles) tlSeverityColor(sev int) lipgloss.Color {
	switch sev {
	case 2:
		return s.theme.Red
	case 1:
		return s.theme.Yellow
	default:
		return s.theme.Green
	}
}

// cellHash avalanches a cell coordinate into a well-mixed value so neighbors
// don't correlate.
func cellHash(r, c int) uint64 {
	h := uint64(uint32(r))*0x100000001b3 + 0xcbf29ce484222325
	h = (h ^ uint64(uint32(c))) * 0x100000001b3
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// sparkleLift returns how much to brighten a cell this frame (0 = none). Each
// cell gets its own phase offset from cellHash, so dots twinkle in and out on
// staggered schedules — no synchronized cohorts — and fade via a triangular
// ramp rather than flashing on/off.
func sparkleLift(r, c, frame int) float64 {
	const cycle, window = 50, 7
	off := int(cellHash(r, c) % cycle)
	phase := (frame + off) % cycle
	if phase >= window {
		return 0
	}
	d := float64(phase) / float64(window-1) // 0..1 across the window
	return 0.55 * (1 - math.Abs(2*d-1))     // peak in the middle, fade either side
}

// renderTimeline draws a single full-width dot-matrix activity chart `height`
// rows tall: each column is a bucket, its bar a stack of ⣿ dots scaled by log
// volume, colored by the worst severity in that time slice. A dotted "typical"
// guide line sits at the median of the active buckets (robust to bursts) and
// is labeled with the count + bucket width, so the y-axis is legible. One dot
// per cell keeps each point independently colorable for the sparkle.
func (s Styles) renderTimeline(content string, width, height, frame int, sparkle bool) string {
	if width < 4 || height < 2 {
		return ""
	}
	chartH := height - 2 // bottom two rows are the tick + label axis
	buckets, start, end, ok := buildBuckets(content, width)
	if !ok {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			s.dim.Render("no activity timeline — logs lack dated timestamps"))
	}

	maxCount := 1
	var active []int
	for _, b := range buckets {
		if b.count > maxCount {
			maxCount = b.count
		}
		if b.count > 0 {
			active = append(active, b.count)
		}
	}

	// Median "typical" level + the row it sits on (top of a median-height bar).
	med := median(active)
	medLevel := int(float64(med)/float64(maxCount)*float64(chartH) + 0.5)
	if medLevel < 1 {
		medLevel = 1
	}
	if medLevel > chartH {
		medLevel = chartH
	}
	guideRow := chartH - medLevel

	// Bucket width label, e.g. "typical 22 per 3m".
	span := end.Sub(start)
	if span <= 0 {
		span = time.Minute
	}
	label := fmt.Sprintf("typical %d lines per %s", med, humanizeDur(span/time.Duration(width)))
	labelW := lipgloss.Width(label)

	barCell := func(r, c, h int) string {
		col := s.tlSeverityColor(buckets[c].sev)
		if sparkle {
			if lift := sparkleLift(r, c, frame); lift > 0 {
				col = lighten(col, lift)
			}
		}
		return lipgloss.NewStyle().Foreground(col).Render("⣿")
	}

	var sb strings.Builder
	for r := 0; r < chartH; r++ {
		for c := 0; c < width; c++ {
			// Overlay the label across the left of the guide row.
			if r == guideRow && c < labelW {
				if c == 0 {
					sb.WriteString(s.logDividerLabel.Render(label)) // blue+bold, like a title
				}
				continue
			}
			b := buckets[c]
			h := int(float64(b.count)/float64(maxCount)*float64(chartH) + 0.5)
			if b.count > 0 && h == 0 {
				h = 1 // always show at least one dot where there's any activity
			}
			switch {
			case chartH-1-r < h: // within this column's bar
				sb.WriteString(barCell(r, c, h))
			case r == guideRow: // on the guide line, no bar here
				col := s.theme.Surface1
				if sparkle && ((c-frame)%7+7)%7 == 0 {
					col = lighten(col, 0.6) // a brighter dot marches along the line
				}
				sb.WriteString(lipgloss.NewStyle().Foreground(col).Render("·"))
			default:
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	sb.WriteString(s.renderAxis(start, end, width))
	return sb.String()
}

func median(xs []int) int {
	if len(xs) == 0 {
		return 1
	}
	s := append([]int(nil), xs...)
	sort.Ints(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func humanizeDur(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d >= time.Minute:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		s := int(d.Seconds())
		if s < 1 {
			s = 1
		}
		return fmt.Sprintf("%ds", s)
	}
}

// renderAxis draws a tick row (┬ marks on a rule) and a label row beneath it,
// with several evenly-spaced time labels across the span.
// renderLoadingTimeline draws an animated Braille area fill (a rolling wave,
// filled from the bottom like the real activity chart) while a log loads. It
// keeps the same height and a static tick axis below so the layout doesn't
// shift when real data arrives.
func (s Styles) renderLoadingTimeline(width, height, frame int) string {
	if width < 4 || height < 2 {
		return ""
	}
	chartH := height - 2
	dotCols := width * 2
	dotRows := chartH * 4

	bits := make([][]uint8, chartH)
	for r := range bits {
		bits[r] = make([]uint8, width)
	}
	mid := float64(dotRows) * 0.45
	amp := float64(dotRows) * 0.3
	freq := 2 * math.Pi / float64(dotCols) * 2 // ~2 cycles across the width
	const barW = 4                             // dot-columns per bar (2 cells) → chunky
	for bx := 0; bx < dotCols; bx += barW {
		// Flat-topped bar height, quantized to whole cells so each filled cell is
		// a solid block — blocky, like the real chart's columns.
		h := int((mid+amp*math.Sin(float64(bx+barW/2)*freq-float64(frame)*0.5))/4) * 4
		if h < 0 {
			h = 0
		}
		if h > dotRows {
			h = dotRows
		}
		for x := bx; x < bx+barW && x < dotCols; x++ {
			for d := 0; d < h; d++ { // fill from the bottom up
				bits[chartH-1-d/4][x/2] |= brailleDots[x%2][3-d%4]
			}
		}
	}

	style := lipgloss.NewStyle().Foreground(s.theme.Surface1) // axis grey
	var sb strings.Builder
	for r := 0; r < chartH; r++ {
		for c := 0; c < width; c++ {
			if bits[r][c] == 0 {
				sb.WriteByte(' ')
			} else {
				sb.WriteString(style.Render(string(rune(0x2800 + int(bits[r][c])))))
			}
		}
		sb.WriteByte('\n')
	}

	// Static tick axis with a blank label row (no "loading…" text), keeping the
	// 2-row axis footprint.
	ticks := []rune(strings.Repeat("─", width))
	n := width/16 + 1
	if n < 2 {
		n = 2
	}
	if n > 6 {
		n = 6
	}
	for i := 0; i < n; i++ {
		ticks[int(float64(i)/float64(n-1)*float64(width-1)+0.5)] = '┴'
	}
	sb.WriteString(s.logDividerRule.Render(string(ticks)))
	sb.WriteByte('\n')
	return sb.String()
}

func (s Styles) renderAxis(start, end time.Time, width int) string {
	span := end.Sub(start)
	n := width/16 + 1
	if n < 2 {
		n = 2
	}
	if n > 6 {
		n = 6
	}

	ticks := []rune(strings.Repeat("─", width))
	labels := []rune(strings.Repeat(" ", width))
	for i := 0; i < n; i++ {
		frac := float64(i) / float64(n-1)
		col := int(frac*float64(width-1) + 0.5)
		ticks[col] = '┴'

		t := start.Add(time.Duration(float64(span) * frac))
		last := i == n-1
		lbl := axisLabel(t, span, last, end)

		// Left tick left-aligns its label; the last tick right-aligns so it
		// never overflows the edge.
		startCol := col
		if last {
			startCol = width - len(lbl)
		}
		if startCol < 0 {
			startCol = 0
		}
		for j, r := range lbl {
			if c := startCol + j; c >= 0 && c < width {
				labels[c] = r
			}
		}
	}
	return s.logDividerRule.Render(string(ticks)) + "\n" + s.dim.Render(string(labels))
}

func axisLabel(t time.Time, span time.Duration, last bool, end time.Time) string {
	if last && time.Since(end) < 5*time.Minute {
		return "now"
	}
	lt := t.Local()
	if span <= 36*time.Hour {
		return lt.Format("15:04")
	}
	return lt.Format("Jan 2")
}
