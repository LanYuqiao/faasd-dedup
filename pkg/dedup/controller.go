package dedup

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

var SnapshotsPath = "/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/"

type PathToInode struct {
	mu     sync.RWMutex
	lookup map[string]string
}
type InodeToPaths struct {
	mu     sync.RWMutex
	lookup map[string][]string
}

var path2Inode *PathToInode

var inode2Paths *InodeToPaths

func init() {
	path2Inode = &PathToInode{
		mu:     sync.RWMutex{},
		lookup: map[string]string{},
	}
	inode2Paths = &InodeToPaths{
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
			if v, ok := path2Inode.Lookup(cpath); ok {
				handleDuplication(cpath, inode, v)
			} else {
				path2Inode.Add(cpath, inode)
			}
			if _, ok := inode2Paths.Lookup(inode); !ok {
				go handleNewInode(inode, cpath)
			}
		}
	}
	log.Printf("path2inode: %v\n", path2Inode.lookup)
	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("成功接收 lsof 输出"))
}

func handleDuplication(cpath string, inode string, value string) {
	if inode == value {
		return
	}

}

func handleNewInode(inode string, cpath string) {
	var out []byte
	var err error
	cmd := exec.Command("find", SnapshotsPath, "-inum", inode)
	var b bytes.Buffer
	cmd.Stderr = &b
	out, err = cmd.Output()
	if err != nil {
		log.Fatalf("error when perform find -inum %s: %s", inode, err.Error())
	}
	if len(out) > 0 {
		hostpaths := strings.Split(string(out), "\n")
		inode2Paths.Add(inode, hostpaths)
	}
}

func (m *PathToInode) Lookup(key string) (inode string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inode, ok = m.lookup[key]
	return
}

func (m *PathToInode) Add(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.lookup[key]; !ok {
		m.lookup[key] = value
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
