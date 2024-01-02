package geecache

// 远程节点选择器
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// 远程节点
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
