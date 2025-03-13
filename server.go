package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip        string
	Port      int
	onlineMap map[string]*User
	mapLock   sync.RWMutex
	message   chan string
}

// 创建一个server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		onlineMap: make(map[string]*User),
		message:   make(chan string),
	}
	return server
}

// 监听message广播消息channel的goroutine，一旦有消息就发送给全部的在线user
func (s *Server) ListenMessage() {
	for {
		msg := <-s.message

		// 将msg发送给全部的在线user
		s.mapLock.Lock()
		for _, _user := range s.onlineMap {
			_user.C <- msg
		}
		s.mapLock.Unlock()
	}
}

// 广播消息的方法
func (s *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	s.message <- sendMsg
}

func (s *Server) ServerHandler(conn net.Conn) {
	user := NewUser(conn, s)
	fmt.Println("connect success")

	// 用户上线
	user.Online()

	// 监听用户是否活跃的channel
	isaLive := make(chan bool)

	// 接收客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("conn.Read err:", err)
				return
			}
			msg := string(buf[:n-1])
			user.DoMessage(msg)

			// 用户的任意消息，代表当前用户是活跃的
			isaLive <- true
		}
	}()

	// 当前handler阻塞
	for {
		select {
		case <-isaLive:
			// 当前用户是活跃的，应该重置定时器
		case <-time.After(time.Second * 300):
			// 已经超时
			// 将当前的user强制关闭
			user.SendMsg("你被踢了")
			close(user.C) // 关闭用户的channel
			conn.Close()  // 关闭用户的连接
			return        // runtime break
		}

	}

}

// 启动服务器的接口
func (s *Server) Start() {
	// socekt listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	// close listen socket
	defer listener.Close()

	// 启动监听Message的goroutine
	go s.ListenMessage()

	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}

		// do handler
		go s.ServerHandler(conn)
	}

}
