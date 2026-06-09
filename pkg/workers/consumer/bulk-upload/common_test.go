package bulkupload

import "testing"

func TestCountNonEmptyRows(t *testing.T) {
	rows := [][]string{
		{"INV-001", "value"},
		{"", ""},
		{"   ", "\t"},
		{"INV-002", ""},
	}

	if got := countNonEmptyRows(rows); got != 2 {
		t.Fatalf("expected 2 non-empty rows, got %d", got)
	}
}

func TestCountCSVRowsExcludesBlankRows(t *testing.T) {
	data := []byte("invoice_number,issue_date\nINV-001,2026-01-16\n,\n   ,   \nINV-002,2026-01-17\n")

	processor := &CSVProcessor{}
	totalRows, err := processor.countCSVRows(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if totalRows != 2 {
		t.Fatalf("expected 2 counted rows, got %d", totalRows)
	}
}
