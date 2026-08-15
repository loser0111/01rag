package document

// 文件读取函数

type IFileReader interface {
	Read(file *File) ([]*Chunk, error)
}

// 文件定义

type File struct {
	FileName string
	FilePath string
}

// 数据块内容

type Chunk struct {
	Data string
}
