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
