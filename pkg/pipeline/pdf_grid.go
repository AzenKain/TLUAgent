package pipeline

import (
	"math"
	"sort"
	"strings"

	"tluagent-web/pkg/pdf"
)

const (
	gridLineMaxThickness = 1.5
	gridLineMinLength    = 3.0
	gridSnapTolerance    = 1.5
	gridMaxLines         = 4000
)

type gridSegment struct {
	pos, from, to float64
}

type gridCell struct {
	row0, row1, col0, col1 int
	parts                  []pdf.Text
}

type gridTable struct {
	top      float64
	markdown string
}

type gridBounds struct {
	xs, ys []float64
}

// extractGridTables rebuilds ruled tables from drawn border lines and returns them with the text left outside any table.
func extractGridTables(content pdf.Content) ([]pdf.Text, []gridTable) {
	vert, horiz := collectGridSegments(content.Rect)
	if len(vert) < 2 || len(horiz) < 2 || len(vert)+len(horiz) > gridMaxLines {
		return content.Text, nil
	}

	remaining := content.Text
	var tables []gridTable
	for _, group := range groupConnectedSegments(vert, horiz) {
		bounds, ok := buildGridBounds(group.vert, group.horiz)
		if !ok {
			continue
		}
		cells := mergeGridCells(bounds, group.vert, group.horiz)
		var outside []pdf.Text
		var filled int
		for _, t := range remaining {
			if cell := locateGridCell(bounds, cells, t); cell != nil {
				cell.parts = append(cell.parts, t)
				continue
			}
			outside = append(outside, t)
		}
		for _, c := range cells {
			if len(c.parts) > 0 {
				filled++
			}
		}
		if filled < 2 {
			continue
		}
		remaining = outside
		tables = append(tables, gridTable{top: bounds.ys[0], markdown: renderGridMarkdown(bounds, cells)})
	}
	return remaining, tables
}

func collectGridSegments(rects []pdf.Rect) (vert, horiz []gridSegment) {
	seenV := map[[3]int]bool{}
	seenH := map[[3]int]bool{}
	for _, r := range rects {
		w, h := r.Max.X-r.Min.X, r.Max.Y-r.Min.Y
		switch {
		case w <= gridLineMaxThickness && h >= gridLineMinLength:
			key := [3]int{int(math.Round((r.Min.X + r.Max.X) * 5)), int(math.Round(r.Min.Y * 5)), int(math.Round(r.Max.Y * 5))}
			if !seenV[key] {
				seenV[key] = true
				vert = append(vert, gridSegment{pos: (r.Min.X + r.Max.X) / 2, from: r.Min.Y, to: r.Max.Y})
			}
		case h <= gridLineMaxThickness && w >= gridLineMinLength:
			key := [3]int{int(math.Round((r.Min.Y + r.Max.Y) * 5)), int(math.Round(r.Min.X * 5)), int(math.Round(r.Max.X * 5))}
			if !seenH[key] {
				seenH[key] = true
				horiz = append(horiz, gridSegment{pos: (r.Min.Y + r.Max.Y) / 2, from: r.Min.X, to: r.Max.X})
			}
		}
	}
	return vert, horiz
}

type segmentGroup struct {
	vert, horiz []gridSegment
}

func groupConnectedSegments(vert, horiz []gridSegment) []segmentGroup {
	n := len(vert) + len(horiz)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i, v := range vert {
		for j, h := range horiz {
			if v.pos >= h.from-gridSnapTolerance && v.pos <= h.to+gridSnapTolerance &&
				h.pos >= v.from-gridSnapTolerance && h.pos <= v.to+gridSnapTolerance {
				parent[find(i)] = find(len(vert) + j)
			}
		}
	}
	groups := map[int]*segmentGroup{}
	var order []int
	for i := 0; i < n; i++ {
		root := find(i)
		g, ok := groups[root]
		if !ok {
			g = &segmentGroup{}
			groups[root] = g
			order = append(order, root)
		}
		if i < len(vert) {
			g.vert = append(g.vert, vert[i])
		} else {
			g.horiz = append(g.horiz, horiz[i-len(vert)])
		}
	}
	out := make([]segmentGroup, 0, len(order))
	for _, root := range order {
		out = append(out, *groups[root])
	}
	return out
}

func snapPositions(values []float64) []float64 {
	sort.Float64s(values)
	var out []float64
	for _, v := range values {
		if len(out) > 0 && v-out[len(out)-1] <= gridSnapTolerance {
			continue
		}
		out = append(out, v)
	}
	return out
}

func buildGridBounds(vert, horiz []gridSegment) (gridBounds, bool) {
	xs := make([]float64, 0, len(vert))
	for _, v := range vert {
		xs = append(xs, v.pos)
	}
	ys := make([]float64, 0, len(horiz))
	for _, h := range horiz {
		ys = append(ys, h.pos)
	}
	xs = snapPositions(xs)
	ys = snapPositions(ys)
	if len(xs) < 3 || len(ys) < 3 {
		return gridBounds{}, false
	}
	for i, j := 0, len(ys)-1; i < j; i, j = i+1, j-1 {
		ys[i], ys[j] = ys[j], ys[i]
	}
	return gridBounds{xs: xs, ys: ys}, true
}

func hasVerticalBorder(vert []gridSegment, x, yLow, yHigh float64) bool {
	mid := (yLow + yHigh) / 2
	for _, v := range vert {
		if math.Abs(v.pos-x) <= gridSnapTolerance && v.from-gridSnapTolerance <= mid && v.to+gridSnapTolerance >= mid {
			return true
		}
	}
	return false
}

