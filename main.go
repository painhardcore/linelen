package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/influxdata/tdigest"
)

// this is evil don't do this at home
var globalMutex sync.RWMutex

const (
	// refreshInterval is the interval at which the statistics are printed
	refreshInterval = 10 * time.Second
	// bucketSize is the size of each bucket, i.e. 0-1000, 1000-2000, etc.
	bucketSize = 1000
)

// Category represents a category of line lengths
type Category struct {
	name  string
	start int
	end   int
	count int
}

// inRange checks if the given value is in the range of the category
func (cat *Category) inRange(val int) bool {
	return val >= cat.start && val < cat.end
}

func main() {
	var outputFile string
	flag.StringVar(&outputFile, "f", "", "Filename to write the output in CSV format. If not provided, will print to stdout.")
	flag.Parse()

	// 1000 is the compression factor, the higher the more accurate the results
	td := tdigest.NewWithCompression(10000)

	// initialize the first category
	categories := []*Category{
		{"0-1000 chars", 0, 1000, 0},
	}
	scanner := bufio.NewScanner(os.Stdin)

	// 100MB buffer should be enough for everyone
	buf := make([]byte, 0, 100*1024*1024)
	scanner.Buffer(buf, cap(buf))

	totalLines := 0

	// print statistics every interval
	ticker := time.NewTicker(refreshInterval)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				printStatistics(categories, totalLines, td)
			case <-done:
				return
			}
		}
	}()

	// main loop
	for scanner.Scan() {
		globalMutex.Lock()
		length := len(scanner.Text())
		td.Add(float64(length), 1)
		totalLines++

		lastCategory := categories[len(categories)-1]
		for length >= lastCategory.end {
			newEnd := lastCategory.end + bucketSize
			categories = append(categories, &Category{fmt.Sprintf("%d-%d chars", lastCategory.end, newEnd), lastCategory.end, newEnd, 0})
			lastCategory = categories[len(categories)-1]
		}

		for _, cat := range categories {
			if cat.inRange(length) {
				cat.count++
				break
			}
		}
		globalMutex.Unlock()
	}

	done <- true

	// print statistics one last time
	printStatistics(categories, totalLines, td)

	if outputFile != "" {
		err := writeToCSV(outputFile, categories)
		if err != nil {
			fmt.Printf("Error writing to output file: %v\n", err)
			return
		}
	}
}

// clearScreen clears the screen
func clearScreen() {
	// ANSI is not working with ssh
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// printStatistics prints the current statistics
func printStatistics(categories []*Category, totalLines int, td *tdigest.TDigest) {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	clearScreen()
	fmt.Println("Current statistics:")
	for _, cat := range categories {
		if cat.count > 0 {
			fmt.Printf("%s - %d\n", cat.name, cat.count)
		}
	}
	fmt.Printf("\nTotal lines: %d\n", totalLines)
	fmt.Println("------------------")
	fmt.Print("\nGlobal Summary:\n")
	fmt.Printf("50th percentile: %f\n", td.Quantile(0.5))
	fmt.Printf("90th percentile: %f\n", td.Quantile(0.9))
	fmt.Printf("95th percentile: %f\n", td.Quantile(0.95))
	fmt.Printf("99th percentile: %f\n", td.Quantile(0.99))
}

func writeToCSV(filename string, categories []*Category) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	writer.WriteString("Range,Count\n")
	for _, cat := range categories {
		if cat.count > 0 {
			writer.WriteString(fmt.Sprintf("%s,%d\n", cat.name, cat.count))
		}
	}
	return writer.Flush()
}
