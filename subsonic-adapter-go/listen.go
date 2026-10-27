package main

import (
	"fmt"
	"log"
	"net"
)

// maxPortFallback 自动顺延时最多尝试多少个端口。
const maxPortFallback = 20

// listenWithFallback 监听 host:port。
//
// 行为约定：
//   - portExplicit 为 true（用户显式设置了 PORT）：端口被占直接返回错误，
//     不擅自改端口，以免与用户的部署脚本或反代配置不符。
//   - portExplicit 为 false（使用默认端口）：端口被占时自动顺延到后续可用端口，
//     并打印实际监听的端口，保证双击启动不会因为端口冲突而直接失败。
//
// 返回的 addr 是可打印的监听地址，ln 为已建立的监听器。
func listenWithFallback(host string, port int, portExplicit bool) (string, net.Listener, error) {
	tryListen := func(p int) (net.Listener, error) {
		return net.Listen("tcp", fmt.Sprintf("%s:%d", host, p))
	}

	ln, err := tryListen(port)
	if err == nil {
		return fmt.Sprintf("%s:%d", host, port), ln, nil
	}

	if portExplicit {
		log.Printf("端口 %d 已被占用（已通过 PORT 显式指定，不自动更换）。", port)
		log.Printf("请换一个端口，例如：PORT=%d ./subsonic-adapter-go", port+1)
		return "", nil, err
	}

	log.Printf("端口 %d 已被占用，尝试自动顺延…", port)
	for p := port + 1; p <= port+maxPortFallback; p++ {
		if l, e := tryListen(p); e == nil {
			log.Printf("已自动改用端口 %d（原端口 %d 被占用）。", p, port)
			return fmt.Sprintf("%s:%d", host, p), l, nil
		}
	}

	log.Printf("端口 %d 及后续 %d 个端口均被占用。", port, maxPortFallback)
	log.Printf("请显式指定可用端口，例如：PORT=5435 ./subsonic-adapter-go")
	return "", nil, err
}
