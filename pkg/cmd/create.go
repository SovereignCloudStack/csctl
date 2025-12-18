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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SovereignCloudStack/csctl/pkg/assetsclient"
	"github.com/SovereignCloudStack/csctl/pkg/assetsclient/github"
	"github.com/SovereignCloudStack/csctl/pkg/assetsclient/oci"
	"github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	"github.com/SovereignCloudStack/csctl/pkg/cshash"
	"github.com/SovereignCloudStack/csctl/pkg/providerplugin"
	"github.com/SovereignCloudStack/csctl/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	stableMode = "stable"
	hashMode   = "hash"
	customMode = "custom"
)

var (
	shortDescription = "Creates a cluster stack release with the help of given cluster stack"
	longDescription  = `It takes cluster stacks and mode as an input and based on that creates
	the cluster stack release in the current directory named "release/".
	Supported modes are - stable, alpha, beta, hash

	note - Hash mode takes the last hash of the git commit.`
	example = `csctl create tests/cluster-stacks/docker/ferrol -m hash (for hash mode)

csctl create tests/cluster-stacks/docker/ferrol -m hash github-release/ (for stable mode)

csctl create --publish --remote oci tests/cluster-stacks/docker/ferrol (publish to OCI repository)`
)

var (
	mode                string
	outputDirectory     string
	nodeImageRegistry   string
	clusterStackVersion string
	clusterAddonVersion string
	nodeImageVersion    string
	remote              string
	publish             bool
)

// CreateOptions contains config for creating a release.
type CreateOptions struct {
	newClusterStackConvention bool
	ClusterStackPath          string
	ClusterStackReleaseDir    string
	Config                    *clusterstack.CsctlConfig
	Metadata                  *clusterstack.MetaData
	CurrentReleaseHash        cshash.ReleaseHash
	LatestReleaseHash         cshash.ReleaseHash
	NodeImageRegistry         string
	releaseName               string
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
	createCmd.Flags().StringVar(&clusterStackVersion, "cluster-stack-version", "", "It is used to specify the semver version for the cluster stack in the custom mode")
	createCmd.Flags().StringVar(&clusterAddonVersion, "cluster-addon-version", "", "It is used to specify the semver version for the cluster addon in the custom mode")
	createCmd.Flags().StringVar(&nodeImageVersion, "node-image-version", "", "It is used to specify the semver version for the node images in the custom mode")
	createCmd.Flags().StringVar(&remote, "remote", "github", "Which remote repository to use and thus which credentials are required. Currently supported are 'github' and 'oci'.")
	createCmd.Flags().BoolVar(&publish, "publish", false, "Publish release after creation is done. This is only implemented for OCI currently.")
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

	currentHash, err := cshash.GetHash(clusterStackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash: %w", err)
	}
	createOption.CurrentReleaseHash = currentHash

	switch mode {
	case hashMode:
		createOption.Metadata = clusterstack.HandleHashMode(createOption.CurrentReleaseHash, config.Config.KubernetesVersion)
	case stableMode:
		createOption.Metadata = &clusterstack.MetaData{}

		var remoteFactory assetsclient.Factory

		// using switch here in case more will be added in the future (aws?)
		switch remote {
		case "github":
			remoteFactory = github.NewFactory()
		case "oci":
			remoteFactory = oci.NewFactory()
		}

		ac, err := remoteFactory.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create new asset client: %w", err)
		}

		latestRepoRelease, err := getLatestReleaseFromRemoteRepository(ctx, mode, config, ac)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest release form remote repository: %w", err)
		}
		fmt.Printf("latest release found: %q\n", latestRepoRelease)

		if latestRepoRelease == "" {
			createOption.Metadata.APIVersion = "metadata.clusterstack.x-k8s.io/v1alpha1"
			createOption.Metadata.Versions.Kubernetes = config.Config.KubernetesVersion
			createOption.Metadata.Versions.ClusterStack = "v1"
			createOption.Metadata.Versions.Components.ClusterAddon = "v1"
			createOption.Metadata.Versions.Components.NodeImage = "v1"
		} else {
			if err := downloadReleaseAssets(ctx, latestRepoRelease, "./.tmp/release/", ac); err != nil {
				return nil, fmt.Errorf("failed to download release asset: %w", err)
			}

			createOption.LatestReleaseHash, err = cshash.ParseReleaseHash("./.tmp/release/hashes.json")
			if err != nil {
				return nil, fmt.Errorf("failed to read hash from the github: %w", err)
			}

			createOption.Metadata, err = clusterstack.HandleStableMode("./.tmp/release/", createOption.CurrentReleaseHash, createOption.LatestReleaseHash)
			if err != nil {
				return nil, fmt.Errorf("failed to handle stable mode: %w", err)
			}

			// update the metadata kubernetes version with the csctl.yaml config
			createOption.Metadata.Versions.Kubernetes = config.Config.KubernetesVersion
		}
	case customMode:
		if clusterStackVersion == "" {
			return nil, errors.New("please specify a semver for custom version with --cluster-stack-version flag")
		}
		if clusterAddonVersion == "" {
			return nil, errors.New("please specify a semver for custom version with --cluster-addon-version flag")
		}
		if nodeImageVersion == "" {
			return nil, errors.New("please specify a semver for custom version with --node-image-version flag")
		}

		createOption.Metadata, err = clusterstack.HandleCustomMode(createOption.Config.Config.KubernetesVersion, clusterStackVersion, clusterAddonVersion, nodeImageVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to handle custom mode: %w", err)
		}
	}

	releaseDirName, err := clusterstack.GetClusterStackReleaseDirectoryName(createOption.Metadata, createOption.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stack release directory name: %w", err)
	}

	createOption.releaseName, err = clusterstack.GetClusterStackReleaseDirectoryName(createOption.Metadata, createOption.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stack release name: %w", err)
	}

	// Release directory name `release/docker-ferrol-1-27-v1`
	createOption.ClusterStackReleaseDir = filepath.Join(outputDirectory, releaseDirName)

	createOption.NodeImageRegistry = nodeImageRegistry

	return createOption, nil
}

