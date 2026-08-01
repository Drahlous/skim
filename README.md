# skim

Tool for skimming plaintext log files, like TextAnalysisTool or glogg.

## Screenshots

![Skim example usage](./screenshots/skim.png)

## Usage

```text
Usage of skim:
  -filter string
        supply the path to a TAT filter file (default "./examples/simple_filter_two.tat")
  -log string
        supply the path to the input log file (default "./examples/simple_longer.log")
```

## Development

Build:

```sh
go build -v ./...
```

Run all unit tests:

```sh
go test ./...
```

For verbose output and the race detector (matches what CI runs):

```sh
go test -v -race ./...
```
