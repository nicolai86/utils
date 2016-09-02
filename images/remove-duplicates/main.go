package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/nicolai86/fshelpers/images/remove-duplicates/Godeps/_workspace/src/github.com/MichaelTJones/walk"
)

// only files matching these extensions are evaluated
var extensions = map[string]struct{}{
	"jpeg": struct{}{},
	"jpg":  struct{}{},
	"png":  struct{}{},
	"nef":  struct{}{},
	"tiff": struct{}{},
}

type dirWalker struct {
	root    string
	recurse bool
	c       chan string
	w       sync.WaitGroup
	mu      sync.Mutex
	files   map[string][]string
}

func (d *dirWalker) hash(p string) {
	f, err := os.Open(p)
	if err != nil {
		return
	}

	h := md5.New()
	io.Copy(h, f)
	f.Close()

	hash := fmt.Sprintf("%x", h.Sum(nil))
	d.mu.Lock()
	if _, ok := d.files[hash]; !ok {
		d.files[hash] = []string{p}
	} else {
		d.files[hash] = append(d.files[hash], p)
	}
	d.mu.Unlock()
}

// runs from inside a gorountine
func (d *dirWalker) walk(p string, info os.FileInfo, err error) error {
	stat, err := os.Stat(p)
	if err != nil {
		return err
	}

	if stat.IsDir() && p != d.root && !d.recurse {
		return filepath.SkipDir
	}

	var parts = strings.Split(p, ".")
	if _, ok := extensions[strings.ToLower(parts[len(parts)-1])]; ok {
		d.w.Add(1)
		d.c <- p
	}
	return nil
}

// runs from inside a goroutine
func (d *dirWalker) process() {
	for p := range d.c {
		d.hash(p)
		d.w.Done()
	}
}

type stringList []string

func (a stringList) Len() int           { return len(a) }
func (a stringList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a stringList) Less(i, j int) bool { return len(a[i]) < len(a[j]) }

func main() {
	var (
		walkPath, _ = os.Getwd()
	)
	var walker = &dirWalker{walkPath, true, make(chan string), sync.WaitGroup{}, sync.Mutex{}, map[string][]string{}}
	go walker.process()
	go walker.process()
	go walker.process()

	if err := walk.Walk(walkPath, walker.walk); err != nil {
		log.Fatal(err)
	}
	walker.w.Wait()

	for _, files := range walker.files {
		if len(files) < 2 {
			continue
		}

		sort.Sort(stringList(files))
		fmt.Printf("got %d duplicates. keeping %q\n", len(files), files[0])
		for _, file := range files[1:] {
			os.Remove(file)
			fmt.Printf("\tdeleting %q\n", file)
		}
	}
	fmt.Printf("Done\n")
}
