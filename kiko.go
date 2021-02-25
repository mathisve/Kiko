package main

import (
	"fmt"
	"log"
	"os/exec"
	"sync"

	"gopkg.in/yaml.v2"
)

const HashcacheFilename = ".kikoCache.json"

type HashCache struct {
	Cache []Cache
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

var config Config

var hashCache HashCache

var newHashCache HashCache

var useLocalFile bool
var hashCacheLock sync.RWMutex

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

	if config.Backend.Config.Bucket == "" || config.Backend.Config.Region == "" {
		useLocalFile = true
	}

	// gets hashCache from local or s3
	err = GetCache()

	var wg sync.WaitGroup

	for _, function := range config.Functions {
		wg.Add(1)
		go build(function.Name, function.Path, &wg)
	}

	wg.Wait()

	// saves hashCache to local or s3
	err = SaveCache()
	if err != nil {
		log.Println(err)
	}

}

func build(name string, path string, wg *sync.WaitGroup) {
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
		log.Print(ErrCompiling, name, err)
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
		d := Cache{Path: path, Hash: hash}

		newHashCache.appendToCache(d)
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
	newHashCache.appendToCache(d)
}
