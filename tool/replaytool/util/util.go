package util

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/lotus/api"
	"io/ioutil"
)

func WriteTofile(invocResult []*api.InvocResult) error {
	var filename = "replay.txt"

	fileContent, err := json.Marshal(invocResult)
	if err = ioutil.WriteFile(filename, fileContent, 0666); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}