func createAction(cmd *cobra.Command, args []string) error {
	defer cleanTmpDirectory()

	if len(args) != 1 {
		return errors.New("please provide a valid command, create only accept one argument to path to the cluster stacks")
	}
	clusterStackPath := args[0]

	if mode != stableMode && mode != hashMode && mode != customMode {
		return fmt.Errorf("mode %q is not supported please choose from - stable, hash or custom", mode)
	}

	createOpts, err := GetCreateOptions(cmd.Context(), clusterStackPath)
	if err != nil {
		return fmt.Errorf("failed to create create options: %w", err)
	}

	// Validate if there any change or not
	if err := createOpts.validateHash(); err != nil {
		return fmt.Errorf("failed to validate with latest release hash: %w", err)
	}

	if err := createOpts.generateRelease(cmd.Context()); err != nil {
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
		return errors.New("no change in the cluster stack")
	}

	return nil
}

func (c *CreateOptions) generateRelease(ctx context.Context) error {
	if err := os.MkdirAll(c.ClusterStackReleaseDir, 0o750); err != nil {
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
		clusterAddonData, err := os.ReadFile(filepath.Join(".tmp", "cluster-addon-values.yaml"))
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

	if publish {
		if remote != "oci" {
			return errors.New("not pushing assets. --publish is only implemented for remote OCI")
		}

		ociClient, err := oci.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create new oci client: %w", err)
		}

		var hashAnnotation string
		if len(c.CurrentReleaseHash.ClusterStack) >= 7 {
			hashAnnotation = c.CurrentReleaseHash.ClusterStack[:7]
		}

		annotations := map[string]string{
			"kubernetesVersion": c.Metadata.Versions.Kubernetes,
			"hash":              hashAnnotation,
		}

		if err := pushReleaseAssets(ctx, ociClient, c.ClusterStackReleaseDir, c.releaseName, annotations); err != nil {
			return fmt.Errorf("failed to push release assets to the oci registry: %w", err)
		}
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
		return errors.New("failed to read version in yaml")
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

func pushReleaseAssets(ctx context.Context, pusher assetsclient.Pusher, clusterStackReleasePath, releaseName string, annotations map[string]string) error {
	releaseAssets := []assetsclient.ReleaseAsset{}

	ociclient, err := oci.NewClient()
	if err != nil {
		return fmt.Errorf("error creating oci client: %w", err)
	}

	if ociclient.FoundRelease(ctx, releaseName) {
		fmt.Printf("release tag \"%s\" found in oci registry. aborting push\n", releaseName)
		return nil
	}

	files, err := os.ReadDir(clusterStackReleasePath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", clusterStackReleasePath, err)
	}

	for _, file := range files {
		if file.Type().IsRegular() {
			releaseAssets = append(releaseAssets, assetsclient.ReleaseAsset{
				FileName:  file.Name(),
				MediaType: getMediaType(file.Name()),
			})
		}
	}

	if err := pusher.PushReleaseAssets(ctx, releaseAssets, releaseName, clusterStackReleasePath, clusterStackArtifactType, annotations); err != nil {
		return fmt.Errorf("failed to push release assets to oci registry: %w", err)
	}

	fmt.Printf("successfully pushed clusterstack release: %s:%s \n", ociclient.Repository.Reference.String(), releaseName)
	return nil
}
