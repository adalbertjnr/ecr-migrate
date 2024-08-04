package main

import (
	"log/slog"
	"os"
)

func main() {
	args := NewArgsGetter()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	repoFinder := newRepositoryFinder()
	repositories := repoFinder.locateIn(args.file).registryList()

	aws := mustInitConfig(
		withRegion(args.fromRegion),
		withProfile(args.from),
	)

	svc := aws.stablishClientWith(
		ecrService(aws.cfg),
	)

	ecrRegistry := newEcr(svc.ecr)
	imageMetadataList := ecrRegistry.walk(repositories.List)

	docker := newDocker().mustStartCli()
	docker.addMetadataList(imageMetadataList).withArgs(args).migrate()
}
