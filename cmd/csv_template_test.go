package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCsvTemplate_Default_Golden(t *testing.T) {
	out := captureOutput(t, func() {
		if err := CsvTemplateCmd([]string{}); err != nil {
			t.Fatalf("CsvTemplateCmd: %v", err)
		}
	})
	goldenPath := filepath.Join("testdata", "template_default.csv")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if strings.TrimSpace(out) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch\nGot:\n%s\nWant:\n%s", out, string(want))
	}
}

func TestCsvTemplate_WithGroups_Golden(t *testing.T) {
	out := captureOutput(t, func() {
		if err := CsvTemplateCmd([]string{"--with-groups"}); err != nil {
			t.Fatalf("CsvTemplateCmd: %v", err)
		}
	})
	goldenPath := filepath.Join("testdata", "template_with_groups.csv")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if strings.TrimSpace(out) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch\nGot:\n%s\nWant:\n%s", out, string(want))
	}
}
