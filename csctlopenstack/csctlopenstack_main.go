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

// Package main provides a dummy plugin for csctl. You can use that code
// to create a real csctl plugin.
// You can implement the "create-node-images" command to create node images during
// a `csclt create` call.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	csctlclusterstack "github.com/SovereignCloudStack/csctl/pkg/clusterstack"
)

const provider = "openstack"

func usage() {
	fmt.Printf(`%s create-node-images cluster-stack-directory cluster-stack-release-directory
This command is a csctl plugin.
https://github.com/SovereignCloudStack/csctl
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
	config, err := csctlclusterstack.GetCsctlConfig(clusterStackPath)
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
	method := config.Config.Provider.Config.Method
	switch strings.ToLower(method) {
	case "get":
		// #nosec G304
		nodeImageData, err := os.ReadFile(filepath.Join(clusterStackPath, "node-images", "config.yaml"))
		if err != nil {
			fmt.Printf("failed to read config.yaml: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(filepath.Join(releaseDir, "node-images.yaml"), nodeImageData, os.FileMode(0o644)); err != nil {
			fmt.Printf("failed to write config.yaml: %v\n", err)
			os.Exit(1)
		}
	case "build":
		if len(config.Config.Provider.Config.Images) > 0 {
			for _, image := range config.Config.Provider.Config.Images {
				// Construct the path to the image folder
				packerImagePath := filepath.Join(clusterStackPath, "node-images", *image)

				if _, err := os.Stat(packerImagePath); err == nil {
					fmt.Println("Running packer build...")
					cmd := exec.Command("packer", "build", packerImagePath)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						fmt.Printf("Error running packer build: %v\n", err)
						os.Exit(1)
					}
					fmt.Println("Packer build completed successfully.")

					// // Push the built image to S3
					// if err := pushToS3(releaseDir, *image); err != nil {
					// 	fmt.Printf("Error pushing image to S3: %v\n", err)
					// 	os.Exit(1)
					// }
				} else {
					fmt.Printf("Image folder %s does not exist\n", packerImagePath)
				}
			}
		} else {
			fmt.Println("No images to build")
		}
	default:
		fmt.Println("Unknown method:", method)
	}
}
