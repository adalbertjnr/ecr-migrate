package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"math/rand"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"gopkg.in/yaml.v2"
)

var (
	seed   = rand.NewSource(time.Now().UnixNano())
	random = rand.New(seed)
)

func randomSuffix() string {
	return strings.TrimPrefix(fmt.Sprintf("%.10f", random.Float64()), "0.")
}

func createTempYamlList() (string, []string, error) {
	file, err := os.CreateTemp("", "file*.yaml")
	if err != nil {
		return "", nil, err
	}

	repo := struct {
		Repositories []string `yaml:"repositories"`
	}{
		Repositories: []string{
			"repo/test/" + randomSuffix(),
			"repo/test/" + randomSuffix(),
		},
	}

	b, err := yaml.Marshal(repo)
	if err != nil {
		return "", nil, err
	}

	if _, err := file.Write(b); err != nil {
		return "", nil, err
	}

	if err := file.Close(); err != nil {
		return "", nil, err
	}

	return file.Name(), repo.Repositories, nil
}

func ecrClients(ecrConfig ecrConfigs) (*ResoureceConfig, *ResoureceConfig) {
	awsFrom := mustInitConfig(
		withRegion(ecrConfig.fromRegion),
		withProfile(ecrConfig.fromProfile),
	)

	svcFrom := awsFrom.stablishClientWith(
		ecrService(awsFrom.cfg),
		stsService(awsFrom.cfg),
	)

	awsTo := mustInitConfig(
		withRegion(ecrConfig.toRegion),
		withProfile(ecrConfig.toProfile),
	)

	svcTo := awsTo.stablishClientWith(
		ecrService(awsTo.cfg),
		stsService(awsTo.cfg),
	)

	return svcFrom, svcTo
}

func delete(e *ecr.Client, repositoryNames []string) error {
	for _, repositoryName := range repositoryNames {
		_, err := e.DeleteRepository(context.Background(), &ecr.DeleteRepositoryInput{
			RepositoryName: aws.String(repositoryName),
			Force:          true,
		})
		if err != nil {
			return err
		}

		slog.Info("ecrDelete", "repositoryName", repositoryName, "status", "deleted")
	}
	return nil
}

func createImageURI(s *sts.Client, region string, repoList []string) map[string]string {
	resp, err := s.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}

	m := make(map[string]string, len(repoList))
	for _, repo := range repoList {
		uri := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", *resp.Account, region, repo)
		m[repo] = uri
	}
	return m
}

func generateConfigs(repositories []string, uriImages map[string]string) []ecrImageConfig {
	ecrImageConfigs := make([]ecrImageConfig, len(repositories))
	for _, repo := range repositories {
		uri := uriImages[repo]
		ecrImageConfigs = append(ecrImageConfigs, ecrImageConfig{
			repository:    repo,
			repositoryURI: uri,
			pullImage:     []string{"alpine:latest", "alpine:latest"},
			generateTags:  []string{"1.0", "1.1"},
		})
	}

	return ecrImageConfigs
}
