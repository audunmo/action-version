package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
)

func ListDir(pwd string) ([]string, error) {
	dirFiles, err := os.ReadDir(pwd)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, file := range dirFiles {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".md") {
			files = append(files, file.Name())
		}
	}
	return files, nil
}

func ReadFile(fileName string) (string, error) {
	ff, err := os.Open(fileName)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(ff)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

type SemverAction struct {
	Action  string
	Version string
	Full    string
}

var usesRegex = regexp.MustCompile(`(m?)uses: (.+)@(v.+)`)

// FindSemverActions finds all actions in a file that match patterns like:
//
//	uses: actions/checkout@v2
//
// It returns the matches, or an error if there was a problem parsing the file.
func FindSemverActions(fileContents string) ([]SemverAction, error) {
	stringMatches := usesRegex.FindAllStringSubmatch(fileContents, -1)

	var matches []SemverAction
	for _, m := range stringMatches {
		mm := SemverAction{
			Full:    m[0],
			Action:  m[2],
			Version: m[3],
		}

		matches = append(matches, mm)
	}
	return matches, nil
}

type HashGetter struct {
	memory *memory.Storage
	repos  map[string]*git.Repository
	mut    *sync.Mutex
}

func NewHashGetter() *HashGetter {
	return &HashGetter{
		memory: memory.NewStorage(),
		repos:  make(map[string]*git.Repository),
		mut:    &sync.Mutex{},
	}
}

func (h *HashGetter) GetHashForAction(a SemverAction) (string, error) {
	h.mut.Lock()
	defer h.mut.Unlock()
	repo, ok := h.repos[a.Action]
	if !ok || repo == nil {
		var err error
		repo, err = git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
			URL: "https://github.com/" + a.Action,
		})

		if err != nil && !errors.Is(err, git.ErrRepositoryAlreadyExists) {
			return "", err
		}
	}

	tag, err := repo.Tag(a.Version)
	if err != nil {
		return "", err
	}

	h.repos[a.Action] = repo

	return tag.Hash().String(), nil
}

type Update struct {
	FileName   string
	OldVersion string
	NewVersion string
	FullMatch  string
}

func UpdateFile(fileName string, status chan string, writer func(string, []byte, os.FileMode) error) error {
	fileContents, err := ReadFile(fileName)
	if err != nil {
		return err
	}

	matches, err := FindSemverActions(fileContents)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return nil
	}
	errs := make(chan error, 1)
	updates := make(chan Update, len(matches)-1)

	hashGetter := NewHashGetter()
	for _, a := range matches {
		go func(a SemverAction) {
			status <- fmt.Sprintf("  (%s) Finding hash for %s@%s\n", fileName, a.Action, a.Version)
			hash, err := hashGetter.GetHashForAction(a)
			// Multiple action files may pull the same action, either at the same or different versions. This causes ErrRepositoryAlreadyExists error. This doesnt represent an error in this case.
			if err != nil && !errors.Is(err, git.ErrRepositoryAlreadyExists) {
				errs <- err
				return
			}

			if hash == "" {
				fmt.Println("Could not find hash for ", a.Action, a.Version)
				return
			}

			status <- fmt.Sprintf("  (%s) Updating with hash %s for %s@%s\n", fileName, hash, a.Action, a.Version)
			updates <- Update{
				FileName:   fileName,
				OldVersion: a.Version,
				NewVersion: hash,
				FullMatch:  a.Full,
			}
		}(a)
	}

	updated := 0
	content := fileContents
	for {
		select {
		case u := <-updates:
			content = replace(u, content)
			updated += 1
			if updated != len(matches) {
				continue
			}

			err := writer(fileName, []byte(content), 0644)
			if err != nil {
				errs <- err
				continue
			}

			return nil
		case err := <-errs:
			return err
		}
	}
}

func replace(u Update, fileContents string) string {
	newContent := strings.ReplaceAll(u.FullMatch, u.OldVersion, u.NewVersion) + " # " + u.OldVersion
	return strings.ReplaceAll(fileContents, u.FullMatch, newContent)
}
