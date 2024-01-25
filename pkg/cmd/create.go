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

	"github.com/SovereignCloudStack/cluster-stack-operator/pkg/version"
	csmctlclusterstack "github.com/SovereignCloudStack/csmctl/pkg/clusterstack"
	"github.com/SovereignCloudStack/csmctl/pkg/git"
	"github.com/SovereignCloudStack/csmctl/pkg/github"
	"github.com/SovereignCloudStack/csmctl/pkg/hash"
	"github.com/SovereignCloudStack/csmctl/pkg/providerplugin"
	"github.com/SovereignCloudStack/csmctl/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	stableMode = "stable"
	alphaMode  = "alpha"
	betaMode   = "beta"
	hashMode   = "hash"
)

var (
	mode            string
	outputDirectory string
	// TODO: remove this later.
	githubReleasePath string
)

// CreateOptions contains config for creating a release.
type CreateOptions struct {
	config                 *csmctlclusterstack.CsmctlConfig
	metadata               *csmctlclusterstack.MetaData
	clusterStackPath       string
	clusterStackReleaseDir string
	currentHash            hash.ReleaseHash
}

// createCmd represents the create command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a cluster stack release with the help of given cluster stack",
	Long: `It takes cluster stacks and mode as an input and based on that creates
the cluster stack release in the current directory named "release/".
Supported modes are - stable, alpha, beta, hash

note - Hash mode takes the last hash of the git commit.`,
	RunE:         createAction,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().StringVarP(&mode, "mode", "m", "stable", "It defines the mode of the cluster stack manager")
	createCmd.Flags().StringVarP(&outputDirectory, "output", "o", "./releases", "It defines the output directory in which the release artifacts will be generated")
	// TODO: remove this later
	createCmd.Flags().StringVar(&githubReleasePath, "github-release", "github-release", "It is used to get the path to local github release (for stable mode only)")
}

func createAction(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please provide a valid command, create only accept one argument to path to the cluster stacks")
	}
	clusterStackPath := args[0]

	// Get csmctl config form cluster stacks.
	config, err := csmctlclusterstack.GetCsmctlConfig(clusterStackPath)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	_, _, err = providerplugin.GetProviderExecutable(config)
	if err != nil {
		return err
	}

	// Skip downloading github release for the hash mode.
	// var latestRepoRelease *string
	if mode == hashMode {
		// Handle metadata based on mode and create the -
		// release directory and creates the metadata.yaml file in the release directory.
		metadata, csrDirName, err := handleMetadataAndGetCSRDirectoryName(mode, githubReleasePath, config, hash.ReleaseHash{}, hash.ReleaseHash{})
		if err != nil {
			return err
		}

		// Calculate the current Hash, and check if anything changed in the cluster stacks.
		currentHash, err := hash.GetHash(clusterStackPath)
		if err != nil {
			return fmt.Errorf("failed to get hash: %w", err)
		}

		create := &CreateOptions{
			config:                 config,
			metadata:               metadata,
			clusterStackPath:       clusterStackPath,
			clusterStackReleaseDir: csrDirName,
			currentHash:            currentHash,
		}
		if err := create.buildNodeImagesAndGenerateRelease(); err != nil {
			return fmt.Errorf("failed to build packer and generate release: %w", err)
		}
		return nil
	}

	// TODO: delete it later once we have github release available
	// Get Release with Hash information
	latestReleaseInfoWithHash, err := github.GetLocalReleaseInfoWithHash(githubReleasePath)
	if err != nil {
		return fmt.Errorf("failed to get local release info with hash: %w", err)
	}

	// check if release directory exists or not, if not create it.
	if err := os.MkdirAll(outputDirectory, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create release directory: %w", err)
	}

	// Calculate the current Hash, and check if anything changed in the cluster stacks.
	currentHash, err := hash.GetHash(clusterStackPath)
	if err != nil {
		return fmt.Errorf("failed to get hash: %w", err)
	}
	if err := currentHash.ValidateWithLatestReleaseHash(latestReleaseInfoWithHash.Hash); err != nil {
		return fmt.Errorf("failed to validate with latest release hash: %w", err)
	}

	// Handle metadata based on mode and create the -
	// release directory and creates the metadata.yaml file in the release directory.
	metadata, csrDirName, err := handleMetadataAndGetCSRDirectoryName(mode, githubReleasePath, config, latestReleaseInfoWithHash.Hash, currentHash)
	if err != nil {
		return err
	}

	create := &CreateOptions{
		config:                 config,
		metadata:               metadata,
		clusterStackPath:       clusterStackPath,
		clusterStackReleaseDir: csrDirName,
		currentHash:            currentHash,
	}
	if err := create.buildNodeImagesAndGenerateRelease(); err != nil {
		return fmt.Errorf("failed to build packer and generate release: %w", err)
	}

	return nil
}

