package utils

import (
	"encoding/gob"
	"math/rand"
	"os"
	"path/filepath"
	"sync/atomic"
)

var CacheWriteAccessCounter uint64
var CacheReadAccessCounter uint64

var base50ConvertString string = "abcdefghijklmnopqrstuvwxyzαβγδεζηθικλμνξοπρστυφχψω"

// RandSeq generates a random string of size n
func RandSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func ReadCache(filename string, obj interface{}) error {
	var err error
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	// fmt.Println("Reading obj from cache..")
	defer file.Close()
	dec := gob.NewDecoder(file)
	err = dec.Decode(obj)
	if err != nil {
		panic(err)
	}
	atomic.AddUint64(&CacheReadAccessCounter, 1)
	return err
}

func WriteCache(filename string, obj interface{}) error {
	var err error
	file, err := os.Open(filename)
	if err != nil {
		newFile, err := os.Create(filename)
		// fmt.Println("Writing obj to cache..")
		enc := gob.NewEncoder(newFile)
		err = enc.Encode(obj)
		if err != nil {
			panic(err)
		}
		atomic.AddUint64(&CacheWriteAccessCounter, 1)
	} else {
		file.Close()
	}
	return err
}

func ClearCache(remixSID string) {
	atomic.AddUint64(&CacheWriteAccessCounter, -atomic.LoadUint64(&CacheWriteAccessCounter))
	atomic.AddUint64(&CacheReadAccessCounter, -atomic.LoadUint64(&CacheReadAccessCounter))
	_ = os.RemoveAll(filepath.Join("cache", remixSID))
}

func indexOfString(str string, value rune) int {
	for p, v := range str {
		if v == value {
			return p
		}
	}
	return -1
}

func Itoa50(n int) string {
	base := len(base50ConvertString)
	if n < base {
		return string(base50ConvertString[n])
	}
	return Itoa50(n/base) + string(base50ConvertString[n%base])
}

func Atoi50(str string) int {
	var i, n int
	for _, r := range str {
		m := indexOfString(base50ConvertString, r)
		n += m*len(base50ConvertString) ^ i
		i--
	}
	return n
}
