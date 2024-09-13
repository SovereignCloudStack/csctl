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

	"github.com/SovereignCloudStack/csctl/pkg/assetsclient"
	"github.com/SovereignCloudStack/csctl/pkg/assetsclient/github"
	"github.com/SovereignCloudStack/csctl/pkg/assetsclient/oci"
	"github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	"github.com/SovereignCloudStack/csctl/pkg/hash"
	"github.com/SovereignCloudStack/csctl/pkg/providerplugin"
	"github.com/SovereignCloudStack/csctl/pkg/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	publishShortDescription = "Creates a cluster stack release with the help of given cluster stack and push it to the oci registry."
	publishLongDescription  = `It takes cluster stacks and mode as an input and based on that creates
	the cluster stack release in the current directory named "release/".
	Supported modes are - stable, hash, custom

	note - Hash mode takes the last hash of the git commit.`
	publishExample = `csctl publish tests/cluster-stacks/docker/ferrol -m hash (for hash mode)

	csctl publish tests/cluster-stacks/docker/ferrol (for stable mode)`
)

// PublishOptions has the options for the csctl create command.
type PublishOptions struct {
	ClusterStackPath                          string
	clusterStackReleaseTemporaryOutputDirName string
	clusterStackReleaseDirName                string
	latestRepoReleasePath                     string
	releaseName                               string
	NewClusterStackConvention                 bool
	Config                                    *clusterstack.CsctlConfig
	Metadata                                  *clusterstack.MetaData

	// CurrentReleaseHash represent current clusterstack hash.
	CurrentReleaseHash hash.ReleaseHash

	// LatestReleaseHash represent latest release hash from github.
	LatestReleaseHash hash.ReleaseHash
}

// createCmd represents the create command.
var publishCmd = &cobra.Command{
	Use:          "publish",
	Short:        publishShortDescription,
	Long:         publishLongDescription,
	Example:      publishExample,
	RunE:         publishAction,
	SilenceUsage: true,
}

func init() {
	publishCmd.Flags().StringVarP(&mode, "mode", "m", "stable", "It defines the mode of the cluster stack manager")
	publishCmd.Flags().StringVarP(&outputDirectory, "output", "o", "./.release", "It defines the output directory in which the release assets will be generated")
	publishCmd.Flags().StringVarP(&nodeImageRegistry, "node-image-registry", "r", "", "It defines the node image registry. For example oci://ghcr.io/foo/bar/node-images/staging/")
	publishCmd.Flags().StringVar(&clusterStackVersion, "cluster-stack-version", "", "It is used to specify the semver version for the cluster stack in the custom mode")
	publishCmd.Flags().StringVar(&clusterAddonVersion, "cluster-addon-version", "", "It is used to specify the semver version for the cluster addon in the custom mode")
	publishCmd.Flags().StringVar(&nodeImageVersion, "node-image-version", "", "It is used to specify the semver version for the node images in the custom mode")
}

