# Lazycomment - a simple cli tool to set default comments on exported functions in Go files

This library helps in removing squiggly lines and adding default comments to exported functions and variables.

## Usage

In the command line, run the following snippet to Parse all files in the current directory or a single file.

```bash
go run lazycomment.go -c=defaultcomment -dir="." //for the current directory

go run lazycomment.go -c=defaultcomment -dir="./sample.go" //for a single file
```
