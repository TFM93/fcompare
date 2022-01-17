package fcompare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

var shred map[string]int = map[string]int{}

func main() {
	//async control variables
	mux := &sync.Mutex{}
	wg := sync.WaitGroup{}

	// read positional args
	positionalArgs := os.Args[1:]
	if len(positionalArgs) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ./main /path/f1.json /path/f2.json")
		os.Exit(1)
	}
	//file 1 location
	fin := positionalArgs[0]
	//file 2 location
	fout := positionalArgs[1]

	// open files without loading them to memory
	f1, err := os.Open(fin)
	ErrCheck(err)
	f2, err := os.Open(fout)
	ErrCheck(err)

	// create 2 goroutines to generate hashes and compare them
	wg.Add(2)
	go StreamedJParse(true, f1, mux, &wg)
	go StreamedJParse(false, f2, mux, &wg)
	wg.Wait()

	// at the end of the parsing, check the shred map
	if len(shred) > 0 {
		fmt.Fprintf(os.Stderr, "Files not identical. %v differences found.\n", len(shred))
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Files identical :) ")
	os.Exit(0)
}

func StreamedJParse(master bool, input *os.File, mux *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	//var count int64 = 0
	dec := json.NewDecoder(input)

	// updater variable points to the proper function
	var updater func(string, *sync.Mutex)
	if master {
		updater = UpdateShredMaster
	} else {
		updater = UpdateShredSlave
	}

	// read open bracket (type json.Delim)
	_, err := dec.Token()
	if err != nil {
		log.Fatal(err)
	}

	// loop through values
	for dec.More() {
		var m interface{}
		// decode object
		err := dec.Decode(&m)
		if err != nil {
			log.Fatal(err)
		}
		// decoding to interface{} and then mashalling does the trick
		// of sorting json keys
		entry, _ := json.Marshal(m)
		//fmt.Println(entry)
		//fmt.Println(m)
		x := HashAnything(entry)
		updater(x, mux)
		//count++
	}

	// read closing bracket (type json.Delim)
	_, err = dec.Token()
	if err != nil {
		log.Fatal(err)
	}
}

func UpdateShredMaster(hash string, mux *sync.Mutex) {
	// here we should use mutex to avoid race conditions
	mux.Lock()
	val, ok := shred[hash]
	if ok {
		if val == -1 {
			// means that is on the slave file with one occurence
			delete(shred, hash)
		} else {
			//increment counter
			shred[hash] += 1
		}
	} else {
		// add to map
		shred[hash] = 1
	}
	mux.Unlock()
}

func UpdateShredSlave(hash string, mux *sync.Mutex) {
	// here we should use mutex to avoid race conditions
	mux.Lock()
	val, ok := shred[hash]
	if ok {
		if val == 1 {
			// means that is on the master file with one occurence
			delete(shred, hash)
		} else {
			//decrement counter
			shred[hash] -= 1
		}
	} else {
		// add to map
		shred[hash] = -1
	}
	mux.Unlock()
}

// ErrCheck panics on error
func ErrCheck(e error) {
	if e != nil {
		panic(e)
	}
}

// HashAnything generates a sha256 of a []byte
func HashAnything(anything []byte) string {
	sum := sha256.Sum256(anything)
	return hex.EncodeToString(sum[0:])
}