// GetPublishOptions create a Pubish Option for publish command.
func GetPublishOptions(ctx context.Context, clusterStackPath string) (*PublishOptions, error) {
	publishOption := &PublishOptions{}

	// ClusterAddon config
	config, err := clusterstack.GetCsctlConfig(clusterStackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	publishOption.ClusterStackPath = clusterStackPath
	publishOption.Config = config

	// ClusterStack convention
	if _, err := os.Stat(filepath.Join(clusterStackPath, "clusteraddon.yaml")); err != nil {
		// old if clusteraddon.yaml is not present.
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to find clusteraddon.yaml: %w", err)
		}
	} else {
		// new if clusteraddon.yaml is present.
		publishOption.NewClusterStackConvention = true
	}

	currentHash, err := hash.GetHash(clusterStackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash: %w", err)
	}
	publishOption.CurrentReleaseHash = currentHash

	switch mode {
	case hashMode:
		publishOption.Metadata = clusterstack.HandleHashMode(publishOption.CurrentReleaseHash, config.Config.KubernetesVersion)
	case stableMode:
		publishOption.Metadata = &clusterstack.MetaData{}

		gc, err := github.NewFactory().NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create new github client: %w", err)
		}

		latestRepoRelease, err := getLatestReleaseFromRemoteRepository(ctx, mode, config, gc)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest release from remote repository: %w", err)
		}
		fmt.Printf("latest release found: %q\n", latestRepoRelease)

		if latestRepoRelease == "" {
			publishOption.Metadata.APIVersion = "metadata.clusterstack.x-k8s.io/v1alpha1"
			publishOption.Metadata.Versions.Kubernetes = config.Config.KubernetesVersion
			publishOption.Metadata.Versions.ClusterStack = "v1"
			publishOption.Metadata.Versions.Components.ClusterAddon = "v1"
			publishOption.Metadata.Versions.Components.NodeImage = "v1"
		} else {
			publishOption.latestRepoReleasePath = filepath.Join(".tmp", "release", latestRepoRelease)

			if err := downloadReleaseAssets(ctx, latestRepoRelease, publishOption.latestRepoReleasePath, gc); err != nil {
				return nil, fmt.Errorf("failed to download release asset: %w", err)
			}

			publishOption.LatestReleaseHash, err = hash.ParseReleaseHash(filepath.Join(publishOption.latestRepoReleasePath, "hashes.json"))
			if err != nil {
				return nil, fmt.Errorf("failed to read hash from the github: %w", err)
			}

			publishOption.Metadata, err = clusterstack.HandleStableMode(publishOption.latestRepoReleasePath, publishOption.CurrentReleaseHash, publishOption.LatestReleaseHash)
			if err != nil {
				return nil, fmt.Errorf("failed to handle stable mode: %w", err)
			}

			// update the metadata kubernetes version with the csctl.yaml config
			publishOption.Metadata.Versions.Kubernetes = config.Config.KubernetesVersion
		}
	case customMode:
		if clusterStackVersion == "" {
			return nil, fmt.Errorf("please specify a semver for custom version with --cluster-stack-version flag")
		}
		if clusterAddonVersion == "" {
			return nil, fmt.Errorf("please specify a semver for custom version with --cluster-addon-version flag")
		}
		if nodeImageVersion == "" {
			return nil, fmt.Errorf("please specify a semver for custom version with --node-image-version flag")
		}

		publishOption.Metadata, err = clusterstack.HandleCustomMode(publishOption.Config.Config.KubernetesVersion, clusterStackVersion, clusterAddonVersion, nodeImageVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to handle custom mode: %w", err)
		}
	}

	releaseDirName, err := clusterstack.GetClusterStackReleaseDirectoryName(publishOption.Metadata, publishOption.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stack release directory name: %w", err)
	}

	publishOption.releaseName, err = clusterstack.GetClusterStackReleaseDirectoryName(publishOption.Metadata, publishOption.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stack release name: %w", err)
	}

	publishOption.clusterStackReleaseTemporaryOutputDirName = filepath.Join(".tmp", releaseDirName)
	publishOption.clusterStackReleaseDirName = filepath.Join(outputDirectory, releaseDirName)

	if err := os.MkdirAll(publishOption.clusterStackReleaseDirName, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temporary directory for clusterstack: %w", err)
	}

	return publishOption, nil
}

func publishAction(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please provide a valid command, create only accept one argument to path to the cluster stacks")
	}
	clusterStackPath := args[0]

	if mode != stableMode && mode != hashMode && mode != customMode {
		fmt.Println("The mode is ", mode)
		return fmt.Errorf("mode is not supported please choose from - stable, hash, custom")
	}

	publishOpts, err := GetPublishOptions(cmd.Context(), clusterStackPath)
	if err != nil {
		return fmt.Errorf("failed to create publish options: %w", err)
	}

	// clean the clusterstack templated output
	defer cleanTmpDirectory()

	// Validate if there any change or not
	if err := publishOpts.validateHash(); err != nil {
		return fmt.Errorf("failed to validate with latest release hash: %w", err)
	}

	if err := publishOpts.generateRelease(cmd.Context()); err != nil {
		return fmt.Errorf("failed to generate release: %w", err)
	}
	fmt.Printf("Created %s\n", publishOpts.clusterStackReleaseDirName)

	return nil
}

// validateHash returns if some hash changes or not.
func (p *PublishOptions) validateHash() error {
	if p.CurrentReleaseHash.ClusterStack == p.LatestReleaseHash.ClusterStack {
		return fmt.Errorf("no change in the cluster stack")
	}

	return nil
}

