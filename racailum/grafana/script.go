package grafana

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	scriptFileExt     = ".js"
	scriptFileExtLen  = len(scriptFileExt)
	scriptFilePattern = "*" + scriptFileExt
)

type script struct {
	collection string
	action     string
	code       string
}

func newScriptLoader(dir string) (*scriptLoader, error) {
	return &scriptLoader{
		pattern: filepath.Join(dir, scriptFilePattern),
	}, nil
}

type scriptLoader struct {
	pattern string
}

func (s *scriptLoader) loadAll(ctx context.Context) ([]script, error) {
	matches, err := filepath.Glob(s.pattern)
	if err != nil {
		return nil, fmt.Errorf("search for matched files: %w", err)
	}

	contents := make([]script, 0, len(matches))
	for _, match := range matches {
		fname := filepath.Base(match)
		pieces := strings.SplitN(fname[:len(fname)-scriptFileExtLen], ".", 2)
		col := strings.TrimSpace(pieces[0])

		if col == "" {
			log.Warnw("ignore invalid json file", "path", match)
			continue
		}

		var action string
		if len(pieces) > 1 {
			action = pieces[1]
		}

		content, err := ioutil.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("read content of %s: %w", match, err)
		}

		contents = append(contents, script{
			collection: col,
			action:     action,
			code:       string(content),
		})
	}

	return contents, nil
}
