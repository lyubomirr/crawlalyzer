package analyzer

import (
	"encoding/json"
	"fmt"
)

type category struct {
	Name string `json:"name"`
}

//The json could contain only a single string or array of strings
//so we always parse it to a slice with custom marshalling
type singleOrArray []string

func (s *singleOrArray) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("no bytes to unmarshal")
	}

	if s == nil {
		*s = singleOrArray{}
	}

	if b[0] == '[' {
		slice := make([]string, 0, len(b))
		json.Unmarshal(b, &slice)
		*s = slice
	} else {
		*s = append(*s, string(b))
	}

	return nil
}

type technology struct {
	Cats        []int             `json:"cats"`
	Description string            `json:"description"`
	Cookies     map[string]string `json:"cookies"`
	Headers     map[string]string `json:"headers"`
	Website     string            `json:"website"`
	JS          map[string]string `json:"js"`
	HTML        singleOrArray     `json:"html"`
	Scripts     singleOrArray     `json:"scripts"`
	Excludes    singleOrArray     `json:"excludes"`
	Implies     singleOrArray     `json:"implies"`
	CSS         singleOrArray     `json:"css"`
}

type technologyPrints struct {
	Categories   map[string]category   `json:"categories"`
	Technologies map[string]technology `json:"technologies"`
}

type marshalabeResult struct {
	DetectedTechnologies []DetectedTechnology
	Implies              []string
	Excludes             []string
	Categories           []string
}