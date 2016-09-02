package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nicolai86/fshelpers/images/group-by-exif/Godeps/_workspace/src/github.com/MichaelTJones/walk"
	"github.com/nicolai86/fshelpers/images/group-by-exif/Godeps/_workspace/src/github.com/rwcarlsen/goexif/exif"
)

// only files matching these extensions are evaluated
var extensions = map[string]struct{}{
	"jpeg": struct{}{},
	"jpg":  struct{}{},
	"png":  struct{}{},
	"nef":  struct{}{},
	"tiff": struct{}{},
}

var skipDirs = []string{
	"2000",
	"2001",
	"2002",
	"2003",
	"2004",
	"2005",
	"2006",
	"2007",
	"2008",
	"2009",
	"2010",
	"2011",
	"2012",
	"2013",
	"2014",
	"2015",
	"2016",
	"2017",
}

func imageCaptureDate(filename string) *time.Time {
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return nil
	}

	tm, _ := x.DateTime()
	if tm == (time.Time{}) {
		return nil
	}
	return &tm
}

type dirWalker struct {
	root    string
	recurse bool
	c       chan string
	w       sync.WaitGroup
}

func (d *dirWalker) move(p string) {
	var capturedAt = imageCaptureDate(p)
	if capturedAt == nil {
		log.Printf("missing exif %q", p)
		return
	}

	var destinationDir = fmt.Sprintf("%v/%d/%02d", d.root, capturedAt.Year(), capturedAt.Month())
	if err := os.MkdirAll(destinationDir, 0744); err != nil {
		log.Printf("unable to create dir %q", destinationDir)
		return
	}

	var destination = fmt.Sprintf("%v/%v", destinationDir, path.Base(p))
	if destination == p {
		return
	}

	_, err := os.Stat(destination)
	if err != nil && os.IsNotExist(err) {
		os.Rename(p, destination)
	} else {
		log.Printf("%q exists already (%q)", p, destination)
	}
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
		d.move(p)
		d.w.Done()
	}
}

func main() {
	skipDirs = append(skipDirs, os.Args[1:]...)
	fmt.Printf("ignoring %q\n", skipDirs)
	// TODO specify which folder to look at through flags
	// TODO specify where to place files through flags
	var (
		walkPath, _ = os.Getwd()
	)

	var walker = &dirWalker{walkPath, true, make(chan string), sync.WaitGroup{}}

	go walker.process()
	go walker.process()
	go walker.process()

	if err := walk.Walk(walkPath, walker.walk); err != nil {
		log.Fatal(err)
	}
	walker.w.Wait()
}
