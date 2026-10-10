package main

import "testing"

func TestGridCellsSitOnTheContentGutters(t *testing.T) {
	first, last := gridCell(0), gridCell(gridColumns-1)
	if first.Min.X != gutter || last.Max.X != screenWidth-gutter {
		t.Fatalf("grid spans %d..%d, want the gutters %d..%d", first.Min.X, last.Max.X, gutter, screenWidth-gutter)
	}
	for index := 0; index < 12; index++ {
		cell := gridCell(index)
		if cell.Dx() != contentWidth/gridColumns || cell.Dy() != gridRowHeight {
			t.Fatalf("cell %d is %dx%d, want identical cells", index, cell.Dx(), cell.Dy())
		}
	}
	if gridRows() != 4 {
		t.Fatalf("rows = %d, want 4 in the 1139 px page area of the Libra 2", gridRows())
	}
}

func TestGridNameIsAtMostTwoLinesAndElided(t *testing.T) {
	lines := gridNameText("Đồng bộ lịch cho danh sách đọc sách rất dài của người dùng")
	if len(lines) != gridNameLines {
		t.Fatalf("name wraps to %d lines, want %d", len(lines), gridNameLines)
	}
}
