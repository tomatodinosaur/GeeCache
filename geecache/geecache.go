package geecache

import (
	"GeeCache/geecache/singleflight"
	pb "GeeCache/geecachepb"
	"fmt"
	"log"
	"sync"
)

// 接口函数
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
/*
一个 Group 可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。
比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。

第二个属性是 getter Getter，即缓存未命中时获取源数据的回调(callback)。

第三个属性是 mainCache cache，即一开始实现的并发缓存。

构建函数 NewGroup 用来实例化 Group，并且将 group 存储在全局变量 groups 中。
GetGroup 用来特定名称的 Group，这里使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作。
*/

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker

	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	v, ok := groups[name]
	if ok {
		return v
	}
	return nil
}

// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。
// 若是本机节点或失败，则回退到 getLocally()
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	//未击中
	return g.load(key)
}

// 先尝试从远程节点缓存查找，找不到再调本地的getter进行CallBack
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
// 	bytes, err := peer.Get(g.name, key)
// 	if err != nil {
// 		return ByteView{}, err
// 	}
// 	return ByteView{b: bytes}, nil
// }

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
