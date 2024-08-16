package main

import "flag"

type Args struct {
	pullers     int
	pushers     int
	fromRegion  string
	toRegion    string
	fromProfile string
	toProfile   string
	file        string
}

func NewArgsGetter() *Args {

	var (
		file        = flag.String("config_file", "list.yaml", "file with list of repositories")
		fromRegion  = flag.String("from_region", "us-east-1", "default ecr client region")
		toRegion    = flag.String("to_region", "us-east-1", "target ecr client region")
		fromProfile = flag.String("from", "default", "default ecr origin profile")
		toProfile   = flag.String("to", "HOME-LAB", "default ecr destination profile")
		pullers     = flag.Int("pullers", 3, "set the amount of workers for pull images concurrently")
		pushers     = flag.Int("pushers", 3, "set the amount of workers for push images concurrently")
	)

	flag.Parse()
	return &Args{
		file:        *file,
		fromRegion:  *fromRegion,
		toRegion:    *toRegion,
		fromProfile: *fromProfile,
		toProfile:   *toProfile,
		pullers:     *pullers,
		pushers:     *pushers,
	}
}
