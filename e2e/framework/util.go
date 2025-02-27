package framework

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/onsi/ginkgo"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

func BeforeAll(body func()) {
	once := sync.Once{}
	ginkgo.BeforeEach(func() {
		once.Do(func() {
			body()
		})
	})
}

func LoadConfigWithOptionsAndResolve(f factory.Factory, configPath string, configOptions *loader.ConfigOptions, resolveOptions dependency.ResolveOptions) (config.Config, []types.Dependency, error) {
	before, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	defer SwitchDir(before)

	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader(configPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return nil, nil, err
	} else if !configExists {
		return nil, nil, errors.New(message.ConfigNotFound)
	}

	// load config
	loadedConfig, err := configLoader.Load(configOptions, log)
	if err != nil {
		return nil, nil, err
	}

	// resolve dependencies
	dependencies, err := dependency.NewManager(loadedConfig, nil, configOptions, log).ResolveAll(resolveOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("error resolving dependencies: %v", err)
	}

	return loadedConfig, dependencies, nil
}

func LoadConfigWithOptions(f factory.Factory, configPath string, configOptions *loader.ConfigOptions) (config.Config, []types.Dependency, error) {
	return LoadConfigWithOptionsAndResolve(f, configPath, configOptions, dependency.ResolveOptions{})
}

func LoadConfig(f factory.Factory, configPath string) (config.Config, []types.Dependency, error) {
	return LoadConfigWithOptions(f, configPath, &loader.ConfigOptions{})
}

func InterruptChan() (chan error, func()) {
	once := sync.Once{}
	c := make(chan error)
	return c, func() {
		once.Do(func() {
			close(c)
		})
	}
}

func SwitchDir(dir string) {
	err := os.Chdir(dir)
	ExpectNoError(err)
}

func CleanupTempDir(initialDir, tempDir string) {
	err := os.RemoveAll(tempDir)
	ExpectNoError(err)

	err = os.Chdir(initialDir)
	ExpectNoError(err)
}

func CopyToTempDir(relativePath string) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	err = copy.Copy(relativePath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}

func ChangeToTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}
