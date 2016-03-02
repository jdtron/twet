// -*- tab-width: 4; -*-

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v1"
)

type Config struct {
	Nick      string
	Twturl    string
	Twtfile   string
	Following map[string]string
}

func (c *Config) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	if c.Nick == "" || c.Twturl == "" {
		return errors.New("both nick and twturl must be set!")
	}
	return nil
}

func (c *Config) Read() string {
	var paths []string
	if xdg := os.Getenv("XDG_BASE_DIR"); xdg != "" {
		paths = append(paths, fmt.Sprintf("%s/config/twet", xdg))
	}
	paths = append(paths, fmt.Sprintf("%s/config/twet", homedir))
	paths = append(paths, fmt.Sprintf("%s/Library/Application Support/twet", homedir))
	paths = append(paths, fmt.Sprintf("%s/.twet", homedir))

	filename := "config.yaml"

	foundpath := ""
	for _, path := range paths {
		configfile := fmt.Sprintf("%s/%s", path, filename)
		data, err := ioutil.ReadFile(configfile)
		if err != nil {
			// try next path
			continue
		}
		if err := c.Parse(data); err != nil {
			log.Fatal(fmt.Sprintf("error parsing config file: %s: %s", filename, err))
		}
		foundpath = path
		break
	}
	if foundpath == "" {
		log.Fatal(fmt.Sprintf("config file %q not found; looked in: %q", filename, paths))
	}
	return foundpath
}
