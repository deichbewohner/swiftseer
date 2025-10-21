package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// captureOutput runs fn while capturing its stdout output and returns it as a string.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan struct{})
	var out []byte
	go func() {
		out, _ = io.ReadAll(r)
		close(done)
	}()
	fn()
	_ = w.Close()
	<-done
	os.Stdout = old
	return string(out)
}

func TestInspectCmd_JSON(t *testing.T) {
	csv := "date,product,units\n2020-11-01,A,10\n2020-12-01,A,20\n"
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.csv")
	if err := os.WriteFile(fp, []byte(csv), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := captureOutput(t, func() {
		if err := InspectCmd([]string{"--json", fp}); err != nil {
			t.Fatalf("InspectCmd: %v", err)
		}
	})
	var r inspectReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; out=%s", err, out)
	}
	if r.DateColumn != "date" || r.DateFormat != "%Y-%m-%d" {
		t.Errorf("date detection mismatch: %+v", r)
	}
	if len(r.ValueCols) != 1 || r.ValueCols[0] != "units" {
		t.Errorf("value columns mismatch: %+v", r)
	}
	if len(r.GroupCols) != 1 || r.GroupCols[0] != "product" {
		t.Errorf("group columns mismatch: %+v", r)
	}
}

func TestInspectCmd_JSON_Golden(t *testing.T) {
	csv := "date,product,units\n2020-11-01,A,10\n2020-12-01,A,20\n"
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.csv")
	if err := os.WriteFile(fp, []byte(csv), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := captureOutput(t, func() {
		if err := InspectCmd([]string{"--json", fp}); err != nil {
			t.Fatalf("InspectCmd: %v", err)
		}
	})
	goldenPath := filepath.Join("testdata", "inspect_simple.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if normalizeNewlines(out) != normalizeNewlines(string(want)) {
		t.Errorf("golden mismatch\nGot:\n%s\nWant:\n%s", out, string(want))
	}
}

func TestInspectCmd_JSON_Semicolon_Golden(t *testing.T) {
	csv := "date;product;units\n2020-11-01;A;10\n2020-12-01;A;20\n"
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.csv")
	if err := os.WriteFile(fp, []byte(csv), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := captureOutput(t, func() {
		if err := InspectCmd([]string{"--json", fp}); err != nil {
			t.Fatalf("InspectCmd: %v", err)
		}
	})
	goldenPath := filepath.Join("testdata", "inspect_semicolon.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if normalizeNewlines(out) != normalizeNewlines(string(want)) {
		t.Errorf("golden mismatch\nGot:\n%s\nWant:\n%s", out, string(want))
	}
}

func TestInspectCmd_JSON_Tab_Golden(t *testing.T) {
	csv := "date\tproduct\tunits\n2020-11-01\tA\t10\n2020-12-01\tA\t20\n"
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.csv")
	if err := os.WriteFile(fp, []byte(csv), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := captureOutput(t, func() {
		if err := InspectCmd([]string{"--json", fp}); err != nil {
			t.Fatalf("InspectCmd: %v", err)
		}
	})
	goldenPath := filepath.Join("testdata", "inspect_tab.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if normalizeNewlines(out) != normalizeNewlines(string(want)) {
		t.Errorf("golden mismatch\nGot:\n%s\nWant:\n%s", out, string(want))
	}
}

func TestInspectCmd_JSON_EUDate_Golden(t *testing.T) {
	csv := "date,product,units\n01.11.2020,A,10\n01.12.2020,A,20\n"
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.csv")
	if err := os.WriteFile(fp, []byte(csv), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := captureOutput(t, func() {
		if err := InspectCmd([]string{"--json", fp}); err != nil {
			t.Fatalf("InspectCmd: %v", err)
		}
	})
	goldenPath := filepath.Join("testdata", "inspect_eu_date.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if normalizeNewlines(out) != normalizeNewlines(string(want)) {
		t.Errorf("golden mismatch\nGot:\n%s\nWant:\n%s", out, string(want))
	}
}
