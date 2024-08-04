package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageFinder(t *testing.T) {
	repoFinder := newRepositoryFinder()

	fileName, repoList, err := createTempYamlList()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName)

	repositories := repoFinder.locateIn(fileName).registryList()
	assert.Equal(t, repositories.List, repoList)
}
