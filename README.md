# LineLen - Line Length Analyzer

<p align="center">
 <img src="https://github.com/painhardcore/linelen/assets/4718161/98e38e4b-732c-467e-b98f-95ddb5010784" alt="size-is-everything-neil-degrasse-tyson" width="400"/>
</p>

LineLen is a tool that analyzes text input to provide insights into line lengths. Understanding line sizes in logs is crucial: it helps in identifying anomalies, optimizing batch processes, fine-tuning buffers, and ensuring efficient data processing. Know your data; size does matter.

## Features

- Dynamic bucketing based on line lengths.
- Periodic display of statistics.
- Calculate the 50th, 90th, 95th, and 99th percentile of line lengths.
- Option to export results to a CSV file.
- Uses T-Digest to efficiently compute percentiles without storing all data points in memory.

## Usage

```
cat yourfile.txt | linelen
```
**Options:**

`-f` : Filename to write the output in CSV format. If not provided, will print to stdout.

Example:
```
cat yourfile.txt | linelen -f output.csv
```

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
go build -o linelen main.go
```

3. Now, you can use the `linelen` command as shown in the Usage section.

### Using `go get`

You can directly install the tool to your `$GOPATH/bin` directory (make sure this directory is in your `PATH`) using:

```bash
go get -u github.com/painhardcore/linelen
```

## Known Issues

- **Screen Clearing**: The screen-clearing mechanism might not work properly on all terminals, especially when using some SSH sessions or terminal multiplexers like `screen`. Unfortunately, there's nothing we can do about this behavior in such environments.
  
- **Hardcoded Settings**: A lot of settings are currently hardcoded. Feel free to adjust them in the code according to your requirements, or raise an issue to have them converted to configurable flags.

## Contributing

Contributions are welcome! If you find any issues or have feature requests, please open an issue in the repository. Pull requests are also appreciated.
