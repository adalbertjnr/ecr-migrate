package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type ECR struct {
	ecr *ecr.Client
	ctx context.Context
}

func newEcr(ecr *ecr.Client) *ECR {
	return &ECR{
		ecr: ecr,
		ctx: context.Background(),
	}
}

type metadataList struct {
	auth        authorization
	repoList    []repositoryMetadata
	imagesCount int
}

type repositoryMetadata struct {
	repositoryURI    string
	repositoryPolicy string
	repositoryName   string
	tags             []string
}

func (e *ECR) getRepositoryMetadata(repoList []string) map[string]repositoryMetadata {
	m := make(map[string]repositoryMetadata, len(repoList))
	resp, err := e.ecr.DescribeRepositories(e.ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: repoList,
	})
	if err != nil {
		panic(err)
	}

	for _, repo := range resp.Repositories {
		m[*repo.RepositoryName] = repositoryMetadata{
			repositoryName:   *repo.RepositoryName,
			repositoryURI:    *repo.RepositoryUri,
			repositoryPolicy: e.pullPolicy(*repo.RepositoryName),
		}
	}

	return m
}

func (e *ECR) pullPolicy(repositoryName string) string {
	resp, err := e.ecr.GetRepositoryPolicy(e.ctx, &ecr.GetRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
	})

	if err != nil {
		var policyNotFoundErr *types.RepositoryPolicyNotFoundException
		if errors.As(err, &policyNotFoundErr) {
			return ""
		} else {
			slog.Error("pullPolicy", "error", err, "repository", repositoryName)
			return ""
		}
	}

	return *resp.PolicyText
}

type authorization struct {
	username string
	password string
}

func (e *ECR) authenticate() (authorization, error) {
	authOutput, err := e.ecr.GetAuthorizationToken(e.ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return authorization{}, err
	}

	if len(authOutput.AuthorizationData) == 0 {
		return authorization{}, fmt.Errorf("no authorizationData found in the response")
	}

	authData := authOutput.AuthorizationData[0]
	token, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return authorization{}, fmt.Errorf("decoding auth token not possible")
	}

	auth := strings.Split(string(token), ":")

	return authorization{
		username: auth[0],
		password: auth[1],
	}, nil
}

func (e *ECR) walk(repoList []string) metadataList {
	data := e.getRepositoryMetadata(repoList)

	auth, err := e.authenticate()
	if err != nil {
		panic(err)
	}

	metadata := metadataList{
		auth: auth,
	}

	counter := 0
	for _, repository := range repoList {
		list, err := e.ecr.ListImages(e.ctx, &ecr.ListImagesInput{
			RepositoryName: aws.String(repository),
		})

		if err != nil {
			slog.Error("listing ecr images", "error", err)
			continue
		}

		tags := make([]string, len(list.ImageIds))
		for i, image := range list.ImageIds {
			if *image.ImageTag != "" {
				tags[i] = *image.ImageTag
				slog.Info("ecrListing", "repository", repository, "tag", *image.ImageTag)
				counter++
			}
		}

		if repositoryValue, found := data[repository]; found {
			repositoryValue.tags = tags
			metadata.repoList = append(metadata.repoList, repositoryValue)
		}
	}

	metadata.imagesCount = counter
	return metadata
}

func (e *ECR) exists(repository string) bool {
	_, err := e.ecr.DescribeRepositories(e.ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repository},
	})
	if err != nil {
		var notFoundErr *types.RepositoryNotFoundException
		if errors.As(err, &notFoundErr) {
			return false
		}
	}
	return true
}

func (e *ECR) create(repository, policy string) error {
	_, err := e.ecr.CreateRepository(e.ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repository),
	})
	if err != nil {
		slog.Error("ecrCreate", "error", err)
		return err
	}
	slog.Info("ecrCreate", "repositoryName", repository, "status", "created")
	return e.setPolicy(repository, policy)
}

func (e *ECR) validate(metadata metadataList) []string {
	repositoryList := make([]string, len(metadata.repoList))
	for i, metadata := range metadata.repoList {
		repositoryList[i] = metadata.repositoryName
		if !e.exists(metadata.repositoryName) {
			if err := e.create(metadata.repositoryName, metadata.repositoryPolicy); err != nil {
				slog.Error("ecr", "error", err)
				continue
			}
		} else {
			slog.Info("ecrCreate", "repository", metadata.repositoryName, "status", "already exists")
		}
	}

	return repositoryList
}

func (e *ECR) setPolicy(repositoryName, repositoryPolicy string) error {
	var err error
	if repositoryPolicy != "" {
		_, err = e.ecr.SetRepositoryPolicy(e.ctx, &ecr.SetRepositoryPolicyInput{
			RepositoryName: aws.String(repositoryName),
			PolicyText:     aws.String(repositoryPolicy),
		})
	}
	return err
}
