package dedup

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var SnapshotsPath = "/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/"

type PathToInodes struct {
	mu     sync.RWMutex
	lookup map[string][]uint64
}
type InodeToPaths struct {
	mu     sync.RWMutex
	lookup map[string][]string
}
type Tuple struct {
	inode uint64
	path  string
}

var path2Inodes *PathToInodes

func init() {
	path2Inodes = &PathToInodes{
		mu:     sync.RWMutex{},
		lookup: map[string][]uint64{},
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
			inode, err := strconv.ParseUint(fields[len(fields)-2], 10, 64)
			if err != nil {
				log.Fatalf("Cannot parse %s to uint64", fields[len(fields)-2])
			}
			if !path2Inodes.Exist(cpath, inode) {
				path2Inodes.Add(cpath, inode)
			}
			if ns, ok := path2Inodes.Lookup(cpath); ok {
				if len(ns) > 1 {
					log.Printf("Found more than one inode at path: %s, dedup triggered", cpath)
					go func(inodes []uint64) {
						var results []Tuple
						var wg sync.WaitGroup
						wg.Add(len(ns))
						for _, n := range inodes {
							var b []byte
							var err error

							go func(node uint64) {
								defer wg.Done()
								cmd := exec.Command("find", SnapshotsPath, "-inum", fmt.Sprintf("%d", node))
								b, err = cmd.Output()
								if err != nil {
									log.Printf("Error when running find -inum %d", node)
								}
								// log.Printf("inode: %s, hostpath: %s", node, string(b))
								results = append(results, Tuple{
									inode: node,
									path:  filterTruePath(strings.Fields(string(b)), cpath),
								})
							}(n)
							// log.Printf("inode: %s, hostpath: %s", n, string(b))

						}
						wg.Wait()
						log.Printf("%v", results)
						slices.SortFunc[[]Tuple, Tuple](results, func(a, b Tuple) int { return int(a.inode - b.inode) })
						preserved := results[0]
						for i, t := range results {
							if i == 0 {
								continue
							}
							// See https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/loopclosure
							t := t
							go func() {
								victim := t.path
								var err error
								err = syscall.Rename(victim, fmt.Sprintf("%s-victim", victim))
								if err != nil {
									log.Panicf("Error when rename %s", victim)
								}
								err = syscall.Link(preserved.path, victim)
								if err != nil {
									log.Panicf("Error when link %s to %s", preserved.path, victim)
								}
							}()
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

func filterTruePath(paths []string, cpath string) string {
	for _, p := range paths {
		if filepath.Base(cpath) == filepath.Base(p) {
			return p
		}
	}
	return ""
}

func (m *PathToInodes) Lookup(key string) (inodes []uint64, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inodes, ok = m.lookup[key]
	return
}

func (m *PathToInodes) Exist(cpath string, inode uint64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inodes, ok := m.lookup[cpath]
	if ok && Contain(inodes, inode) {
		return true
	}
	return false
}

func (m *PathToInodes) Add(key string, value uint64) {
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

// func (m *InodeToPaths) Add(key string, value []string) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	exist, ok := m.lookup[key]
// 	if !ok || (ok && !Contain(exist, key)) {
// 		m.lookup[key] = append(m.lookup[key], value...)
// 		return
// 	}
// }

func Contain(list []uint64, s uint64) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
