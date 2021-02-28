package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"gopkg.in/yaml.v2"
)

type HashCache struct {
	Cache []Cache
	lock sync.RWMutex
}

type Cache struct {
	Path string `json:"path"`
	Hash []byte `json:"hash"`
}

type Config struct {
	Backend struct {
		Config struct {
			Bucket string `yaml:"bucket"`
			Key    string `yaml:"key"`
			Region string `yaml:"region"`
		}
	}
	Functions []struct {
		Name string `yaml:"name"`
		Path string `yaml:"path"`
	}
}

var (
	config Config
	useLocalFile bool
)

func main() {

	// read function files
	b, err := readFile("functions.yaml")
	if err != nil {
		log.Println(err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Println(err)
	}

	// check if backend config is empty
	if config.Backend.Config.Bucket == "" || config.Backend.Config.Region == "" {
		useLocalFile = true
	}

	// gets hashCache from local or s3
	cache, err := GetCache()
	if err != nil {
		log.Println(err)
	}

	var newCache HashCache

	var wg sync.WaitGroup

	for _, function := range config.Functions {
		wg.Add(1)
		go build(&cache, &newCache, function.Name, function.Path, &wg)
	}

	// wait till all goroutines are finished
	wg.Wait()

	// saves hashCache to local or s3
	err = newCache.Save()
	if err != nil {
		log.Println(err)
	}

}

func build(cache *HashCache, newCache *HashCache, name string, path string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Compiling go binary

	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH","amd64")
	os.Setenv("CGO_ENABLED", "0")

	cmd := exec.Command(
		"go",
		"build",
		"-o",
		fmt.Sprintf("%v/main", path),
		fmt.Sprintf("%v/main.go", path),
	)

	err := cmd.Run()

	if err != nil {
		log.Printf(ErrCompiling, name, err)
		return
	}

	// Cache stuff
	cachedHash, err := cache.getHashFromCache(path)
	if err != nil {
		log.Println(err)
	}

	f, err := readFile(path + "/main")
	hash := hashBytes(f)
	if err != nil {
		log.Println(err)
	}

	if string(cachedHash) == string(hash) {
		d := Cache{Path: path, Hash: hash}

		newCache.appendToCache(d)
		return
	}

	log.Printf(InfoRebuilding, name)

	// Archiving into zip file
	cmd = exec.Command("zip",
		fmt.Sprintf("%v/archive.zip", path),
		fmt.Sprintf("%v/main", path),
	)

	err = cmd.Run()

	if err != nil {
		log.Printf(ErrArchiving, name, err)
		return
	}

	f, err = readFile(path + "/main")
	hash = hashBytes(f)
	if err != nil {
		log.Println(err)
	}


	d := Cache{Path: path, Hash: hash}
	newCache.appendToCache(d)
}
