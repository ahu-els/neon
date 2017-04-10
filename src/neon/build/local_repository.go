package build

import (
	"fmt"
	"io/ioutil"
	"neon/util"
	"path/filepath"
	"sort"
	"strings"
)

type LocalRepository interface {
	GetResource(path string) ([]byte, error)
	InstallPlugin(plugin, version string, data []byte) error
}

type FileRepository struct {
	Root string
}

func NewFileRepository(root string) LocalRepository {
	root = util.ExpandUserHome(root)
	repository := FileRepository{
		Root: root,
	}
	return repository
}

func (repo FileRepository) GetResource(path string) ([]byte, error) {
	if IsRepositoryPath(path) {
		plugin, version, artifact, err := SplitRepositoryPath(path)
		if err != nil {
			return nil, err
		}
		directory := filepath.Join(repo.Root, plugin)
		if !util.FileExists(directory) {
			return nil, fmt.Errorf("plugin '%s' not found (download it with 'neon -get %s')", plugin, plugin)
		}
		if version == "" {
			dirs, err := ioutil.ReadDir(directory)
			if err != nil {
				return nil, fmt.Errorf("listing plugin directory: %v", err)
			}
			var versions = make([]util.Version, len(dirs))
			for i, dir := range dirs {
				if !dir.IsDir() {
					return nil, fmt.Errorf("bad '%s' plugin structure: '%s' is not a directory", plugin, dir.Name())
				}
				versions[i], err = util.NewVersion(dir.Name())
				if err != nil {
					return nil, err
				}
			}
			sort.Sort(util.Versions(versions))
			version = versions[len(versions)-1].Name
		}
		file := filepath.Join(repo.Root, plugin, version, artifact)
		resource, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("loading resource '%s': %v", file, err)
		}
		return resource, nil
	} else {
		bytes, err := util.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading resource '%s': %v", path, err)
		}
		return bytes, nil
	}
}

func (repo FileRepository) InstallPlugin(plugin, version string, data []byte) error {
	return nil
}

func IsRepositoryPath(path string) bool {
	return strings.HasPrefix(path, ":")
}

func SplitRepositoryPath(path string) (string, string, string, error) {
	if IsRepositoryPath(path) {
		parts := strings.Split(path[1:], ":")
		if len(parts) < 1 || len(parts) > 3 {
			return "", "", "", fmt.Errorf("Bad Neon path '%s'", path)
		}
		if len(parts) == 1 {
			parts = []string{parts[0], "", ""}
		} else if len(parts) == 2 {
			parts = []string{parts[0], "", parts[1]}
		} else {
			parts = []string{parts[0], parts[1], parts[2]}
		}
		return parts, nil
	} else {
		return "", "", "", fmt.Errorf("'%s' is not a repository path", path)
	}
}
