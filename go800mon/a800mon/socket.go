package a800mon

type SocketTransport struct {
	Path string
}

func NewSocketTransport(path string) *SocketTransport {
	return &SocketTransport{Path: path}
}
