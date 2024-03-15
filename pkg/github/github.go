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

package github

import (
	"context"
	"fmt"
	"sort"

	csoclusterstack "github.com/SovereignCloudStack/cluster-stack-operator/pkg/clusterstack"
	"github.com/SovereignCloudStack/cluster-stack-operator/pkg/version"
	"github.com/SovereignCloudStack/csctl/pkg/clusterstack"
	githubclient "github.com/SovereignCloudStack/csctl/pkg/github/client"
)

// GetLatestReleaseFromRemoteRepository returns the latest release from the github repository.
func GetLatestReleaseFromRemoteRepository(ctx context.Context, mode string, config *clusterstack.CsctlConfig, gc githubclient.Client) (string, error) {
	ghReleases, resp, err := gc.ListRelease(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list releases on remote Git repository: %w", err)
	}
	if resp != nil && resp.StatusCode != 200 {
		return "", fmt.Errorf("got unexpected status from call to remote Git repository: %s", resp.Status)
	}

	var clusterStacks csoclusterstack.ClusterStacks

	for _, ghRelease := range ghReleases {
		clusterStackObject, matches, err := matchesSpec(ghRelease.GetTagName(), mode, config)
		if err != nil {
			return "", fmt.Errorf("failed to get match release tag %q with spec of ClusterStack: %w", ghRelease.GetTagName(), err)
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
