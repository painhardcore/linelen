package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/influxdata/tdigest"
)

const (
	refreshInterval = 10 * time.Second
	bucketSize      = 1000
	maxLineSize     = 100 * 1024 * 1024
)

type statistics struct {
	mu         sync.Mutex
	buckets    []int
	totalLines int
	td         *tdigest.TDigest
}

func newStatistics() *statistics {
	return &statistics{
		td: tdigest.NewWithCompression(10000),
	}
}

func (s *statistics) add(length int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.td.Add(float64(length), 1)
	s.totalLines++
	bucket := length / bucketSize
	for len(s.buckets) <= bucket {
		s.buckets = append(s.buckets, 0)
	}
	s.buckets[bucket]++
}

func main() {
	var outputFile string
	flag.StringVar(&outputFile, "f", "", "Write bucket counts to a CSV file in addition to stdout statistics.")
	flag.Parse()
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	if err := run(os.Stdin, os.Stdout, outputFile, ticker.C); err != nil {
		fmt.Fprintf(os.Stderr, "linelen: %v\n", err)
		os.Exit(1)
	}
}

func run(input io.Reader, output io.Writer, outputFile string, ticks <-chan time.Time) error {
	s := newStatistics()
	done := make(chan struct{})
	stopped := make(chan struct{})
	var reportErr error
	go func() {
		defer close(stopped)
		for {
			select {
			case <-ticks:
				if reportErr = s.print(output); reportErr != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)
	for scanner.Scan() {
		s.add(len(scanner.Bytes()))
	}
	close(done)
	<-stopped
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	if reportErr != nil {
		return fmt.Errorf("write statistics: %w", reportErr)
	}
	if err := s.print(output); err != nil {
		return fmt.Errorf("write statistics: %w", err)
	}
	if outputFile != "" {
		if err := s.writeToCSV(outputFile); err != nil {
			return fmt.Errorf("write CSV %q: %w", outputFile, err)
		}
	}
	return nil
}

func (s *statistics) print(output io.Writer) error {
	// Quantile can update the digest, so reports also need the exclusive lock.
	s.mu.Lock()
	defer s.mu.Unlock()
	w := bufio.NewWriter(output)
	fmt.Fprintln(w, "Current statistics:")
	for i, count := range s.buckets {
		if count > 0 {
			fmt.Fprintf(w, "%d-%d bytes - %d\n", i*bucketSize, (i+1)*bucketSize, count)
		}
	}
	fmt.Fprintf(w, "\nTotal lines: %d\n", s.totalLines)
	if s.totalLines == 0 {
		fmt.Fprintln(w, "No lines read.")
		return w.Flush()
	}
	fmt.Fprintln(w, "------------------\n\nGlobal Summary:")
	fmt.Fprintf(w, "50th percentile: %f\n", s.td.Quantile(0.5))
	fmt.Fprintf(w, "90th percentile: %f\n", s.td.Quantile(0.9))
	fmt.Fprintf(w, "95th percentile: %f\n", s.td.Quantile(0.95))
	fmt.Fprintf(w, "99th percentile: %f\n", s.td.Quantile(0.99))
	return w.Flush()
}

func (s *statistics) writeToCSV(filename string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	w := bufio.NewWriter(file)
	fmt.Fprintln(w, "Range,Count")
	for i, count := range s.buckets {
		if count > 0 {
			fmt.Fprintf(w, "%d-%d bytes,%d\n", i*bucketSize, (i+1)*bucketSize, count)
		}
	}
	return w.Flush()
}
