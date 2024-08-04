package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type CloudConfig struct {
	cfg     aws.Config
	region  string
	profile string
}

type Option func(*CloudConfig)

func withRegion(region string) Option {
	return func(cc *CloudConfig) {
		cc.region = region
	}
}

func withProfile(profile string) Option {
	return func(cc *CloudConfig) {
		cc.profile = profile
	}
}

func mustInitConfig(opts ...Option) *CloudConfig {
	defaultOpts := &CloudConfig{
		cfg:     aws.Config{},
		region:  "",
		profile: "",
	}

	for _, opt := range opts {
		opt(defaultOpts)
	}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithSharedConfigProfile(defaultOpts.profile),
		config.WithRegion(defaultOpts.region),
	)
	if err != nil {
		panic(err)
	}

	defaultOpts.cfg = cfg
	return defaultOpts
}

func (c *CloudConfig) stablishClientWith(opts ...ResourceOpt) *ResoureceConfig {
	o := &ResoureceConfig{}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

type ResoureceConfig struct {
	ecr *ecr.Client
	sts *sts.Client
}

type ResourceOpt func(*ResoureceConfig)

func ecrService(cfg aws.Config) ResourceOpt {
	return func(rc *ResoureceConfig) {
		e := ecr.NewFromConfig(cfg)
		rc.ecr = e
	}
}

func stsService(cfg aws.Config) ResourceOpt {
	return func(rc *ResoureceConfig) {
		s := sts.NewFromConfig(cfg)
		rc.sts = s
	}
}
