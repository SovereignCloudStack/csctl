package clusterstack

import (
	"fmt"

	"github.com/SovereignCloudStack/csmctl/pkg/git"
	"github.com/SovereignCloudStack/csmctl/pkg/hash"
)

// HandleStableMode returns metadata for the stable mode.
func HandleStableMode(gitHubReleasePath string, currentReleaseHash, latestReleaseHash hash.ReleaseHash) (MetaData, error) {
	metadata, err := ParseMetaData(gitHubReleasePath)
	if err != nil {
		return MetaData{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata.Versions.ClusterStack, err = BumpVersion(metadata.Versions.ClusterStack)
	if err != nil {
		return MetaData{}, fmt.Errorf("failed to bump cluster stack: %w", err)
	}
	fmt.Printf("Bumped ClusterStack Version: %s\n", metadata.Versions.ClusterStack)

	if currentReleaseHash.ClusterAddonDir != latestReleaseHash.ClusterAddonDir || currentReleaseHash.ClusterAddonValues != latestReleaseHash.ClusterAddonValues {
		metadata.Versions.Components.ClusterAddon, err = BumpVersion(metadata.Versions.Components.ClusterAddon)
		if err != nil {
			return MetaData{}, fmt.Errorf("failed to bump cluster addon: %w", err)
		}
		fmt.Printf("Bumped ClusterAddon Version: %s\n", metadata.Versions.Components.ClusterAddon)
	} else {
		fmt.Printf("ClusterAddon Version unchanged: %s\n", metadata.Versions.Components.ClusterAddon)
	}

	if currentReleaseHash.NodeImageDir != latestReleaseHash.NodeImageDir {
		metadata.Versions.Components.NodeImage, err = BumpVersion(metadata.Versions.Components.NodeImage)
		if err != nil {
			return MetaData{}, fmt.Errorf("failed to bump node image: %w", err)
		}
		fmt.Printf("Bumped NodeImage Version: %s\n", metadata.Versions.Components.NodeImage)
	} else {
		if metadata.Versions.Components.NodeImage == "" {
			fmt.Println("No NodeImage Version.")
		} else {
			fmt.Printf("NodeImage Version unchanged: %s\n", metadata.Versions.Components.NodeImage)
		}
	}

	return metadata, nil
}

// HandleHashMode returns metadata of Hash mode.
func HandleHashMode(kubernetesVersion string) (MetaData, error) {
	commitHash, err := git.GetLatestGitCommit("./")
	if err != nil {
		return MetaData{}, fmt.Errorf("failed to get latest commit hash: %w", err)
	}

	commitHash = fmt.Sprintf("v0-sha.%s", commitHash)

	return MetaData{
		APIVersion: "metadata.clusterstack.x-k8s.io/v1alpha1",
		Versions: Versions{
			Kubernetes:   kubernetesVersion,
			ClusterStack: commitHash,
			Components: Component{
				ClusterAddon: commitHash,
				NodeImage:    commitHash,
			},
		},
	}, nil
}
