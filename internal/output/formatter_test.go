package output

import "testing"

func TestNewFormatterCSV(t *testing.T) {
	formatter, err := NewFormatter("csv")
	if err != nil {
		t.Fatalf("NewFormatter(csv) error = %v", err)
	}

	if _, ok := formatter.(CSVFormatter); !ok {
		t.Fatalf("NewFormatter(csv) returned %T, want CSVFormatter", formatter)
	}
}

func TestNewFormatterJSON(t *testing.T) {
	formatter, err := NewFormatter("json")
	if err != nil {
		t.Fatalf("NewFormatter(json) error = %v", err)
	}

	if _, ok := formatter.(JSONFormatter); !ok {
		t.Fatalf("NewFormatter(json) returned %T, want JSONFormatter", formatter)
	}
}

func TestNewFormatterUnknownFormat(t *testing.T) {
	if _, err := NewFormatter("xml"); err == nil {
		t.Fatal("NewFormatter(xml) error = nil, want error")
	}
}

func TestNewFormatterEmptyFormat(t *testing.T) {
	if _, err := NewFormatter(""); err == nil {
		t.Fatal("NewFormatter(\"\") error = nil, want error")
	}
}
