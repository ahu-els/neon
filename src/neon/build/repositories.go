package build

const (
	DEFAULT_LOCAL_REPOSITORY    = "~/.neon"
	DEFAULT_DOWNLOAD_REPOSITORY = "http://sweetohm.net/neon"
)

type Repositories struct {
	Local    LocalRepository
	Download []DownloadRepository
}

func NewRepositories() Repositories {
	repositories := Repositories{
		Local:    NewFileRepository(DEFAULT_LOCAL_REPOSITORY),
		Download: []DownloadRepository{NewHttpRepository(DEFAULT_DOWNLOAD_REPOSITORY)},
	}
	return repositories
}

func (repos Repositories) GetResource(path string) ([]byte, error) {
	return repos.Local.GetResource(path)
}

func (repos Repositories) Get(plugin string) ([]byte, error) {
	var err error
	var bytes []byte
	for _, repo := range repos.Download {
		bytes, err = repo.Get(plugin)
		if err == nil {
			return bytes, nil
		}
	}
	return nil, fmt.Errorf("Could not find plugin '%s' in repositories", plugin)
}
