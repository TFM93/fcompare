package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

func main() {
	shred := NewShredMap()
	wg := sync.WaitGroup{}

	// read positional args
	positionalArgs := os.Args[1:]
	if len(positionalArgs) != 2 {
		log.Fatal("usage: ./main /path/f1.json /path/f2.json")
	}

	fin := positionalArgs[0]
	fout := positionalArgs[1]

	// open files without loading them to memory
	f1, err := os.Open(fin)
	ErrCheck(err)
	defer f1.Close()

	f2, err := os.Open(fout)
	ErrCheck(err)
	defer f2.Close()

	// create 2 goroutines to generate hashes and compare them
	wg.Add(2)
	errChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		errChan <- StreamedJParse(shred.IncreaseCounter, f1)
	}()
	go func() {
		defer wg.Done()
		errChan <- StreamedJParse(shred.DecreaseCounter, f2)
	}()

	// closes errChan when wg tasks are done
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// keeps reading until errChan is closed
	for err := range errChan {
		if err != nil {
			log.Fatalf("Error: %s\n", err)
		}
	}

	// at the end of the parsing, check the shred map contents
	if !shred.IsEmpty() {
		log.Fatalf("Files not identical. %v differences found.\n", shred.Size())
	}
}

func StreamedJParse(updater func(string), input io.Reader) error {
	dec := json.NewDecoder(input)

	// read open bracket (type json.Delim)
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read open bracket: %w", err)
	}

	switch t := token.(type) {
	case json.Delim:
		if t.String() != "[" {
			return fmt.Errorf("expected opening bracket '[' but got %s", t.String())
		}
	default:
		return fmt.Errorf("expected JSON array but got %T", token)
	}

	for dec.More() {
		// decoding to interface{} and then marshalling does the trick of sorting json keys
		// an alternative would be switching m contents to figure out the type ([]interface{}/map[string]interface{}/...)
		// and sort the contents recursively
		var m interface{}
		err := dec.Decode(&m)
		if err != nil {
			return fmt.Errorf("failed to decode: %w", err)
		}
		entry, _ := json.Marshal(m)
		updater(HashAnything(entry))
	}

	// read closing bracket (type json.Delim)
	_, err = dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read closing bracket: %w", err)
	}
	return nil
}

// ShredMap wraps a map and its associated mutex
type ShredMap struct {
	mux   sync.Mutex
	shred map[string]int
}

// NewShredMap initializes and returns a new ShredMap
func NewShredMap() *ShredMap {
	return &ShredMap{
		shred: make(map[string]int),
	}
}

// IncreaseCounter increases hash counter and if val is -1, deletes the entry to guarantee that 0 values are removed.
func (s *ShredMap) IncreaseCounter(hash string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	val, ok := s.shred[hash]
	if ok {
		if val == -1 {
			// on the slave file with one occurrence
			delete(s.shred, hash)
		} else {
			s.shred[hash] += 1
		}
	} else {
		s.shred[hash] = 1
	}
}

// DecreaseCounter decreases hash counter and if val is 1, deletes the entry to guarantee that 0 values are removed.
func (s *ShredMap) DecreaseCounter(hash string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	val, ok := s.shred[hash]
	if ok {
		if val == 1 {
			// on the master file with one occurrence
			delete(s.shred, hash)
		} else {
			s.shred[hash] -= 1
		}
	} else {
		s.shred[hash] = -1
	}
}

// IsEmpty checks if the length of shred is 0
func (s *ShredMap) IsEmpty() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return len(s.shred) == 0
}

// Size returns the length of shred
func (s *ShredMap) Size() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return len(s.shred)
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
