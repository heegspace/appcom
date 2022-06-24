package appcom

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"

	"go-micro.dev/v4/logger"
)

var (
	DefaultMaxMessageSize = int(1 << 20)
)

func byteArrayToUInt32(bytes []byte) (result int64, bytesRead int) {
	return binary.Varint(bytes)
}

func intToByteArray(value int64, bufferSize int) []byte {
	toWriteLen := make([]byte, bufferSize)
	binary.PutVarint(toWriteLen, value)
	return toWriteLen
}

type ListenCb func(context.Context, *net.TCPListener) error
type RecvCb func(context.Context, *net.TCPConn, []byte) error
type ConnectCb func(context.Context, *net.TCPConn) error
type CloseCb func(context.Context, *net.TCPConn) error
type TCPListener struct {
	socket          *net.TCPListener
	address         string
	headerByteSize  int
	maxMessageSize  int
	listenCb        ListenCb
	connectCb       ConnectCb
	recvCb          RecvCb
	closeCb         CloseCb
	shutdownChannel chan struct{}
	shutdownGroup   *sync.WaitGroup
}

type TCPListenerConfig struct {
	MaxMessageSize int
	EnableLogging  bool
	Address        string
	ListenCb       ListenCb
	ConnectCb      ConnectCb
	RecvCb         RecvCb
	CloseCb        CloseCb
}

func ListenTCP(cfg TCPListenerConfig) (*TCPListener, error) {
	maxMessageSize := DefaultMaxMessageSize
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	btl := &TCPListener{
		maxMessageSize:  maxMessageSize,
		headerByteSize:  4, // 4byte(int32)
		listenCb:        cfg.ListenCb,
		connectCb:       cfg.ConnectCb,
		recvCb:          cfg.RecvCb,
		closeCb:         cfg.CloseCb,
		shutdownChannel: make(chan struct{}),
		address:         cfg.Address,
		shutdownGroup:   &sync.WaitGroup{},
	}

	if err := btl.openSocket(); err != nil {
		return nil, err
	}

	return btl, nil
}

func (btl *TCPListener) blockListen() error {
	for {
		conn, err := btl.socket.AcceptTCP()
		if err != nil {
			logger.Error("[blockListen] Error attempting to accept connection: ", err)
			select {
			case <-btl.shutdownChannel:
				return nil
			default:
			}
		} else {
			go handleListenedConn(conn, btl.maxMessageSize, btl.recvCb, btl.closeCb, btl.shutdownGroup)
		}
	}
}

func (btl *TCPListener) openSocket() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", btl.address)
	if err != nil {
		return err
	}
	receiveSocket, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	btl.socket = receiveSocket
	btl.listenCb(context.TODO(), receiveSocket)
	return err
}

func (btl *TCPListener) StartListening() error {
	return btl.blockListen()
}

func (btl *TCPListener) Close() {
	close(btl.shutdownChannel)
	btl.shutdownGroup.Wait()
}

func (btl *TCPListener) StartListeningAsync() error {
	var err error
	go func() {
		err = btl.blockListen()
	}()
	return err
}

func handleListenedConn(conn *net.TCPConn, maxMessageSize int, rcb RecvCb, ccb CloseCb, sdGroup *sync.WaitGroup) {
	// sdGroup.Add(1)
	// defer sdGroup.Done()
	dataBuffer := make([]byte, maxMessageSize)
	defer func() {
		if err := recover(); nil != err {
			logger.Error("handleListenedConn ", err)
		}

		if nil != conn {
			logger.Infof("Address %s: Client closed connection", conn.RemoteAddr())
			ccb(context.TODO(), conn)
			conn.Close()
		}

		return
	}()

	recvCh := make(chan int)
	// 读数据协程
	go func() {
		for {
			_, dataReadError := readFromConnection(conn, dataBuffer[0:maxMessageSize])
			if dataReadError != nil {
				if dataReadError != io.EOF {
					// log the error from the call to read
					logger.Infof("Address %s: Failure to read from connection. Underlying error: %s", conn.RemoteAddr(), dataReadError)
				} else {
					// The client wrote the header but closed the connection
					logger.Infof("Address %s: Client closed connection during data read. Underlying error: %s", conn.RemoteAddr(), dataReadError)
				}

				return
			}

			// 如果读取消息没有错误，就调用回调函数
			err := rcb(context.TODO(), conn, dataBuffer)
			if err != nil {
				logger.Infof("Error in Callback: %s", err)
			}

			recvCh <- 0
			continue
		}
	}()

	// 主逻辑协程
	for {
		select {
		case <-recvCh:

		}
	}
}

// Handles reading from a given connection.
func readFromConnection(reader *net.TCPConn, buffer []byte) (int, error) {
	// This fills the buffer
	bytesLen, err := reader.Read(buffer)
	// Output the content of the bytes to the queue
	if bytesLen == 0 {
		if err != nil && err == io.EOF {
			// "End of individual transmission"
			// We're just done reading from that conn
			return bytesLen, err
		}
	}

	if err != nil {
		//"Underlying network failure?"
		// Not sure what this error would be, but it could exist and i've seen it handled
		// as a general case in other networking code. Following in the footsteps of (greatness|madness)
		return bytesLen, err
	}
	// Read some bytes, return the length
	return bytesLen, nil
}

func WriteToConnections(conn *net.TCPConn, packet []byte) (n int, err error) {
	if 0 == len(packet) {
		return
	}

	conn.Write(packet)
	return
}
