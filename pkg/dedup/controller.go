package dedup

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

var SnapshotsPath = "/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/"

type PathToInodes struct {
	mu     sync.RWMutex
	lookup map[string][]string
}
type InodeToPaths struct {
	mu     sync.RWMutex
	lookup map[string][]string
}

var path2Inodes *PathToInodes

func init() {
	path2Inodes = &PathToInodes{
		mu:     sync.RWMutex{},
		lookup: map[string][]string{},
	}
}

func ReceiveLSOF(w http.ResponseWriter, r *http.Request) {
	// 从请求体中读取数据
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求体时出错", http.StatusInternalServerError)
		return
	}

	lines := strings.Split(string(data), "\n")

	// 处理收到的数据，这里简单打印出来
	fmt.Println("收到的 lsof 输出:")
	// fmt.Println(string(data))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[4] == "REG" {
			cpath := fields[len(fields)-1]
			inode := fields[len(fields)-2]
			if !path2Inodes.Exist(cpath, inode) {
				path2Inodes.Add(cpath, inode)
			}
			if ns, ok := path2Inodes.Lookup(cpath); ok {
				if len(ns) > 1 {
					log.Printf("Found more than one inode at path: %s, dedup triggered", cpath)
					go func(inodes []string) {
						for _, n := range inodes {
							go func(node string) {
								cmd := exec.Command("find", SnapshotsPath, "-inum", node)
								b, err := cmd.Output()
								if err != nil {
									log.Printf("Error when running find -inum %s", node)
								}
								log.Printf("inode: %s, hostpath: %s", node, string(b))
							}(n)
						}
					}(ns)
				}
			}
		}
	}
	log.Printf("path2inodes: %v\n", path2Inodes.lookup)
	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("成功接收 lsof 输出"))
}

func (m *PathToInodes) Lookup(key string) (inodes []string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inodes, ok = m.lookup[key]
	return
}

func (m *PathToInodes) Exist(cpath string, inode string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inodes, ok := m.lookup[cpath]
	if ok && Contain(inodes, inode) {
		return true
	}
	return false
}

func (m *PathToInodes) Add(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exist, ok := m.lookup[key]
	if !ok || (ok && !Contain(exist, value)) {
		m.lookup[key] = append(m.lookup[key], value)
		return
	}
}

func (m *InodeToPaths) Lookup(key string) (paths []string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	paths, ok = m.lookup[key]
	return
}

func (m *InodeToPaths) Add(key string, value []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	exist, ok := m.lookup[key]
	if !ok || (ok && !Contain(exist, key)) {
		m.lookup[key] = append(m.lookup[key], value...)
		return
	}
}

func Contain(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
