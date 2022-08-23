package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	// 消息广播的channel
	Message chan string
}

// NewServer server接口
func NewServer(ip string, port int) *Server {
	return &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
}

// Start 启动服务的接口
func (this *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	// close listen socket
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener close err:", err)
		}
	}(listener)

	// 启动监听线程Message的goroutine,发送消息
	go this.BroadCast()

	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
		println(conn.RemoteAddr().String(), "连接成功")
		// do handler
		go this.Handler(conn)
	}

}

func (this *Server) Handler(conn net.Conn) {
	// 当前连接业务
	//fmt.Println("连接建立成功...")

	user := NewUser(conn, this)

	// 接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)

		for {
			n, err := conn.Read(buf)
			if n == 0 {
				this.SendMsg(user, "下线")
				return
			}
			if err != nil && err != io.EOF {
				println("conn read err:", err)
				return
			}
			// 提取用户的消息，去除'\n'
			msg := string(buf)
			this.SendMsg(user, msg)
		}
	}()

	// 当前handler阻塞
	select {}
}

func (this *Server) SendMsg(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg
}

func (this *Server) BroadCast() {

	for {
		msg := <-this.Message

		// 将msg发送给全部在线的User
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}

}
