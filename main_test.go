package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
	}{
		{"empty", "", "Total lines: 0\nNo lines read.\n"},
		{"blank", "\n", "50th percentile: 0.000000"},
		{"unicode", "é\r\né", "50th percentile: 2.000000"},
		{"unterminated", "abc", "50th percentile: 3.000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := run(strings.NewReader(tc.input), &output, "", nil); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tc.want) {
				t.Fatalf("output = %q, want %q", output.String(), tc.want)
			}
			if strings.ContainsAny(output.String(), "\x1b") || strings.Contains(output.String(), "NaN") {
				t.Fatalf("unexpected terminal escapes or NaN: %q", output.String())
			}
		})
	}
}

type failingIO struct{ err error }

func (f failingIO) Read([]byte) (int, error)  { return 0, f.err }
func (f failingIO) Write([]byte) (int, error) { return 0, f.err }

func TestRunErrors(t *testing.T) {
	failure := errors.New("I/O failed")
	t.Run("input", func(t *testing.T) {
		var output bytes.Buffer
		filename := filepath.Join(t.TempDir(), "output.csv")
		err := run(io.MultiReader(strings.NewReader("one\n"), failingIO{failure}), &output, filename, nil)
		if !errors.Is(err, failure) || !strings.Contains(err.Error(), "read input") {
			t.Fatalf("error = %v", err)
		}
		if output.Len() != 0 {
			t.Fatalf("printed a final report after failed input: %q", output.String())
		}
		if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("CSV exists after failed input: %v", err)
		}
	})
	t.Run("output", func(t *testing.T) {
		err := run(strings.NewReader("one\n"), failingIO{failure}, "", nil)
		if !errors.Is(err, failure) || !strings.Contains(err.Error(), "write statistics") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("CSV", func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), "missing", "output.csv")
		err := run(strings.NewReader("one\n"), io.Discard, filename, nil)
		if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "write CSV") {
			t.Fatalf("error = %v", err)
		}
	})
}

type longLineReader struct{}

func (longLineReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestLineLimit(t *testing.T) {
	err := run(io.LimitReader(longLineReader{}, maxLineSize+1), io.Discard, "", nil)
	if err == nil || !strings.Contains(err.Error(), "read input: bufio.Scanner: token too long") {
		t.Fatalf("error = %v", err)
	}
}

func TestPeriodicReportError(t *testing.T) {
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	failure := errors.New("report failed")
	ticks := make(chan time.Time)
	result := make(chan error, 1)
	go func() { result <- run(input, failingIO{failure}, "", ticks) }()
	ticks <- time.Time{}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, failure) {
		t.Fatalf("error = %v", err)
	}
}

func TestCSV(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "output.csv")
	if err := run(strings.NewReader("a\nb\n"), io.Discard, filename, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Range,Count\n0-1000 chars,2\n"; string(data) != want {
		t.Fatalf("CSV = %q, want %q", data, want)
	}
}

func TestConcurrentStatistics(t *testing.T) {
	s := newStatistics()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			s.add(i * 10)
		}
	}()
	for i := 0; i < 100; i++ {
		if err := s.print(io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	var output bytes.Buffer
	if err := s.print(&output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Total lines: 1000\n") {
		t.Fatalf("output = %q", output.String())
	}
}

type reportWriter chan string

func (w reportWriter) Write(p []byte) (int, error) {
	w <- string(p)
	return len(p), nil
}

func TestPeriodicReport(t *testing.T) {
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	ticks := make(chan time.Time)
	output := make(reportWriter, 2)
	result := make(chan error, 1)
	go func() { result <- run(input, output, "", ticks) }()
	if _, err := io.WriteString(writer, "one\n"); err != nil {
		t.Fatal(err)
	}
	ticks <- time.Time{}
	if report := <-output; !strings.Contains(report, "Current statistics:") {
		t.Fatalf("periodic report = %q", report)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if report := <-output; !strings.Contains(report, "Total lines: 1\n") {
		t.Fatalf("final report = %q", report)
	}
}
