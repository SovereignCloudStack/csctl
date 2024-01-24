package providerplugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/SovereignCloudStack/csmctl/pkg/clusterstack"
)

func CheckProviderExecutable(config *clusterstack.CsmctlConfig) (path string, err error) {
	pluginName := "csmctl-" + config.Config.Provider.Type
	_, err = os.Stat(pluginName)
	if err == nil {
		path, err := filepath.Abs(pluginName)
		if err != nil {
			return "", err
		}
		return path, err
	}
	path, err = exec.LookPath(pluginName)
	if err != nil {
		return "", fmt.Errorf("could not find plugin %s in $PATH or current working directory", pluginName)
	}
	return path, nil
}

func CreateNodeImages(config *clusterstack.CsmctlConfig, clusterStackPath string, clusterStackReleaseDir string) error {
	path, err := CheckProviderExecutable(config)
	if err != nil {
		return err
	}
	args := []string{"create-node-images", clusterStackPath, clusterStackReleaseDir}
	fmt.Printf("Calling Provider Plugin: %s\n", path)
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
