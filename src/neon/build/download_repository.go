package build

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type DownloadRepository interface {
	DownloadPlugin(string, string) ([]byte, error)
}

type HttpRepository struct {
	Root string
}

func NewHttpRepository(root string) DownloadRepository {
	repository := HttpRepository{
		Root: root,
	}
	return repository
}

func (repo HttpRepository) DownloadPlugin(plugin string) ([]byte, error) {
	plugin, version, _ = SplitRepositoryPath(plugin)
	if version == "" {
		// list plugin versions in repository
		url := repo.Root + "/" + plugin + "/"
		response, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("listing plugin versions at '%s': %v", url, err)
		}
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		return body, nil
	}
	return nil, nil
}
