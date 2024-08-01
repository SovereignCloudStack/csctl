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
	"fmt"
	"os"
	"sort"
	"strings"

	csoclusterstack "github.com/SovereignCloudStack/cluster-stack-operator/pkg/clusterstack"
	"github.com/SovereignCloudStack/cluster-stack-operator/pkg/version"
	"github.com/SovereignCloudStack/csctl/pkg/assetsclient"
	"github.com/SovereignCloudStack/csctl/pkg/clusterstack"
)

// getLatestReleaseFromRemoteRepository returns the latest release from the github repository.
func getLatestReleaseFromRemoteRepository(ctx context.Context, mode string, config *clusterstack.CsctlConfig, ac assetsclient.Client) (string, error) {
	ghReleases, err := ac.ListRelease(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list releases on remote Git repository: %w", err)
	}

	var clusterStacks csoclusterstack.ClusterStacks

	for _, ghRelease := range ghReleases {
		clusterStackObject, matches, err := matchesSpec(ghRelease, mode, config)
		if err != nil {
			return "", fmt.Errorf("failed to get match release tag %q with spec of ClusterStack: %w", ghRelease, err)
		}

		if matches {
			clusterStacks = append(clusterStacks, clusterStackObject)
		}
	}

	if len(clusterStacks) == 0 {
		return "", nil
	}

	sort.Sort(clusterStacks)

	str := clusterStacks.Latest().String()
	return str, nil
}

func matchesSpec(releaseTagName, mode string, cs *clusterstack.CsctlConfig) (csoclusterstack.ClusterStack, bool, error) {
	csObject, err := csoclusterstack.NewFromClusterStackReleaseProperties(releaseTagName)
	if err != nil {
		return csoclusterstack.ClusterStack{}, false, fmt.Errorf("failed to get clusterstack object from string %q: %w", releaseTagName, err)
	}

	kubernetesVersion, err := cs.ParseKubernetesVersion()
	if err != nil {
		return csoclusterstack.ClusterStack{}, false, fmt.Errorf("failed to parse kubernetes version %q: %w", cs.Config.ClusterStackName, err)
	}

	return csObject, csObject.Version.Channel == version.Channel(mode) &&
		csObject.KubernetesVersion.StringWithDot() == kubernetesVersion.StringWithDot() &&
		csObject.Name == cs.Config.ClusterStackName &&
		csObject.Provider == cs.Config.Provider.Type, nil
}

// downloadReleaseAssets downloads the specified release in the specified download path.
func downloadReleaseAssets(ctx context.Context, releaseTag, downloadPath string, ac assetsclient.Client) error {
	if err := ac.DownloadReleaseAssets(ctx, releaseTag, downloadPath); err != nil {
		// if download failed for some reason, delete the release directory so that it can be retried in the next reconciliation
		if err := os.RemoveAll(downloadPath); err != nil {
			return fmt.Errorf("failed to remove release: %w", err)
		}
		return fmt.Errorf("failed to download release assets: %w", err)
	}

	return nil
}

func getMediaType(fileName string) string {
	if fileName == "clusteraddon.yaml" {
		return clusterAddonConfigMediaType
	}

	if fileName == "metadata.yaml" {
		return metadataMediaType
	}

	if fileName == "node-images.yaml" {
		return nodeImageConfigMediaType
	}

	if fileName == "hashes.json" {
		return hashesMediaType
	}

	if strings.Contains(fileName, "cluster-addon") && strings.HasSuffix(fileName, ".tgz") {
		return clusterAddonMediaType
	}

	if strings.Contains(fileName, "cluster-class") && strings.HasSuffix(fileName, ".tgz") {
		return clusterClassMediaType
	}

	if strings.Contains(fileName, "node-image") && strings.HasSuffix(fileName, ".tgz") {
		return nodeImageMediaType
	}

	return ""
}
