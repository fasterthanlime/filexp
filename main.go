package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/itchio/headway/united"
	"github.com/logrusorgru/aurora"
	"golang.org/x/sys/windows"
)

var (
	FileName       = "test.dat"
	FileSize int64 = 512 * 1024 * 1024
)

func main() {
	log.SetFlags(log.Lmicroseconds)

	log.Printf("testing with %s file", united.FormatBytes(FileSize))

	log.Printf("")
	log.Printf("========== fast ===========")
	beforeFast := time.Now()
	run(true)
	fastDuration := time.Since(beforeFast)

	log.Printf("")
	log.Printf("========== slow ===========")
	beforeSlow := time.Now()
	run(false)
	slowDuration := time.Since(beforeSlow)

	log.Printf("")
	log.Printf("========== stats ==========")
	ratio := fastDuration.Seconds() / slowDuration.Seconds()
	log.Printf("fast took %.5fx the time it took slow", ratio)

	err := os.RemoveAll(FileName)
	must(err)
}

func run(fast bool) {
	err := os.RemoveAll(FileName)
	must(err)

	f, err := os.Create(FileName)
	must(err)

	{
		s, err := f.Stat()
		must(err)
		if s.Size() != 0 {
			must(fmt.Errorf("expected size to be 0"))
		}
	}

	{
		before := time.Now()
		preallocate(f, fast)
		msg := fmt.Sprintf("Preallocate took %s", time.Since(before))
		log.Println(aurora.Bold(aurora.Red(msg)))
	}

	err = f.Close()
	must(err)

	{
		s, err := os.Stat(FileName)
		must(err)
		if s.Size() != FileSize {
			must(fmt.Errorf("expected size to be %d", FileSize))
		}
	}

	f, err = os.OpenFile(FileName, os.O_RDWR, 0644)
	must(err)

	{
		s, err := f.Stat()
		must(err)
		if s.Size() != FileSize {
			must(fmt.Errorf("expected size to be %d", FileSize))
		}
	}

	i, err := f.Seek(0, io.SeekEnd)
	must(err)
	if i != FileSize {
		must(fmt.Errorf("expected end seek to be %d", i))
	}

	i, err = f.Seek(0, io.SeekStart)
	must(err)
	if i != 0 {
		must(fmt.Errorf("expected start seek to be 0"))
	}

	{
		before := time.Now()

		wr, err := f.Write([]byte("hello!"))
		must(err)
		log.Printf("Wrote %d bytes", wr)

		msg := fmt.Sprintf("Write took %s", time.Since(before))
		log.Println(aurora.Bold(aurora.Red(msg)))
	}

	err = f.Close()
	must(err)
	log.Printf("Closed")
}

func preallocate(f *os.File, fast bool) {
	desc := "slowly"
	if fast {
		desc = "quickly"
	}
	log.Printf("Preallocating %s", desc)

	if fast {
		_, err := f.Seek(FileSize, io.SeekStart)
		must(err)

		err = windows.SetEndOfFile(windows.Handle(f.Fd()))
		must(err)

		log.Printf("Called SetEndOfFile")
	} else {
		i, err := f.Seek(0, io.SeekEnd)
		must(err)

		remaining := FileSize - i
		buf := make([]byte, 16*1024) // 16K

		for remaining > 0 {
			wbuf := buf
			if int64(len(wbuf)) > remaining {
				wbuf = wbuf[:remaining]
			}

			_, err = f.Write(wbuf)
			must(err)

			remaining -= int64(len(wbuf))
		}

		log.Printf("Zeroed %d bytes", FileSize)
	}
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
}
