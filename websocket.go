package appcom

import (
	"bytes"
	"errors"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	ClientChanSize = 1024 * 4
	maxMessageSize = 1024 * 64

	RecvBufferSize = 64 * 1024
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	hub  *WebSocketServer
	conn *websocket.Conn
	Send chan []byte

	Id    int64
	Meta  string
	RCall func(*Client, []byte)
}

// websocket读取线程
func (c *Client) goRecv() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}

			break
		}

		// 将数据的\n用''来代替//
		// 同时处理数据包的大小 //
		// 定义数据格式 //
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		c.RCall(c, message)
		// 对消息进行处理
	}
}

// websocket连接写线程
func (this *Client) goSend() {
	defer func() {
		this.hub.unregister <- this
		this.conn.Close()
		log.Error("Client connect close!")
	}()

	for {
		select {
		case message, ok := <-this.Send:
			if !ok {
				log.Println("WebSocket Send over? ok: ", ok)

				return
			}

			/**
			* 需要设置为发送二进制数据，否则浏览器会报出UTF-8解码错误
			 */
			w, err := this.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Println("WebSocket Send over? NextWriter: ", err)

				return
			}

			w.Write(message)
			if err := w.Close(); err != nil {
				log.Println("WebSocket Send over? Close: ", err)

				return
			}

			n := len(this.Send)
			for i := 0; i < n-1; i++ {
				data := <-this.Send
				w, err := this.conn.NextWriter(websocket.BinaryMessage)
				if err != nil {
					log.Println("WebSocket Send over? NextWriter: ", err)

					continue
				}

				runtime.Gosched()
				w.Write(data)

				if err := w.Close(); err != nil {
					log.Println("WebSocket Send over? Close: ", err)

					continue
				}
			}
		}
	}

}

type ClientManage struct {
	Clients  map[int64]*Client
	Connects int64

	ConnectsLock sync.Mutex
}

func NewClientManage() *ClientManage {
	v := &ClientManage{
		Connects:     0,
		ConnectsLock: sync.Mutex{},
		Clients:      make(map[int64]*Client),
	}

	return v
}

// 通过id获取client连接
//
// @param id
//
func (this *ClientManage) ClientById(id int64) (client *Client) {
	if _, ok := this.Clients[id]; !ok {
		client = nil

		return
	}

	return this.Clients[id]
}

// 添加客户端到管理器中
// 如果对应的id已经存在，则关闭已经存在的，然后设置为新的client
//
func (this *ClientManage) AddClient(client *Client) {
	this.ConnectsLock.Lock()
	defer this.ConnectsLock.Unlock()
	if _, ok := this.Clients[client.Id]; ok {
		delete(this.Clients, client.Id)
		close(client.Send)
		this.Connects--
	}

	this.Clients[client.Id] = client
	this.Connects++
	return
}

// 从客户端管理器中移除client
//
func (this *ClientManage) RemoveClient(client *Client) {
	this.ConnectsLock.Lock()
	defer this.ConnectsLock.Unlock()
	if _, ok := this.Clients[client.Id]; ok {
		delete(this.Clients, client.Id)
		close(client.Send)
	}

	this.Connects--
	return
}

// 移除所有client
//
func (this *ClientManage) RemoveAll() {
	this.ConnectsLock.Lock()
	defer this.ConnectsLock.Unlock()

	for _, client := range this.Clients {
		delete(this.Clients, client.Id)
		close(client.Send)
	}

	return
}

// 将数据发生到对应的netid的客户端
//
// @param netid 	客户端对应的id
// @param data 		要发送的数据
//
func (this *ClientManage) Send(netid int64, data []byte) (err error) {
	if 0 >= netid {
		err = errors.New("Send netid is error")

		return
	}

	if 0 == len(data) {
		err = errors.New("Send data is nil")

		return
	}

	if _, ok := this.Clients[netid]; !ok {
		err = errors.New("Netid to client not exists.")

		return
	}

	if ClientChanSize < (len(this.Clients[netid].Send) + 10) {
		err = errors.New("Send chan is full")

		return
	}

	this.Clients[netid].Send <- data
	return
}

// 广播数据到所有连接
//
// @param msg
//
func (this *ClientManage) Broadcast(msg []byte) (err error) {
	if 0 == len(msg) {
		err = errors.New("Broadcast data is nil")

		return
	}

	for _, client := range this.Clients {
		if ClientChanSize < (len(client.Send) + 10) {
			err = errors.New("Broadcast other chan is full")

			continue
		}

		client.Send <- msg
	}

	return
}

type WebSocketServer struct {
	register   chan *Client
	unregister chan *Client
	stop       chan bool
	isRun      bool

	Clients *ClientManage
	Status  string
}

var gWebSocketServer *WebSocketServer

// 创建性的WebsocketServer对象
func GetWebSocketServer() *WebSocketServer {
	if nil == gWebSocketServer {
		gWebSocketServer = &WebSocketServer{
			register:   make(chan *Client),
			unregister: make(chan *Client),
			stop:       make(chan bool),
			Status:     "New",
			isRun:      false,
			Clients:    NewClientManage(),
		}
	}

	return gWebSocketServer
}

// 获取client管理结构
//
func (this *WebSocketServer) ClientManage() *ClientManage {
	return this.Clients
}

// 停止服务运行
//
func (this *WebSocketServer) Stop() {
	this.stop <- true
	this.Status = "Stop"

	return
}

// websocket建立的状态服务
// 主要是监听连接状态
//
func (h *WebSocketServer) Run() {
	if h.isRun {
		return
	}

	h.isRun = true
	h.Status = "Run"
	for {
		select {
		case client := <-h.register:
			h.Clients.AddClient(client)

			break

		case client := <-h.unregister:
			h.Clients.RemoveClient(client)

			break

		// 停止websocket服务运行
		case <-h.stop:
			h.isRun = false
			h.Clients.RemoveAll()

			return
		}
	}
}

// 创建websocket连接
// hub 		websocket服务对象
// w 		http请求的响应对象
// r 	 	http请求的请求对象
// id 		连接对应的标识id
// meta 	client对应的元信息
// rcall 	websocket读取数据的回调函数
// onCall 	websocket建立的回调函数
//
func WebSocketConn(hub *WebSocketServer, w http.ResponseWriter, r *http.Request, id int64, meta string, rcall func(*Client, []byte),
	onCall func(client *Client, state bool)) {
	var upgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},

		ReadBufferSize:  1024 * 4,      // 4kb
		WriteBufferSize: 64 * 1024 * 1, // 64kb
	}

	/**
	* 设置响应头部，要不会报错，也就是不设置就连接不上
	 */
	var headers http.Header = make(http.Header)
	headers.Add("Sec-WebSocket-Protocol", "null")

	conn, err := upgrade.Upgrade(w, r, headers)
	if nil != err {
		log.Println(err)
		onCall(nil, false)

		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		Send: make(chan []byte,
			ClientChanSize),
		RCall: rcall,
		Id:    id,
		Meta:  meta,
	}

	client.hub.register <- client

	go client.goRecv()
	go client.goSend()

	onCall(client, true)

	return
}
