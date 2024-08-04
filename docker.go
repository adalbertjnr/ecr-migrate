package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

const (
	pullers = 3
	pushers = 3
)

type Docker struct {
	ctx        context.Context
	cli        *client.Client
	args       *Args
	data       metadataList
	pushch     chan renamedImage
	metadatach chan repositoryMetadata
	done       chan struct{}
	donepushch chan struct{}
}

func newDocker() *Docker {
	return &Docker{
		ctx:        context.Background(),
		done:       make(chan struct{}),
		donepushch: make(chan struct{}),
	}
}

func (d *Docker) mustStartCli() *Docker {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}
	d.cli = cli
	return d
}

func (d *Docker) withArgs(args *Args) *Docker {
	d.args = args
	return d
}

func (d *Docker) addMetadataList(metadataList metadataList) *Docker {
	d.data = metadataList
	return d
}

func (d *Docker) migrate() *Docker {
	auth := d.authorize(d.data.auth)

	authTarget, targetRepositoriesMetadata := d.prepare()

	d.channels()

	for _, metadata := range d.data.repoList {
		d.metadatach <- metadata
	}
	close(d.metadatach)

	for i := 0; i < pullers; i++ {
		go func() {
			d.pullers(auth, targetRepositoriesMetadata)
		}()
	}

	go d.waitPullers()
	for i := 0; i < pushers; i++ {
		go func() {
			d.pushers(authTarget)
		}()
	}

	d.waitPushers()
	return d
}

func (d *Docker) pushers(auth string) {
	slog.Info("pusher", "status", "initializing")
	defer func() {
		d.donepushch <- struct{}{}
		slog.Info("pusher", "status", "terminated")
	}()

	for image := range d.pushch {
		if err := d.push(auth, image); err != nil {
			slog.Error("imagePushing", "image", image.imageName, "error", err)
		}
	}
}

func (d *Docker) pullers(auth string, targetRepositoriesMetadata map[string]repositoryMetadata) {
	slog.Info("puller", "status", "initializing")
	defer func() {
		d.done <- struct{}{}
		slog.Info("puller", "status", "terminated")
	}()

	for metadata := range d.metadatach {
		for _, tag := range metadata.tags {
			from, to := generateECRImageNames(
				targetRepositoriesMetadata,
				metadata.repositoryName,
				metadata.repositoryURI,
				tag,
			)

			docker, err := d.pull(auth, metadata.repositoryURI, tag)
			if err != nil {
				slog.Error("imagePulling", "repositoryName", metadata.repositoryName, "tag", tag, "error", err)
				continue
			}

			if err := docker.rename(from, to); err != nil {
				slog.Error("renaming", "from", from, "to", to, "error", err)
			}
		}
	}
}

func generateECRImageNames(tgRepoMetadata map[string]repositoryMetadata, repositoryName, repositoryURI, tag string) (imageSource, imageTarget string) {
	value, found := tgRepoMetadata[repositoryName]
	if !found {
		return "", ""
	}
	{
		imageSource = fmt.Sprintf("%s:%s", repositoryURI, tag)
		imageTarget = fmt.Sprintf("%s:%s", value.repositoryURI, tag)
	}

	return imageSource, imageTarget
}

func (d *Docker) pull(auth, repositoryName, tag string) (*Docker, error) {
	img := fmt.Sprintf("%s:%s", repositoryName, tag)

	out, err := d.cli.ImagePull(d.ctx, img, image.PullOptions{
		RegistryAuth: auth,
	})
	if err != nil {
		return &Docker{}, err
	}

	defer out.Close()
	io.Copy(io.Discard, out)
	slog.Info("imagePulling", "repositoryName", repositoryName, "tag", tag, "status", "pulled")
	return d, err
}

func (d *Docker) push(auth string, new renamedImage) error {
	out, err := d.cli.ImagePush(d.ctx, new.imageName, image.PushOptions{
		RegistryAuth: auth,
	})
	if err != nil {
		return err
	}

	defer out.Close()
	io.Copy(io.Discard, out)
	slog.Info("imagePushing", "image", new.imageName, "status", "pushed")
	return err
}

type renamedImage struct {
	imageName string
}

func (d *Docker) rename(from, to string) error {
	err := d.cli.ImageTag(d.ctx, from, to)
	if err != nil {
		return err
	}

	slog.Info("renaming", "from", from, "to", to)
	d.pushch <- renamedImage{
		imageName: to,
	}
	return err
}

func (d *Docker) authorize(auth authorization) string {
	authConfig := registry.AuthConfig{
		Username: auth.username,
		Password: auth.password,
	}

	encondedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(encondedJSON)
}

func (d *Docker) createDestinationECRClient() *ECR {
	aws := mustInitConfig(
		withRegion(d.args.toRegion),
		withProfile(d.args.to),
	)

	svc := aws.stablishClientWith(
		ecrService(aws.cfg),
	)

	return newEcr(svc.ecr)
}

func (d *Docker) prepare() (string, map[string]repositoryMetadata) {
	ecrRegistry := d.createDestinationECRClient()
	repositories := ecrRegistry.validate(d.data)
	targetRepositoriesMetadata := ecrRegistry.getRepositoryMetadata(repositories)

	token, err := ecrRegistry.authenticate()
	if err != nil {
		panic(err)
	}

	authTarget := d.authorize(token)
	return authTarget, targetRepositoriesMetadata
}

func (d *Docker) waitPullers() {
	for i := 0; i < pullers; i++ {
		<-d.done
	}
	close(d.pushch)
}

func (d *Docker) channels() {
	d.metadatach = make(chan repositoryMetadata, len(d.data.repoList))
	d.pushch = make(chan renamedImage, d.data.imagesCount)
}

func (d *Docker) waitPushers() {
	for i := 0; i < pushers; i++ {
		<-d.donepushch
	}
}
