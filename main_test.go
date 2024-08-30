package main

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShredMap_IncreaseCounter(t *testing.T) {

	type args struct {
		hashes     []string
		preHashMap map[string]int
	}
	tests := []struct {
		name string
		args args
		res  map[string]int
	}{
		{
			name: "single hash",
			args: args{
				hashes: []string{"123123"},
			},
			res: map[string]int{"123123": 1},
		}, {
			name: "duplicate same hash",
			args: args{
				hashes:     []string{"123123", "123123"},
				preHashMap: map[string]int{"123123": -1},
			},
			res: map[string]int{"123123": 1},
		},
		{
			name: "duplicate same hash already existing",
			args: args{
				hashes:     []string{"123123", "123123"},
				preHashMap: map[string]int{"123123": 1},
			},
			res: map[string]int{"123123": 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewShredMap()
			if tt.args.preHashMap != nil {
				s.shred = tt.args.preHashMap //this does not deep copy the map but its ok here
			}
			wg := sync.WaitGroup{}
			wg.Add(len(tt.args.hashes))
			for _, hash := range tt.args.hashes {
				go func() {
					defer wg.Done()
					s.IncreaseCounter(hash)
				}()
			}
			wg.Wait()
			assert.Equal(t, s.shred, tt.res)

		})
	}
}

func TestShredMap_DecreaseCounter(t *testing.T) {

	type args struct {
		hashes     []string
		preHashMap map[string]int
	}
	tests := []struct {
		name string
		args args
		res  map[string]int
	}{
		{
			name: "single hash",
			args: args{
				hashes: []string{"123123"},
			},
			res: map[string]int{"123123": -1},
		}, {
			name: "duplicate same hash",
			args: args{
				hashes:     []string{"123123", "123123"},
				preHashMap: map[string]int{"123123": 1},
			},
			res: map[string]int{"123123": -1},
		},
		{
			name: "duplicate same hash already existing",
			args: args{
				hashes:     []string{"123123", "123123"},
				preHashMap: map[string]int{"123123": -1},
			},
			res: map[string]int{"123123": -3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewShredMap()
			if tt.args.preHashMap != nil {
				s.shred = tt.args.preHashMap //this does not deep copy the map but its ok here
			}
			wg := sync.WaitGroup{}
			wg.Add(len(tt.args.hashes))
			for _, hash := range tt.args.hashes {
				go func() {
					defer wg.Done()
					s.DecreaseCounter(hash)
				}()
			}
			wg.Wait()
			assert.Equal(t, s.shred, tt.res)

		})
	}
}

func TestShredMap_IsEmptyAndSize(t *testing.T) {

	type args struct {
		preHashMap map[string]int
	}
	tests := []struct {
		name    string
		args    args
		isEmpty bool
		size    int
	}{
		{
			name:    "empty",
			isEmpty: true,
			size:    0,
		}, {
			name: "not empty",
			args: args{
				preHashMap: map[string]int{"123123": 1},
			},
			isEmpty: false,
			size:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewShredMap()
			if tt.args.preHashMap != nil {
				s.shred = tt.args.preHashMap //this does not deep copy the map but its ok here
			}
			assert.Equal(t, s.IsEmpty(), tt.isEmpty)
			assert.Equal(t, s.Size(), tt.size)

		})
	}
}

func TestErrCheck(t *testing.T) {
	type args struct {
		e error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "should panic",
			args: args{
				e: fmt.Errorf("should panic"),
			},
		},
		{
			name: "shouldnt panic",
			args: args{
				e: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.e != nil {
				assert.Panics(t, func() { ErrCheck(tt.args.e) })
			} else {
				assert.NotPanics(t, func() { ErrCheck(tt.args.e) })
			}
		})
	}
}

func TestHashAnything(t *testing.T) {
	type args struct {
		anything []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success",
			args: args{anything: []byte("something")},
			want: "3fc9b689459d738f8c88a3a48aa9e33542016b7a4052e001aaa536fca74813cb",
		},
		{
			name: "empty",
			args: args{anything: []byte{}},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HashAnything(tt.args.anything); got != tt.want {
				t.Errorf("HashAnything() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamedJParse(t *testing.T) {
	type args struct {
		input *bytes.Buffer
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "no error",
			args: args{
				input: bytes.NewBufferString("[]"),
			},
		},
		{
			name: "success",
			args: args{
				input: bytes.NewBufferString(`[
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25}
				]`),
			},
		},
		{
			name: "error- no opening bracket",
			args: args{
				input: bytes.NewBufferString("}"),
			},
			wantErr: fmt.Errorf("failed to read open bracket: invalid character '}' looking for beginning of value"),
		},
		{
			name: "error- no closing bracket",
			args: args{
				input: bytes.NewBufferString("["),
			},
			wantErr: fmt.Errorf("failed to read closing bracket: EOF"),
		},
		{
			name: "error- invalid",
			args: args{
				input: bytes.NewBufferString("[adssad]"),
			},
			wantErr: fmt.Errorf("failed to decode: invalid character 'a' looking for beginning of value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewShredMap()

			err := StreamedJParse(s.IncreaseCounter, tt.args.input)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}

		})
	}
}
