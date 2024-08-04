package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECRClients(t *testing.T) {
	ecrConfig := struct {
		fromRegion  string
		toRegion    string
		fromProfile string
		toProfile   string
	}{
		fromRegion:  "us-east-1",
		toRegion:    "us-east-1",
		fromProfile: "default",
		toProfile:   "HOME-LAB",
	}

	awsFrom := mustInitConfig(
		withRegion(ecrConfig.fromRegion),
		withProfile(ecrConfig.fromProfile),
	)

	svcFrom := awsFrom.stablishClientWith(
		ecrService(awsFrom.cfg),
	)

	awsTo := mustInitConfig(
		withRegion(ecrConfig.toRegion),
		withProfile(ecrConfig.toProfile),
	)

	svcTo := awsTo.stablishClientWith(
		ecrService(awsTo.cfg),
	)

	assert.NotEmpty(t, svcFrom.ecr)
	assert.NotEmpty(t, svcTo.ecr)
}

func TestWalk(t *testing.T) {
	expected := struct {
		RepoListLen int
		UserName    string
		ImageCount  int
	}{
		ImageCount:  0,
		RepoListLen: 2,
		UserName:    "AWS",
	}

	ecrConfig := struct {
		fromRegion  string
		fromProfile string
	}{
		fromRegion:  "us-east-1",
		fromProfile: "default",
	}

	fileName, _, err := createTempYamlList()
	if err != nil {
		t.Fatal(err)
	}

	repoFinder := newRepositoryFinder()
	repositories := repoFinder.locateIn(fileName).registryList()

	awsFrom := mustInitConfig(
		withRegion(ecrConfig.fromRegion),
		withProfile(ecrConfig.fromProfile),
	)

	svcFrom := awsFrom.stablishClientWith(
		ecrService(awsFrom.cfg),
		stsService(awsFrom.cfg),
	)

	ecrRegistry := newEcr(svcFrom.ecr)
	imagesURIMap := createImageURI(svcFrom.sts, ecrConfig.fromRegion, repositories.List)

	for _, repository := range repositories.List {
		if err := ecrRegistry.create(repository, ""); err != nil {
			t.Fatal(err)
		}
	}

	imageMetadataList := ecrRegistry.walk(repositories.List)

	assert.Len(t, imageMetadataList.repoList, expected.RepoListLen, "expected list length %d but got %d", expected.RepoListLen, len(imageMetadataList.repoList))
	assert.Equal(t, expected.UserName, imageMetadataList.auth.username, "expected username %s but got %s", expected.UserName, imageMetadataList.auth.username)
	assert.NotEmpty(t, imageMetadataList.auth.password, "expected non-empty password but got empty")
	assert.Equal(t, expected.ImageCount, imageMetadataList.imagesCount, "expected image count %d but got %d", expected.ImageCount, imageMetadataList.imagesCount)

	for _, imageMetadata := range imageMetadataList.repoList {
		if uri, found := imagesURIMap[imageMetadata.repositoryName]; found {
			assert.Equal(t, uri, imageMetadata.repositoryURI, "expected uri %s but got %s", uri, imageMetadata.repositoryURI)
		}
	}

	delete(svcFrom.ecr, repositories.List)
}
