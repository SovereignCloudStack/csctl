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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	"github.com/SovereignCloudStack/csctl/pkg/github"
	"github.com/SovereignCloudStack/csctl/pkg/github/client"
	"github.com/SovereignCloudStack/csctl/pkg/hash"
	"github.com/SovereignCloudStack/csctl/pkg/providerplugin"
	"github.com/SovereignCloudStack/csctl/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	example = `csctl create tests/cluster-stacks/docker/ferrol -m hash (for hash mode)

	csctl create tests/cluster-stacks/docker/ferrol -m hash --github-release github-release/ (for stable mode)`
)

var (
	mode              string
	outputDirectory   string
	nodeImageRegistry string
)

// CreateOptions contains config for creating a release.
type CreateOptions struct {
	newClusterStackConvention bool
	ClusterStackPath          string
	ClusterStackReleaseDir    string
	Config                    *clusterstack.CsctlConfig
	Metadata                  *clusterstack.MetaData
	CurrentReleaseHash        hash.ReleaseHash
	LatestReleaseHash         hash.ReleaseHash
	NodeImageRegistry         string
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
	createCmd.Flags().StringVarP(&outputDirectory, "output", "o", "./.release", "It defines the output directory in which the release artifacts will be generated")
	createCmd.Flags().StringVarP(&nodeImageRegistry, "node-image-registry", "r", "", "It defines the node image registry. For example oci://ghcr.io/foo/bar/node-images/staging/")
}

// GetCreateOptions create a Create Option for create command.
func GetCreateOptions(ctx context.Context, clusterStackPath string) (*CreateOptions, error) {
	createOption := &CreateOptions{}

	// ClusterAddon config
	config, err := clusterstack.GetCsctlConfig(clusterStackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	createOption.ClusterStackPath = clusterStackPath
	createOption.Config = config

	if _, err := os.Stat(filepath.Join(clusterStackPath, "clusteraddon.yaml")); err != nil {
		// old if clusteraddon.yaml is not present.
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to find clusteraddon.yaml: %w", err)
		}
	} else {
		// new if clusteraddon.yaml is present.
		createOption.newClusterStackConvention = true
	}

	_, _, err = providerplugin.GetProviderExecutable(config)
	if err != nil {
		return createOption, fmt.Errorf("providerplugin.GetProviderExecutable(&config) failed: %w", err)
	}

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
		gc, err := client.NewFactory().NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create new github client: %w", err)
		}

		// update the metadata kubernetes version with the csctl.yaml config
		createOption.Metadata.Versions.Kubernetes = config.Config.KubernetesVersion

		latestRepoRelease, err := github.GetLatestReleaseFromRemoteRepository(ctx, mode, config, gc)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest release form remote repository: %w", err)
		}
		fmt.Printf("latest release found: %q\n", latestRepoRelease)

		if latestRepoRelease == "" {
			createOption.Metadata.APIVersion = "metadata.clusterstack.x-k8s.io/v1alpha1"
			createOption.Metadata.Versions.Kubernetes = config.Config.KubernetesVersion
			createOption.Metadata.Versions.ClusterStack = "v1"
			createOption.Metadata.Versions.Components.ClusterAddon = "v1"
		} else {
			if err := github.DownloadReleaseAssets(ctx, latestRepoRelease, "./.tmp/release/", gc); err != nil {
				return nil, fmt.Errorf("failed to download release asset: %w", err)
			}

			createOption.Metadata, err = clusterstack.HandleStableMode("./.tmp/release/", createOption.CurrentReleaseHash, createOption.LatestReleaseHash)
			if err != nil {
				return nil, fmt.Errorf("failed to handle stable mode: %w", err)
			}
		}
	}

	releaseDirName, err := clusterstack.GetClusterStackReleaseDirectoryName(createOption.Metadata, createOption.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stack release directory name: %w", err)
	}
	// Release directory name `release/docker-ferrol-1-27-v1`
	createOption.ClusterStackReleaseDir = filepath.Join(outputDirectory, releaseDirName)

	createOption.NodeImageRegistry = nodeImageRegistry

	return createOption, nil
}

func createAction(cmd *cobra.Command, args []string) error {
	defer cleanTmpDirectory()

	if len(args) != 1 {
		return fmt.Errorf("please provide a valid command, create only accept one argument to path to the cluster stacks")
	}
	clusterStackPath := args[0]

	if mode != stableMode && mode != hashMode {
		fmt.Println("The mode is ", mode)
		return fmt.Errorf("mode is not supported please choose from - stable, hash")
	}

	createOpts, err := GetCreateOptions(cmd.Context(), clusterStackPath)
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
	fmt.Printf("Creating output in %s\n", c.ClusterStackReleaseDir)
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
	if err := template.GenerateOutputFromTemplate(c.ClusterStackPath, "./.tmp/", c.Metadata); err != nil {
		return fmt.Errorf("failed to generate tmp output: %w", err)
	}

	// Overwrite ClusterAddonVersion in cluster-addon/*/Chart.yaml
	if err := overwriteClusterAddonVersion("./.tmp", c.Metadata.Versions.Components.ClusterAddon); err != nil {
		return fmt.Errorf("failed to overwrite ClusterAddonVersion in tmp output: %w", err)
	}

	// Overwrite ClusterClassVersion in cluster-class/Chart.yaml
	clusterClassChartYaml := "./.tmp/cluster-class/Chart.yaml"
	if err := overwriteVersionInFile(clusterClassChartYaml, c.Metadata.Versions.ClusterStack); err != nil {
		return fmt.Errorf("failed to overwrite ClusterClassVersion in %s output: %w", clusterClassChartYaml, err)
	}

	// Package Helm from the tmp directory to the release directory
	if err := template.CreatePackage("./.tmp/", c.ClusterStackReleaseDir, c.newClusterStackConvention, c.Config, c.Metadata); err != nil {
		return fmt.Errorf("failed to create template package: %w", err)
	}

	if c.newClusterStackConvention {
		// Copy the clusteraddon.yaml config to release if new way
		clusterAddonData, err := os.ReadFile(filepath.Join(c.ClusterStackPath, "clusteraddon.yaml"))
		if err != nil {
			return fmt.Errorf("failed to read clusteraddon.yaml: %w", err)
		}

		if err := os.WriteFile(filepath.Join(c.ClusterStackReleaseDir, "clusteraddon.yaml"), clusterAddonData, os.FileMode(0o644)); err != nil {
			return fmt.Errorf("failed to write clusteraddon.yaml: %w", err)
		}
	} else {
		// Copy the cluster-addon-values.yaml config to release if old way
		clusterAddonData, err := os.ReadFile(filepath.Join(c.ClusterStackPath, "cluster-addon-values.yaml"))
		if err != nil {
			return fmt.Errorf("failed to read cluster-addon-values.yaml: %w", err)
		}

		if err := os.WriteFile(filepath.Join(c.ClusterStackReleaseDir, "cluster-addon-values.yaml"), clusterAddonData, os.FileMode(0o644)); err != nil {
			return fmt.Errorf("failed to write cluster-addon-values.yaml: %w", err)
		}
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

	err = providerplugin.CreateNodeImages(c.Config,
		c.ClusterStackPath,
		c.ClusterStackReleaseDir,
		c.NodeImageRegistry)
	if err != nil {
		return fmt.Errorf("providerplugin.CreateNodeImages() failed: %w", err)
	}
	return nil
}

func overwriteClusterAddonVersion(tmpDir, clusterAddonVersion string) error {
	g := filepath.Join(tmpDir, "cluster-addon", "Chart.yaml")
	files, err := filepath.Glob(g)
	if err != nil {
		return fmt.Errorf("glob for %s failed: %w", g, err)
	}

	for _, chartYaml := range files {
		err := overwriteVersionInFile(chartYaml, clusterAddonVersion)
		if err != nil {
			return fmt.Errorf("failed to replace version in %s: %w", chartYaml, err)
		}
	}
	return nil
}

// overwriteVersionInFile replaces "version: v123" with newVersion.
func overwriteVersionInFile(chartYaml, newVersion string) error {
	chartYaml = filepath.Clean(chartYaml)
	data, err := os.ReadFile(chartYaml)
	if err != nil {
		return fmt.Errorf("reading file failed: %w", err)
	}

	m := make(map[string]interface{})
	err = yaml.Unmarshal(data, m)
	if err != nil {
		return fmt.Errorf("failed parsing: %w", err)
	}

	v := m["version"]
	oldVersion, ok := v.(string)
	if !ok {
		return fmt.Errorf("failed to read version in yaml")
	}

	m["version"] = newVersion
	fmt.Printf("%s updating version from %s to %s\n", chartYaml, oldVersion, newVersion)
	out, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed creating yaml: %w", err)
	}
	err = os.WriteFile(chartYaml, out, 0o600)
	if err != nil {
		return fmt.Errorf("failed write yaml to file: %w", err)
	}
	return nil
}

func cleanTmpDirectory() error {
	if err := os.RemoveAll("./.tmp/"); err != nil {
		return fmt.Errorf("failed to remove tmp directory: %w", err)
	}

	return nil
}