func (p *PublishOptions) generateRelease(ctx context.Context) error {
	// Write the current hash
	hashJSONData, err := json.MarshalIndent(p.CurrentReleaseHash, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hash json: %w", err)
	}

	filePath := filepath.Join(p.clusterStackReleaseDirName, "hashes.json")
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
	if err := template.GenerateOutputFromTemplate(p.ClusterStackPath, p.clusterStackReleaseTemporaryOutputDirName, p.Metadata); err != nil {
		return fmt.Errorf("failed to generate tmp output: %w", err)
	}

	// Overwrite ClusterAddonVersion in cluster-addon/*/Chart.yaml
	if err := overwriteClusterAddonVersion(p.clusterStackReleaseTemporaryOutputDirName, p.Metadata.Versions.Components.ClusterAddon); err != nil {
		return fmt.Errorf("failed to overwrite ClusterAddonVersion in tmp output: %w", err)
	}

	// Overwrite ClusterClassVersion in cluster-class/Chart.yaml
	clusterClassChartYaml := filepath.Join(p.clusterStackReleaseTemporaryOutputDirName, "cluster-class", "Chart.yaml")
	fmt.Printf("clusterclass chart path: %s", clusterClassChartYaml)
	if err := overwriteVersionInFile(clusterClassChartYaml, p.Metadata.Versions.ClusterStack); err != nil {
		return fmt.Errorf("failed to overwrite ClusterClassVersion in %s output: %w", clusterClassChartYaml, err)
	}

	// Package Helm from the tmp directory to the release directory
	if err := template.CreatePackage(p.clusterStackReleaseTemporaryOutputDirName, p.clusterStackReleaseDirName, p.NewClusterStackConvention, p.Config, p.Metadata); err != nil {
		return fmt.Errorf("failed to create template package: %w", err)
	}

	if p.NewClusterStackConvention {
		// Copy the clusteraddon.yaml config to release if new way
		clusterAddonData, err := os.ReadFile(filepath.Join(p.ClusterStackPath, "clusteraddon.yaml"))
		if err != nil {
			return fmt.Errorf("failed to read clusteraddon.yaml: %w", err)
		}

		if err := os.WriteFile(filepath.Join(p.clusterStackReleaseDirName, "clusteraddon.yaml"), clusterAddonData, os.FileMode(0o644)); err != nil {
			return fmt.Errorf("failed to write clusteraddon.yaml: %w", err)
		}
	} else {
		// Copy the cluster-addon-values.yaml config to release if old way
		clusterAddonData, err := os.ReadFile(filepath.Join(p.ClusterStackPath, "cluster-addon-values.yaml"))
		if err != nil {
			return fmt.Errorf("failed to read cluster-addon-values.yaml: %w", err)
		}

		if err := os.WriteFile(filepath.Join(p.clusterStackReleaseDirName, "cluster-addon-values.yaml"), clusterAddonData, os.FileMode(0o644)); err != nil {
			return fmt.Errorf("failed to write cluster-addon-values.yaml: %w", err)
		}
	}

	// Put the final metadata file into the output directory.
	metaDataByte, err := yaml.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata yaml: %w", err)
	}

	metadataFile, err := os.Create(filepath.Join(p.clusterStackReleaseDirName, "metadata.yaml"))
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metadataFile.Close()

	if _, err := metadataFile.Write(metaDataByte); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	ociClient, err := oci.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create new oci client: %w", err)
	}

	var hashAnnotation string
	if len(p.CurrentReleaseHash.ClusterStack) >= 7 {
		hashAnnotation = p.CurrentReleaseHash.ClusterStack[:7]
	}

	annotations := map[string]string{
		"kubernetesVersion": p.Metadata.Versions.Kubernetes,
		"hash":              hashAnnotation,
	}

	// Generate the node-images.yaml file in the release directory
	err = providerplugin.CreateNodeImages(p.Config,
		p.ClusterStackPath,
		p.clusterStackReleaseDirName,
		nodeImageRegistry)
	if err != nil {
		return fmt.Errorf("providerplugin.CreateNodeImages() failed: %w", err)
	}

	// push clusterstack to the remote registry.
	if err := pushReleaseAssets(ctx, ociClient, p.clusterStackReleaseDirName, p.releaseName, annotations); err != nil {
		return fmt.Errorf("failed to push release assets to the oci registry: %w", err)
	}

	return nil
}

func pushReleaseAssets(ctx context.Context, pusher assetsclient.Pusher, clusterStackReleasePath, releaseName string, annotations map[string]string) error {
	releaseAssets := []assetsclient.ReleaseAsset{}

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

	ociclient, err := oci.NewClient()
	if err != nil {
		return fmt.Errorf("error creating oci client: %w", err)
	}

	fmt.Printf("successfully pushed clusterstack release: %s:%s \n", ociclient.Repository.Reference.String(), releaseName)
	return nil
}
