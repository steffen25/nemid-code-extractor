package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/otiai10/gosseract"
	"github.com/schollz/closestmatch"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

var codes = make(map[string]string)
var imageFile string
var key string

func main() {
	setupFlags()
	log.Println("image path", imageFile)
	f, err := os.Open(imageFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	hash := hex.EncodeToString(h.Sum(nil))
	dir := os.TempDir()
	fileName := fmt.Sprintf("%s/%s.json", dir, hash)

	// check if the parsed OCR file exists so we dont have to create it again and rely on its data
	if _, err := os.Stat(fileName); err == nil {
		file, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(file, &codes)
		if err != nil {
			log.Fatal(err)
		}
		val, err := findNemIdKey(key)
		if err != nil {
			log.Fatal(err)
		}
		err = setClipboardValue(val)
		if err != nil {
			log.Printf("Could not set clipboard value %v", err)
		}
		fmt.Printf("Key: %s \nCode: %s - Copied to clipboard", key, val)
		return
	}

	client := gosseract.NewClient()
	err = client.SetImage(imageFile)
	if err != nil {
		log.Fatal(err)
	}
	text, _ := client.Text()
	err = client.Close()
	if err != nil {
		log.Fatal(err)
	}
	words := strings.Fields(text)
	for idx, word := range words {
		if len(words)-1 > idx && len(word) == 4 {
			if _, err := strconv.Atoi(word); err == nil {
				key := word
				code := words[idx+1]
				codes[key] = code
			}
		}
	}

	jsonString, _ := json.Marshal(codes)
	err = ioutil.WriteFile(fileName, jsonString, 0644)
	if err != nil {
		log.Fatal(err)
	}

	val, err := findNemIdKey(key)
	if err != nil {
		log.Fatal(err)
	}

	err = setClipboardValue(val)
	if err != nil {
		log.Printf("Could not set clipboard value %v", err)
	}
	fmt.Printf("Key: %s \nCode: %s \n Code copied to clipboard", key, val)
}

// findNemIdKey will find a specific key and returns its value if it exists or return an error
func findNemIdKey(key string) (string, error) {
	if _, ok := codes[key]; !ok {
		keys := make([]string, 0, len(codes))
		for k := range codes {
			keys = append(keys, k)
		}
		bagSizes := []int{2, 3, 4}

		cm := closestmatch.New(keys, bagSizes)

		match := cm.Closest(key)
		var msg string
		if match != "" {
			msg = fmt.Sprintf("Could not find key %s did you mean %s", key, match)
		} else {
			msg = fmt.Sprintf("Could not find key %s", key)
		}

		return "", errors.New(msg)
	}

	return codes[key], nil
}

// setupFlags will setup all the necessary flags for the CLI to run
func setupFlags() {
	flag.StringVar(&imageFile, "image", "", "Path to a NemID image file 125% zoom level")
	flag.StringVar(&key, "key", "", "NemID 4 digit key")
	flag.Parse()
	if imageFile == "" {
		log.Fatal("You must provide a NemID image path using the --image option")
	}
	if _, err := os.Stat(imageFile); os.IsNotExist(err) {
		log.Fatalf("image file does not exists %s", imageFile)
	}
	if key == "" {
		log.Fatal("You must provide a 4 digit key using the --key option")
	}
}

func setClipboardValue(value string) error {
	err := clipboard.WriteAll(value)
	return err
}