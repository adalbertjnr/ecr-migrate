package main

import (
	"testing"
)

type (
	ecrConfigs struct {
		fromRegion  string
		toRegion    string
		fromProfile string
		toProfile   string
	}

	ecrImageConfig struct {
		repository    string
		repositoryURI string
		pullImage     []string
		generateTags  []string
	}
)

func TestMigrate(t *testing.T) {
	ecrConfig := ecrConfigs{
		fromRegion:  "us-east-1",
		fromProfile: "default",
		toRegion:    "us-east-1",
		toProfile:   "HOME-LAB",
	}

	fileName, _, err := createTempYamlList()
	if err != nil {
		t.Fatal(err)
	}

	var (
		repoFinder     = newRepositoryFinder()
		repositories   = repoFinder.locateIn(fileName).registryList()
		svcFrom, svcTo = ecrClients(ecrConfig)
		ecrRegistry    = newEcr(svcFrom.ecr)
		uriImages      = createImageURI(svcFrom.sts, ecrConfig.fromRegion, repositories.List)
	)

	auth, err := ecrRegistry.authenticate()
	if err != nil {
		t.Fatal(err)
	}

	for _, repository := range repositories.List {
		if err := ecrRegistry.create(repository, ""); err != nil {
			t.Fatal(err)
		}
	}

	args := &Args{
		fromRegion: ecrConfig.fromRegion,
		toRegion:   ecrConfig.toRegion,
		from:       ecrConfig.fromProfile,
		to:         ecrConfig.toProfile,
	}

	docker := newDocker().mustStartCli().withArgs(args)
	token := docker.authorize(auth)

	ecrImageConfigs := generateConfigs(repositories.List, uriImages)
	for _, ic := range ecrImageConfigs {
		for i, pullImage := range ic.pullImage {
			_, err := docker.pull("", downloadImage{name: pullImage})
			if err != nil {
				t.Fatal(err)
			}

			to := generateTargetImageName(ic.repositoryURI, ic.generateTags[i])
			if err := docker.rename(pullImage, to); err != nil {
				t.Fatal(err)
			}

			if err := docker.push(token, uploadImage{name: to}); err != nil {
				t.Fatal(err)
			}
		}
	}

	imageMetadataList := ecrRegistry.walk(repositories.List)
	docker.addMetadataList(imageMetadataList).migrate()

	if err := delete(svcFrom.ecr, repositories.List); err != nil {
		t.Fatal(err)
	}

	if err := delete(svcTo.ecr, repositories.List); err != nil {
		t.Fatal(err)
	}
}
