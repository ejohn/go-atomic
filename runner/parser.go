package runner

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/ejohn/go-atomic/art"
)

// parse is factored out to make yaml unmarshalling easily testable.
func parse(content []byte) (art.Technique, error) {
	var technique art.Technique
	err := yaml.Unmarshal(content, &technique)
	if err != nil {
		return art.Technique{}, err
	}
	return technique, nil
}

// parseYAMLFile parses a yaml technique file and unmarshall's the contents into a struct.
func parseYAMLFile(file string) (art.Technique, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return art.Technique{}, err
	}
	return parse(content)
}
