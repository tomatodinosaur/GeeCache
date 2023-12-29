package main

import (
	geecache "GeeCache"
	"fmt"
	"log"
	"net/http"
)

/*
type server int

http.ListenAndServe 接收 2 个参数，
第一个参数是服务启动的地址，
第二个参数是 Handler，
任何实现了 ServeHTTP 方法的对象都可以作为 HTTP 的 Handler。

	package http
	type Handler interface {
			ServeHTTP(w ResponseWriter, r *Request)
	}

//w:响应 r：请求

	func (h *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)
		w.Write([]byte("hello world!"))
	}

	func main() {
		var s server
		http.ListenAndServe("localhost:9999", &s)
	}
*/
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:9999"
	peers := geecache.NewHTTPPool(addr)

	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
