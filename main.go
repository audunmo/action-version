package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

func ReadFile(fileName string) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

type semverAction struct {
	action  string
	version string
	full    string
}

var usesRegex = regexp.MustCompile(`(m?)uses: (.+)@(v.+)`)

func FindSemverActions(fileName string) ([]semverAction, string, error) {
	fileContents, err := ReadFile(fileName)
	if err != nil {
		return nil, "", err
	}

	stringMatches := usesRegex.FindAllStringSubmatch(fileContents, -1)

	var matches []semverAction
	for _, m := range stringMatches {
		mm := semverAction{
			full:    m[0],
			action:  m[2],
			version: m[3],
		}

		matches = append(matches, mm)
	}
	return matches, fileContents, nil
}

func GetHashForAction(m semverAction) (string, error) {
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: "https://github.com/" + m.action,
	})
	if err != nil {
		return "", err
	}

	tag, err := r.Tag(m.version)
	if err != nil {
		return "", err
	}

	return tag.Hash().String(), nil
}

func updateFile(fileName string) error {
	matches, fileContents, err := FindSemverActions(fileName)
	if err != nil {
		return err
	}
	updates := make(map[semverAction]string)

	for _, m := range matches {
		hash, err := GetHashForAction(m)
		if err != nil && !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return err
		}

		if hash == "" {
			fmt.Println("Could not find hash for ", m.action, m.version)
			continue
		}

		fmt.Printf("  (%s) Updating with hash %s for %s@%s\n", fileName, hash, m.action, m.version)

		updates[m] = strings.Replace(m.full, m.version, hash, 1) + " # " + m.version
	}

	for _, m := range matches {
		fileContents = strings.Replace(fileContents, m.full, updates[m], 1)
	}

	os.WriteFile(fileName, []byte(fileContents), 0644)

	return nil
}

func getFiles(pwd string, allFiles bool) ([]string, error) {
	dirFiles, err := os.ReadDir(pwd)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var files []string
	for _, file := range dirFiles {
		if allFiles {
			files = append(files, file.Name())
			continue
		}

		if strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml") {
			files = append(files, file.Name())
		}
	}
	return files, nil
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}

	var allFiles bool
	flag.BoolVar(&allFiles, "all", false, "Update all files in the directory, not just yaml/yml files")

	files, err := getFiles(dir, allFiles)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No .yaml or .yml files found. Please run this command in a directory with .yaml or .yml Github Action files.")
		return
	}

	s := spinner.New(spinner.CharSets[16], 100*time.Millisecond)
	s.Start()
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			err := updateFile(file)
			if err != nil {
				fmt.Println(err)
				return
			}
		}(file)
	}

	wg.Wait()
	s.Stop()
}
