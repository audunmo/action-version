package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/audunmo/action-version/internal/files"
	"github.com/briandowns/spinner"
)

func process(fileNames []string, status chan string, done chan bool) error {
	errs := make(chan error, len(fileNames))
	var wg sync.WaitGroup
	for _, file := range fileNames {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			status <- fmt.Sprintf("Updating %s", file)
			err := files.UpdateFile(file, status, os.WriteFile)
			if err != nil {
				errs <- err
			}
		}(file)
	}
	wg.Wait()

	done <- true
	close(errs)
	return <-errs
}

func handleSpinner(cancel func(), status chan string, done chan bool) {
	go func() {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Color("yellow")
		s.FinalMSG = ""
		s.Start()
		for {
			select {
			case ss := <-status:
				s.Lock()
				s.Suffix = ss
				s.Unlock()
			case <-done:
				s.Stop()
				cancel()
				close(done)
				return
			}
		}
	}()
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var allFiles bool
	flag.BoolVar(&allFiles, "all", false, "Update all files in the directory, not just yaml/yml files")

	ff, err := files.ListDir(dir, allFiles)
	if err != nil {
		panic(err)
	}

	if len(ff) == 0 {
		panic("No .md, .yaml or .yml files found. Please run this command in a directory with .yaml or .yml Github Action files.")
	}

	// Receive status updates from the process function, to push to the spinner.
	status := make(chan string)
	// Channel to stop the spinner. Indicates all processing is done
	done := make(chan bool)

	var wg sync.WaitGroup
	wg.Add(1)
	handleSpinner(func() { wg.Done() }, status, done)

	err = process(ff, status, done)
	if err != nil {
		panic(err)
	}

	wg.Wait()

	fmt.Printf("🚀 ✅ Successfully pinned Github Actions versions in %d files\n", len(ff))
}