func hasHorizontalBorder(horiz []gridSegment, y, xLow, xHigh float64) bool {
	mid := (xLow + xHigh) / 2
	for _, h := range horiz {
		if math.Abs(h.pos-y) <= gridSnapTolerance && h.from-gridSnapTolerance <= mid && h.to+gridSnapTolerance >= mid {
			return true
		}
	}
	return false
}

func mergeGridCells(b gridBounds, vert, horiz []gridSegment) []*gridCell {
	rows, cols := len(b.ys)-1, len(b.xs)-1
	parent := make([]int, rows*cols)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(a, c int) { parent[find(a)] = find(c) }

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			id := r*cols + c
			if c+1 < cols && !hasVerticalBorder(vert, b.xs[c+1], b.ys[r+1], b.ys[r]) {
				union(id, id+1)
			}
			if r+1 < rows && !hasHorizontalBorder(horiz, b.ys[r+1], b.xs[c], b.xs[c+1]) {
				union(id, id+cols)
			}
		}
	}

	byRoot := map[int]*gridCell{}
	var cells []*gridCell
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			root := find(r*cols + c)
			cell, ok := byRoot[root]
			if !ok {
				cell = &gridCell{row0: r, row1: r, col0: c, col1: c}
				byRoot[root] = cell
				cells = append(cells, cell)
			}
			cell.row0, cell.row1 = min(cell.row0, r), max(cell.row1, r)
			cell.col0, cell.col1 = min(cell.col0, c), max(cell.col1, c)
		}
	}
	return cells
}

func locateGridCell(b gridBounds, cells []*gridCell, t pdf.Text) *gridCell {
	if t.X < b.xs[0]-gridSnapTolerance || t.X > b.xs[len(b.xs)-1] || t.Y > b.ys[0] || t.Y < b.ys[len(b.ys)-1]-gridSnapTolerance {
		return nil
	}
	col := sort.Search(len(b.xs), func(i int) bool { return b.xs[i] > t.X+0.5 }) - 1
	if col < 0 || col >= len(b.xs)-1 {
		return nil
	}
	row := sort.Search(len(b.ys), func(i int) bool { return b.ys[i] < t.Y }) - 1
	if row < 0 || row >= len(b.ys)-1 {
		return nil
	}
	for _, c := range cells {
		if row >= c.row0 && row <= c.row1 && col >= c.col0 && col <= c.col1 {
			return c
		}
	}
	return nil
}

func gridCellText(c *gridCell) string {
	type line struct {
		y     float64
		parts []string
	}
	var lines []*line
	for _, t := range c.parts {
		var target *line
		for _, l := range lines {
			if math.Abs(l.y-t.Y) <= 3.0 {
				target = l
				break
			}
		}
		if target == nil {
			target = &line{y: t.Y}
			lines = append(lines, target)
		}
		target.parts = append(target.parts, t.S)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].y > lines[j].y })
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(strings.Join(l.parts, ""))
		sb.WriteString(" ")
	}
	text := strings.Join(strings.Fields(sb.String()), " ")
	return strings.ReplaceAll(text, "|", "/")
}

func gridCellIsBold(c *gridCell) bool {
	if len(c.parts) == 0 {
		return false
	}
	for _, t := range c.parts {
		if strings.TrimSpace(t.S) != "" && !strings.Contains(strings.ToLower(t.Font), "bold") {
			return false
		}
	}
	return true
}

func gridHeaderRowCount(rows int, cells []*gridCell) int {
	header := 0
	for r := 0; r < rows; r++ {
		allBold, any := true, false
		for _, c := range cells {
			if c.row0 > r || c.row1 < r || len(c.parts) == 0 {
				continue
			}
			any = true
			if !gridCellIsBold(c) {
				allBold = false
				break
			}
		}
		if !any || !allBold {
			break
		}
		header = r + 1
	}
	if header == 0 {
		return 1
	}
	if header >= rows {
		return rows - 1
	}
	return header
}

func renderGridMarkdown(b gridBounds, cells []*gridCell) string {
	rows, cols := len(b.ys)-1, len(b.xs)-1
	texts := make([][]string, rows)
	for r := range texts {
		texts[r] = make([]string, cols)
	}
	for _, c := range cells {
		value := gridCellText(c)
		for r := c.row0; r <= c.row1; r++ {
			for col := c.col0; col <= c.col1; col++ {
				texts[r][col] = value
			}
		}
	}

	headerRows := gridHeaderRowCount(rows, cells)
	headers := make([]string, cols)
	for col := 0; col < cols; col++ {
		var path []string
		for r := 0; r < headerRows; r++ {
			v := texts[r][col]
			if v != "" && (len(path) == 0 || path[len(path)-1] != v) {
				path = append(path, v)
			}
		}
		headers[col] = strings.Join(path, " – ")
		if headers[col] == "" {
			headers[col] = "-"
		}
	}

	var sb strings.Builder
	sb.WriteString("| ");sb.WriteString(strings.Join(headers, " | "));sb.WriteString(" |\n")
	sb.WriteString("|");sb.WriteString(strings.Repeat(" --- |", cols));sb.WriteString("\n")
	for r := headerRows; r < rows; r++ {
		empty := true
		for _, v := range texts[r] {
			if v != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		sb.WriteString("| ");sb.WriteString(strings.Join(texts[r], " | "));sb.WriteString(" |\n")
	}
	return sb.String()
}
