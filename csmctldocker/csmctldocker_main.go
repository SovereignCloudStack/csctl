/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main provides a dummy plugin for csmctl. You can use that code
// to create a real csmctl plugin.
// You can implement the "create-node-images" command to create node images during
// a `csmclt create` call.
package main

import (
	"fmt"
	"os"

	csmctlclusterstack "github.com/SovereignCloudStack/csmctl/pkg/clusterstack"
)

const provider = "docker"

func usage() {
	fmt.Printf(`%s create-node-images cluster-stack-directory cluster-stack-release-directory
This command is a csmctl plugin.

https://github.com/SovereignCloudStack/csmctl
`, os.Args[0])
}

func main() {
	if len(os.Args) != 4 {
		usage()
		os.Exit(1)
	}
	if os.Args[1] != "create-node-images" {
		usage()
		os.Exit(1)
	}
	clusterStackPath := os.Args[2]
	config, err := csmctlclusterstack.GetCsmctlConfig(clusterStackPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if config.Config.Provider.Type != provider {
		fmt.Printf("Wrong provider in %s. Expected %s\n", clusterStackPath, provider)
		os.Exit(1)
	}
	releaseDir := os.Args[3]
	_, err = os.Stat(releaseDir)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Printf("clusterStackPath: %s\n", clusterStackPath)
	fmt.Printf("releaseDir: %s\n", releaseDir)
	fmt.Println(".... pretending to do heavy work (creating node images) ...")
}
