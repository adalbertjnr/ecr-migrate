package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Repositories struct {
	Path string
	List []string `yaml:"repositories"`
}

func newRepositoryFinder() *Repositories {
	return &Repositories{}
}

func (r *Repositories) locateIn(path string) *Repositories {
	r.Path = path
	return r
}

func (r *Repositories) registryList() *Repositories {
	data := &Repositories{}

	b, err := os.ReadFile(r.Path)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(b, data); err != nil {
		panic(err)
	}

	return data
}
