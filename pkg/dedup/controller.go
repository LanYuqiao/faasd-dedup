package dedup

import (
	"fmt"
	"io"
	"net/http"
)

func ReceiveLSOF(w http.ResponseWriter, r *http.Request) {
	// 从请求体中读取数据
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求体时出错", http.StatusInternalServerError)
		return
	}

	// 处理收到的数据，这里简单打印出来
	fmt.Println("收到的 lsof 输出:")
	fmt.Println(string(data))

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("成功接收 lsof 输出"))
}
