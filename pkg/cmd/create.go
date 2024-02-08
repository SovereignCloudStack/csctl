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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SovereignCloudStack/csmctl/pkg/clusterstack"
	"github.com/SovereignCloudStack/csmctl/pkg/github"
	"github.com/SovereignCloudStack/csmctl/pkg/hash"
	"github.com/SovereignCloudStack/csmctl/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	stableMode = "stable"
	hashMode   = "hash"
)

var (
	shortDescription = "Creates a cluster stack release with the help of given cluster stack"
	longDescription  = `It takes cluster stacks and mode as an input and based on that creates
	the cluster stack release in the current directory named "release/".
	Supported modes are - stable, alpha, beta, hash

	note - Hash mode takes the last hash of the git commit.`
	example = `csmctl create tests/cluster-stacks/docker/ferrol -m hash (for hash mode)

	csmctl create tests/cluster-stacks/docker/ferrol -m hash --github-release github-release/ (for stable mode)`
)

var (
	mode            string
	outputDirectory string
	// TODO: remove this later.
	githubReleasePath string
)

// CreateOptions contains config for creating a release.
type CreateOptions struct {
	ClusterStackPath       string
	ClusterStackReleaseDir string
	Config                 clusterstack.CsmctlConfig
	Metadata               clusterstack.MetaData
	CurrentReleaseHash     hash.ReleaseHash
	LatestReleaseHash      hash.ReleaseHash
}

// createCmd represents the create command.
var createCmd = &cobra.Command{
	Use:          "create",
	Short:        shortDescription,
	Long:         longDescription,
	Example:      example,
	RunE:         createAction,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().StringVarP(&mode, "mode", "m", "stable", "It defines the mode of the cluster stack manager")
	createCmd.Flags().StringVarP(&outputDirectory, "output", "o", "./releases", "It defines the output directory in which the release artifacts will be generated")
	// TODO: remove this later
	createCmd.Flags().StringVar(&githubReleasePath, "github-release", "github-release", "It is used to get the path to local github release (for stable mode only)")
}

// GetCreateOptions create a Create Option for create command.
func GetCreateOptions(clusterStackPath string) (*CreateOptions, error) {
	createOption := &CreateOptions{}

	// ClusterAddon config
	config, err := clusterstack.GetCsmctlConfig(clusterStackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	createOption.ClusterStackPath = clusterStackPath
	createOption.Config = config

	currentHash, err := hash.GetHash(clusterStackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash: %w", err)
	}
	createOption.CurrentReleaseHash = currentHash

	switch mode {
	case hashMode:
		createOption.Metadata, err = clusterstack.HandleHashMode(config.Config.KubernetesVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to handle hash mode: %w", err)
		}
	case stableMode:
		createOption.Metadata, err = clusterstack.HandleStableMode(githubReleasePath, createOption.CurrentReleaseHash, createOption.LatestReleaseHash)
		if err != nil {
			return nil, fmt.Errorf("failed to handle stable mode: %w", err)
		}

		// TODO: remove
		latestReleaseInfoWithHash, err := github.GetLocalReleaseInfoWithHash(githubReleasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest github local release: %w", err)
		}
		createOption.LatestReleaseHash = latestReleaseInfoWithHash.Hash
	}

	releaseDirName, err := clusterstack.GetClusterStackReleaseDirectoryName(&createOption.Metadata, &createOption.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stack release directory name: %w", err)
	}
	// Release directory name `release/docker-ferrol-1-27-v1`
	createOption.ClusterStackReleaseDir = filepath.Join(outputDirectory, releaseDirName)

	return createOption, nil
}

func createAction(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please provide a valid command, create only accept one argument to path to the cluster stacks")
	}
	clusterStackPath := args[0]

	if mode != stableMode && mode != hashMode {
		fmt.Println("The mode is ", mode)
		return fmt.Errorf("mode is not supported please choose from - stable, hash")
	}

	createOpts, err := GetCreateOptions(clusterStackPath)
	if err != nil {
		return fmt.Errorf("failed to create create options: %w", err)
	}

	// Validate if there any change or not
	if err := createOpts.validateHash(); err != nil {
		return fmt.Errorf("failed to validate with latest release hash: %w", err)
	}

	if err := createOpts.generateRelease(); err != nil {
		return fmt.Errorf("failed to generate release: %w", err)
	}
	fmt.Printf("Created %s\n", createOpts.ClusterStackReleaseDir)

	return nil
}

// validateHash returns if some hash changes or not.
func (c *CreateOptions) validateHash() error {
	if c.CurrentReleaseHash.ClusterAddonDir == c.LatestReleaseHash.ClusterAddonDir &&
		c.CurrentReleaseHash.ClusterAddonValues == c.LatestReleaseHash.ClusterAddonValues &&
		c.CurrentReleaseHash.NodeImageDir == c.LatestReleaseHash.NodeImageDir {
		return fmt.Errorf("no change in the cluster stack")
	}

	return nil
}

func (c *CreateOptions) generateRelease() error {
	if err := os.MkdirAll(c.ClusterStackReleaseDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write the current hash
	hashJSONData, err := json.MarshalIndent(c.CurrentReleaseHash, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hash json: %w", err)
	}

	filePath := filepath.Join(c.ClusterStackReleaseDir, "hashes.json")
	hashFile, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to create hash json: %w", err)
	}
	defer hashFile.Close()

	_, err = hashFile.Write(hashJSONData)
	if err != nil {
		return fmt.Errorf("failed to write current release hash: %w", err)
	}

	// Build all the templated output and put it in a tmp directory
	if err := template.GenerateOutputFromTemplate(c.ClusterStackPath, "./tmp/", &c.Metadata); err != nil {
		return fmt.Errorf("failed to generate tmp output: %w", err)
	}

	// Package Helm from the tmp directory to the release directory
	if err := template.CreatePackage("./tmp/", c.ClusterStackReleaseDir); err != nil {
		return fmt.Errorf("failed to create template package: %w", err)
	}

	// Copy the cluster-addon-values.yaml config to release if old way
	clusterAddonData, err := os.ReadFile(filepath.Join(c.ClusterStackPath, "cluster-addon-values.yaml"))
	if err != nil {
		return fmt.Errorf("failed to read cluster-addon-values.yaml: %w", err)
	}

	if err := os.WriteFile(filepath.Join(c.ClusterStackReleaseDir, "cluster-addon-values.yaml"), clusterAddonData, os.ModePerm); err != nil {
		return fmt.Errorf("failed to write cluster-addon-values.yaml: %w", err)
	}

	// Put the final metadata file into the output directory.
	metaDataByte, err := yaml.Marshal(c.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata yaml: %w", err)
	}

	metadataFile, err := os.Create(filepath.Join(c.ClusterStackReleaseDir, "metadata.yaml"))
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metadataFile.Close()

	if _, err := metadataFile.Write(metaDataByte); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}
