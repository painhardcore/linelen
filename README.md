# LineLen - Line Length Analyzer

<p align="center">
 <img src="https://github.com/painhardcore/linelen/assets/4718161/98e38e4b-732c-467e-b98f-95ddb5010784" alt="size-is-everything-neil-degrasse-tyson" width="400"/>
</p>

LineLen is a tool that analyzes text input to provide insights into line lengths. Understanding line sizes in logs is crucial: it helps in identifying anomalies, optimizing batch processes, fine-tuning buffers, and ensuring efficient data processing. Know your data; size does matter.

## Features

- Groups line lengths into 1,000-byte buckets.
- Prints statistics every 10 seconds and once when input ends.
- Calculate the 50th, 90th, 95th, and 99th percentile of line lengths.
- Option to export results to a CSV file.
- Uses T-Digest to efficiently compute percentiles without storing all data points in memory.

## Usage

```
cat yourfile.txt | linelen
```
**Options:**

`-f` : Write bucket counts to a CSV file in addition to statistics on stdout.

Example:
```
cat yourfile.txt | linelen -f output.csv
```

Lengths are measured in bytes, excluding LF or CRLF line endings. UTF-8
characters can occupy more than one byte. Bucket ranges include the lower
bound and exclude the upper bound, so a 1,000-byte line belongs to `1000-2000 bytes`.
Percentiles are approximate byte lengths.

Reports use plain text, including when stdout is redirected. Empty input reports
zero lines without percentiles. Read and output errors are printed to stderr
and cause a nonzero exit status. A read error prevents the final report and CSV
export; periodic reports already printed describe only the input read so far.

## Why Input from Stdin?

The tool reads from stdin because many large files, especially logs, are often compressed. By using stdin, you can decompress and pipe your data into the tool in one step, streamlining the analysis process. This approach offers flexibility as you can pipe any data source, transformed or filtered, into the tool.

## Installation

### Manually

To install the LineLen tool, you need to have Go installed on your machine.

1. Clone the repository:
```bash
git clone https://github.com/painhardcore/linelen.git
```

2. Navigate to the directory and build:
```bash
cd linelen
go build -o linelen .
```

3. Run `./linelen`, or put the binary in a directory on your `PATH` to use `linelen`.

### Using `go install`

You can directly install the tool to your `$GOPATH/bin` directory (make sure this directory is in your `PATH`) using:

```bash
go install github.com/painhardcore/linelen@latest
```

## Limits

- Input lines must fit in the 100 MiB scanner buffer, including space needed to
  detect line endings. Longer lines stop analysis with an error. The buffer
  starts at 64 KiB and grows as needed.
- The refresh interval and bucket size are fixed.
- CSV export replaces an existing file at the requested path.

## Development

Run the tests, including concurrent reporting checks:

```bash
go test -race ./...
```

Benchmark recording line lengths without input parsing:

```bash
go test -run '^$' -bench BenchmarkStatisticsAdd -benchmem
```

## Contributing

Contributions are welcome! If you find any issues or have feature requests, please open an issue in the repository. Pull requests are also appreciated.
