package main

import "flag"

type Args struct {
	fromRegion string
	toRegion   string
	from       string
	to         string
	file       string
}

func NewArgsGetter() *Args {
	file := flag.String("config_file", "list.yaml", "file with list of repositories")
	fromRegion := flag.String("from_region", "us-east-1", "default ecr client region")
	toRegion := flag.String("to_region", "us-east-1", "target ecr client region")
	from := flag.String("from", "default", "default ecr origin profile")
	to := flag.String("to", "HOME-LAB", "default ecr destination profile")
	flag.Parse()
	return &Args{
		file:       *file,
		fromRegion: *fromRegion,
		toRegion:   *toRegion,
		from:       *from,
		to:         *to,
	}
}