func handleMetadataAndGetCSRDirectoryName(mode, githubReleasePath string, config *csmctlclusterstack.CsmctlConfig, latestReleaseHash, currentHash hash.ReleaseHash) (metadata *csmctlclusterstack.MetaData, csrDirName string, err error) {
	metadata = &csmctlclusterstack.MetaData{}

	if mode != stableMode && mode != alphaMode && mode != betaMode && mode != hashMode {
		fmt.Println("The mode is ", mode)
		return nil, "", fmt.Errorf("mode is not supported please choose from - stable, alpha, beta, hash")
	}

	metadata.APIVersion = "metadata.clusterstack.x-k8s.io/v1alpha1"
	metadata.Versions.Kubernetes = config.Config.KubernetesVersion

	switch mode {
	case stableMode:
		metadata, err = handleStableMode(githubReleasePath, latestReleaseHash, currentHash)
		if err != nil {
			return nil, "", fmt.Errorf("failed to handle stable mode: %w", err)
		}
	case hashMode:
		metadata, err = handleHashMode(metadata)
		if err != nil {
			return nil, "", fmt.Errorf("failed to handle hash mode: %w", err)
		}
	}

	// Parse the cluster stack version from dot format `v1-alpha.0` to a version way of struct
	// and parse the kubernetes version from `v1.27.3` to a major minor way
	// and create the release directory at the end.
	clusterStackVersion, err := version.New(metadata.Versions.ClusterStack)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse cluster stack version: %w", err)
	}
	kubernetesVerion, err := config.ParseKubernetesVersion()
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse kubernetes version: %w", err)
	}
	clusterStackReleaseDirName := fmt.Sprintf("%s-%s-%s-%s", config.Config.Provider.Type, config.Config.ClusterStackName, kubernetesVerion.String(), clusterStackVersion.String())

	if err := os.MkdirAll(filepath.Join(outputDirectory, clusterStackReleaseDirName), os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create release directory: %w", err)
	}

	// Put the final metadata file into the output directory.
	metaDataByte, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal metadata file: %w", err)
	}

	metadataFile, err := os.Create(filepath.Clean(filepath.Join(outputDirectory, clusterStackReleaseDirName, "metadata.yaml")))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metadataFile.Close()

	if _, err := metadataFile.Write(metaDataByte); err != nil {
		return nil, "", fmt.Errorf("failed to write metadata file: %w", err)
	}

	return metadata, clusterStackReleaseDirName, nil
}

func handleStableMode(githubReleasePath string, latestReleaseHash, currentHash hash.ReleaseHash) (*csmctlclusterstack.MetaData, error) {
	metadata, err := csmctlclusterstack.ParseMetaData(githubReleasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	metadata.Versions.Components.NodeImage = ""

	metadata.Versions.ClusterStack, err = csmctlclusterstack.BumpVersion(metadata.Versions.ClusterStack)
	if err != nil {
		return nil, fmt.Errorf("failed to bump cluster stack version: %w", err)
	}
	fmt.Printf("Bumped ClusterStack Version: %s\n", metadata.Versions.ClusterStack)

	if currentHash.ClusterAddonDir != latestReleaseHash.ClusterAddonDir || currentHash.ClusterAddonValues != latestReleaseHash.ClusterAddonValues {
		metadata.Versions.Components.ClusterAddon, err = csmctlclusterstack.BumpVersion(metadata.Versions.Components.ClusterAddon)
		if err != nil {
			return nil, fmt.Errorf("failed to bump version: %w", err)
		}
		fmt.Printf("Bumped ClusterAddon Version: %s\n", metadata.Versions.Components.ClusterAddon)
	} else {
		fmt.Printf("ClusterAddon Version unchanged: %s\n", metadata.Versions.Components.ClusterAddon)
	}

	return metadata, nil
}

func handleHashMode(metadata *csmctlclusterstack.MetaData) (*csmctlclusterstack.MetaData, error) {
	gitCommitHash, err := git.GetLatestGitCommit("./")
	if err != nil {
		return nil, fmt.Errorf("failed to get latest git commit: %w", err)
	}
	gitCommitHash = fmt.Sprintf("v0-sha.%s", gitCommitHash)
	metadata.Versions.ClusterStack = gitCommitHash
	metadata.Versions.Components.ClusterAddon = gitCommitHash

	return metadata, nil
}

func (c *CreateOptions) buildNodeImagesAndGenerateRelease() error {
	// Release directory name
	releaseDirectory := filepath.Join(outputDirectory, c.clusterStackReleaseDir)
	fmt.Printf("Creating output in %s\n", releaseDirectory)
	// Write the current hash
	hashJSONData, err := json.MarshalIndent(c.currentHash, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	filePath := filepath.Join(releaseDirectory, "hashes.json")
	hashFile, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to create hash file: %w", err)
	}
	defer hashFile.Close()

	_, err = hashFile.Write(hashJSONData)
	if err != nil {
		return fmt.Errorf("failed to write hash file: %w", err)
	}

	// Build all the templated output and put it in a tmp directory
	if err := template.GenerateOutputFromTemplate(c.clusterStackPath, "./tmp/", c.metadata); err != nil {
		return fmt.Errorf("failed to generate new temporary output template: %w", err)
	}

	// Package Helm from the tmp directory to the release directory
	if err := template.CreatePackage("./tmp/", releaseDirectory); err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	return providerplugin.CreateNodeImages(c.config, c.clusterStackPath, releaseDirectory)
}
