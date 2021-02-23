package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

const HashcacheFilename = ".hashCache.json"

type HashCache struct {
	Path string `json:"path"`
	Hash []byte `json:"hash"`
}

var hashCache []HashCache

var newHashCache []HashCache
var hashCacheLock sync.RWMutex

func main() {
	dirs := []string{
		"path/to/function",
	}

	// check if file exists
	if fileExists(HashcacheFilename) {
		b, err := readFile(HashcacheFilename)
		if err != nil {
			log.Println(err)
		}

		err = json.Unmarshal(b, &hashCache)
		if err != nil {
			log.Println(err)
		}

	}

	var wg sync.WaitGroup

	for _, path := range dirs {
		wg.Add(1)
		go build(path, &wg)
	}

	wg.Wait()

	f, err := json.MarshalIndent(newHashCache, "", "	")
	if err != nil {
		log.Println(err)
	}


	pwd, _ := os.Getwd()
	err = os.Remove(fmt.Sprintf("%v/%v", pwd, HashcacheFilename))
	if err != nil {
		log.Println(err)
	}

	err = os.WriteFile(HashcacheFilename, f, 0644)
	if err != nil {
		log.Println(err)
	}

}

func build(path string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Compiling go binary

	cmd := exec.Command("go",
		"build",
		"-o",
		fmt.Sprintf("%v/main", path),
		fmt.Sprintf("%v/main.go", path),
	)

	err := cmd.Run()

	if err != nil {
		log.Printf("Error compiling - %v: %v", path, err)
		return
	}

	// Cache stuff

	cachedHash, err := getHashFromCache(path)
	if err != nil {
		log.Println(err)
	}

	f, err := readFile(path + "/main")
	hash := hashBytes(f)
	if err != nil {
		log.Println(err)
	}

	if string(cachedHash) == string(hash) {
		hashCacheLock.Lock()
		newHashCache = append(newHashCache, HashCache{Path: path, Hash: hash})
		hashCacheLock.Unlock()
		return
	}

	log.Printf("rebuilding %v", path)

	// Archiving into zip file

	cmd = exec.Command("zip",
		fmt.Sprintf("%v/archive.zip", path),
		fmt.Sprintf("%v/main", path),
	)

	err = cmd.Run()

	if err != nil {
		log.Printf("Error archiving - %v: %v", path, err)
		return
	}

	f, err = readFile(path + "/main")
	hash = hashBytes(f)
	if err != nil {
		log.Println(err)
	}

	d := HashCache{Path: path, Hash: hash}

	hashCacheLock.Lock()
	newHashCache = append(newHashCache, d)
	hashCacheLock.Unlock()
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func readFile(filename string) (b []byte, err error) {
	info, err := os.Stat(filename)
	if err != nil {
		return b, err
	}

	b = make([]byte, info.Size())
	f, err := os.Open(filename)
	if err != nil {
		return b, err
	}

	_, err = f.Read(b)
	if err != nil {
		return b, err
	}

	return b, err
}

func hashBytes(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}

func getHashFromCache(path string) ([]byte, error) {
	for _, hash := range hashCache {
		if hash.Path == path {
			return hash.Hash, nil
		}
	}

	return *new([]byte), errors.New("path not found in hashCache")
}
