package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

func ListDir(directory string, recursive bool, vi map[string]bool) ([]string, error) {
	dirFiles, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, file := range dirFiles {
		if _, ok := vi[file.Name()]; ok {
			continue
		}

		vi[file.Name()] = true

		if file.IsDir() {
			if file.Name() == ".git" {
				continue
			}

			if !recursive {
				continue
			}

			dir := path.Join(directory, file.Name())
			ff, err := ListDir(dir, recursive, vi)
			if err != nil {
				return nil, err
			}

			files = append(files, ff...)
		}

		filepath := path.Join(directory, file.Name())

		if strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".md") {
			files = append(files, filepath)
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

var usesRegex = regexp.MustCompile(`uses: ([\w-]+/[\w-]+)(?:/[\w-]+)?@(v[\d.]+)`)

// FindSemverActions finds all actions in a file that match patterns like:
//
//	uses: actions/checkout@v2
//	uses: github/codeql-action/init@v3
//
// It returns the matches.
func FindSemverActions(fileContents string) []SemverAction {
	stringMatches := usesRegex.FindAllStringSubmatch(fileContents, -1)

	var matches []SemverAction
	for _, m := range stringMatches {
		mm := SemverAction{
			Full:    m[0],
			Action:  m[1],
			Version: m[2],
		}

		matches = append(matches, mm)
	}
	return matches
}

type HashGetter struct {
	hashes    map[string]string
	mut       *sync.Mutex
	authToken string
}

func NewHashGetter(authToken string) *HashGetter {
	return &HashGetter{
		hashes:    make(map[string]string),
		mut:       &sync.Mutex{},
		authToken: authToken,
	}
}

func (h *HashGetter) GetHashForAction(a SemverAction) (string, error) {
	h.mut.Lock()
	defer h.mut.Unlock()
	hash := h.hashes[a.Action]
	if hash != "" {
		return hash, nil
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: h.authToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	s := strings.Split(a.Action, "/")
	ref, _, err := client.Git.GetRef(ctx, s[0], s[1], "refs/tags/"+a.Version)
	if err != nil {
		return "", err
	}

	sha := ref.Object.GetSHA()
	h.hashes[a.Action] = sha
	return sha, nil
}

type Update struct {
	FileName   string
	OldVersion string
	NewVersion string
	FullMatch  string
}

func UpdateFile(fileName string, status chan string, writer func(string, []byte, os.FileMode) error, authToken string) error {
	fileContents, err := ReadFile(fileName)
	if err != nil {
		return err
	}

	matches := FindSemverActions(fileContents)
	if len(matches) == 0 {
		return nil
	}
	errs := make(chan error, 1)
	updates := make(chan Update, len(matches)-1)

	hashGetter := NewHashGetter(authToken)
	for _, a := range matches {
		go func(a SemverAction) {
			status <- fmt.Sprintf("  (%s) Finding hash for %s@%s\n", fileName, a.Action, a.Version)
			hash, err := hashGetter.GetHashForAction(a)
			if err != nil {
				errs <- err
				return
			}

			if hash == "" {
				fmt.Println("Could not find hash for ", a.Action, a.Version)
				return
			}

			status <- fmt.Sprintf("  (%s) Updated with hash %s for %s@%s\n", fileName, hash, a.Action, a.Version)
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
