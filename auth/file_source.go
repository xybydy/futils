package auth

import (
	"io/ioutil"
)

func ReadFile(path string) ([]byte, bool, error) {
	if !fileExists(path) {
		return nil, false, nil
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, true, err
	}
	return content, true, nil
}
