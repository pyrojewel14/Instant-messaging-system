package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}

// 创建一个user的接口
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C:    make(chan string),
		conn: conn,

		server: server,
	}
	// 启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (user *User) Online() {
	// 用户上线，将用户加入到onlineMap中
	user.server.mapLock.Lock()
	user.server.onlineMap[user.Name] = user
	user.server.mapLock.Unlock()

	// 广播用户上线消息
	user.server.BroadCast(user, "已上线")
}

// 用户下线业务
func (user *User) Offline() {
	// 用户下线，将用户从onlineMap中删除
	user.server.mapLock.Lock()
	delete(user.server.onlineMap, user.Name)
	user.server.mapLock.Unlock()

	// 广播用户下线消息
	user.server.BroadCast(user, "已下线")
}

// 给当前user对应的客户端发送消息
func (user *User) SendMsg(msg string) {
	user.conn.Write([]byte(msg))
}

// 用户消息业务
func (user *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户
		user.server.mapLock.Lock()
		for _, alluser := range user.server.onlineMap {
			onlineMsg := "[" + alluser.Addr + "]" + alluser.Name + ":" + "在线\n"
			user.SendMsg(onlineMsg)
		}
		user.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 消息格式：rename|张三
		newName := strings.Split(msg, "|")[1]
		// 判断name是否存在
		_, ok := user.server.onlineMap[newName]
		if ok {
			user.SendMsg("当前用户名已被使用\n")
		} else {
			user.server.mapLock.Lock()
			delete(user.server.onlineMap, user.Name)
			user.server.onlineMap[newName] = user
			user.server.mapLock.Unlock()

			user.Name = newName
			user.SendMsg("您已更新用户名：" + user.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式：to|张三|消息内容

		// 获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			user.SendMsg("消息格式不正确，请使用\"to|张三|你好\"格式\n")
			return
		}

		// 获取对方用户对象
		remoteUser, ok := user.server.onlineMap[remoteName]
		if !ok {
			user.SendMsg("该用户名不存在\n")
			return
		}

		// 获取消息内容
		content := strings.Split(msg, "|")[2]
		if content == "" {
			user.SendMsg("无消息内容，请重发\n")
			return
		}

		// 发送消息
		remoteUser.SendMsg(user.Name + "对您说：" + content + "\n")

	} else if msg == "exit" {
		// 用户下线
		user.Offline()
	} else if msg == "help" {
		user.SendMsg("支持以下命令：\n")
		user.SendMsg("who:查询当前在线用户\n")
		user.SendMsg("rename|张三:修改用户名\n")
		user.SendMsg("to|张三|你好:发送消息给张三\n")
		user.SendMsg("exit:退出\n")
	} else {
		// 广播消息
		user.server.BroadCast(user, msg)
	}
}

// 监听当前user channel的方法，一旦有消息，就直接发送给对端客户端
func (user *User) ListenMessage() {
	for {
		msg := <-user.C
		user.SendMsg(("收到消息" + msg + "\n"))
	}
}
