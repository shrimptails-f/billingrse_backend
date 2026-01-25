package oswrapper

// FileReaderInterface はファイル読み取り用の抽象です
type OsWapperInterface interface {
	ReadFile(path string) (string, error)
	GetEnv(key string) (string, error)
}
