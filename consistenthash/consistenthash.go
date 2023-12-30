package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//解决分布式下，由一个KEY值去找哪个节点获取数据的问题
//输入key string,输出节点名称 string
//Hash算法用来将固定的key值映射到固定的节点上
//为了防止发送缓存雪崩，采用一致性Hash算法
//为了解决数据倾斜问题，采用一个真实节点对应多个虚拟节点的方式

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash  //哈希函数
	replicas int   //虚拟节点数量
	nodes    []int //sorted 哈希环

	//虚拟节点与真实节点的映射表
	//<键是虚拟节点的哈希值，值是真实节点的名称>
	hashMap map[int]string
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加若干节点
func (m *Map) Add(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < m.replicas; i++ {
			//虚拟节点key-i在哈希环的位置
			hash := int(m.hash([]byte(strconv.Itoa(i) + node)))
			m.nodes = append(m.nodes, hash)
			//所有的虚拟节点都指向 node
			m.hashMap[hash] = node
		}
	}
	sort.Ints(m.nodes)
}

// Get gets the closest node in the hash to the provided key
func (m *Map) Get(key string) string {
	if len(m.nodes) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))

	idx := sort.Search(len(m.nodes), func(i int) bool {
		return m.nodes[i] >= hash
	})
	nodeHash := m.nodes[idx%len(m.nodes)]
	return m.hashMap[nodeHash]
}

// 删除节点
func (m *Map) Remove(node string) bool {
	if len(m.nodes) == 0 {
		return true
	}
	for i := 0; i < m.replicas; i++ {
		//虚拟节点key-i在哈希环的位置
		hash := int(m.hash([]byte(strconv.Itoa(i) + node)))
		idx := sort.SearchInts(m.nodes, hash) % len(m.nodes)
		if m.nodes[idx] != hash {
			return false
		}
		m.nodes = append(m.nodes[:idx], m.nodes[idx+1:]...)
		delete(m.hashMap, hash)
	}
	return true
}
