package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func sizeOf(file string) (int64, error) {
	st, err := os.Stat(file)
	if err != nil {
		return 0, err
	}

	return st.Size(), nil
}

func main() {
	file := flag.String("watch", "", "file to watch")
	pidfile := flag.String("pidfile", "/tmp/fnotify-watcher.pid", "file for pid checks")
	timeout := flag.Int("timeout", 10, "timeout for events")
	flag.Parse()

	if file == nil || *file == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if _, err := os.Stat(*pidfile); err == nil {
		log.Println("Looks like watcher is already running. Exiting...")
		os.Exit(1)
	}
	{
		pid := os.Getpid()
		f, _ := os.OpenFile(*pidfile, os.O_WRONLY|os.O_CREATE, 0755)
		fmt.Fprintf(f, "%d", pid)
		f.Close()
	}
	var cl = make(chan os.Signal, 1)
	signal.Notify(cl, os.Interrupt, os.Kill)
	go func() {
		<-cl
		err := os.Remove(*pidfile)
		log.Printf("delete %s returned %v\n", *pidfile, err)
		os.Exit(0)
	}()

	kq, err := syscall.Kqueue()
	if err != nil {
		log.Printf("Error creating Kqueue descriptor: %v\n", err.Error())
		return
	}

	fd, err := syscall.Open(*file, syscall.O_RDONLY, 0)
	if err != nil {
		log.Printf("Error opening folder descriptor: %v\n", err.Error())
		return
	}
	// build kevent
	ev1 := syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_VNODE,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
		Fflags: syscall.NOTE_WRITE | syscall.NOTE_EXTEND,
		Data:   0,
		Udata:  nil,
	}

	offset, _ := sizeOf(*file)
	f, err := os.Open(*file)
	if err != nil {
		log.Printf("Failed to open file for reading: %v\n", err.Error())
	}

	log.Println("started watching for events...")
	for {
		// create kevent
		events := make([]syscall.Kevent_t, 10)
		nev, err := syscall.Kevent(kq, []syscall.Kevent_t{ev1}, events, &syscall.Timespec{
			Sec:  int64(*timeout),
			Nsec: 0,
		})
		if err != nil {
			log.Println("Error creating kevent")
		}
		newSize, _ := sizeOf(*file)
		if nev == 0 {
			continue
		}
		line := ""
		if newSize > offset {
			ct := make([]byte, newSize-offset)
			f.ReadAt(ct, offset)
			line = string(ct)
		} else {
			ct := make([]byte, newSize)
			f.ReadAt(ct, 0)
			line = string(ct)
		}

		offset = newSize
		prts := strings.Split(line, " ")
		if len(prts) != 2 {
			continue
		}

		header := strings.TrimSpace(prts[0])
		body := strings.TrimSpace(prts[1])
		cmd := exec.Command("terminal-notifier", "-title", header, "-message", body, "-timeout", "5")
		cmd.Run()
	}
}
