# __FCompare__

**fcompare** is a demo tool that accepts two JSON files and prints out if they are equal.

## __Table of contents__

- [**How To Run**](#how-to-run)
- [**How It Works**](#how-it-works)

## __How To Run__

**fcompare** needs 2 file paths to run. To test this, one could:

```
go build fcompare.go
./fcompare filepath1.json filepath2.json

or just:

go run fcompare filepath1.json filepath2.json
```

## __How It Works__

The program will open both files but not load them immediately to memory. Instead,
2 go routines are created, one per file. Each routine will stream the file contents in chunks (json objects in this case), generate a checksum for the streamed object and map the checksum into the memory. In the end, if the synced memory is empty the files are identical.
