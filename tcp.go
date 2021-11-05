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

type ListenCallback func(context.Context, *net.TCPListener) error
type RecvCallback func(context.Context, *net.TCPConn, []byte) error
type CloseCallback func(context.Context, *net.TCPConn) error
type TCPListener struct {
	socket          *net.TCPListener
	address         string
	headerByteSize  int
	maxMessageSize  int
	enableLogging   bool
	listenCallback  ListenCallback
	recvcallback    RecvCallback
	closeCallback   CloseCallback
	shutdownChannel chan struct{}
	shutdownGroup   *sync.WaitGroup
}

type TCPListenerConfig struct {
	MaxMessageSize int
	EnableLogging  bool
	Address        string
	listenCallback ListenCallback
	recvcallback   RecvCallback
	closeCallback  CloseCallback
}

func ListenTCP(cfg TCPListenerConfig) (*TCPListener, error) {
	maxMessageSize := DefaultMaxMessageSize
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	btl := &TCPListener{
		enableLogging:   cfg.EnableLogging,
		maxMessageSize:  maxMessageSize,
		headerByteSize:  4, // 4byte(int32)
		listenCallback:  cfg.listenCallback,
		recvcallback:    cfg.recvcallback,
		closeCallback:   cfg.closeCallback,
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
			if btl.enableLogging {
				logger.Error("[blockListen] Error attempting to accept connection: ", err)
			}
			select {
			case <-btl.shutdownChannel:
				return nil
			default:
			}
		} else {
			go handleListenedConn(btl.address, conn, btl.headerByteSize, btl.maxMessageSize, btl.enableLogging, btl.recvcallback, btl.shutdownChannel, btl.shutdownGroup)
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
	btl.listenCallback(context.TODO(), receiveSocket)
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

func handleListenedConn(address string, conn *net.TCPConn, headerByteSize int, maxMessageSize int, enableLogging bool, cb RecvCallback, sdChan <-chan struct{}, sdGroup *sync.WaitGroup) {
	sdGroup.Add(1)
	defer sdGroup.Done()
	headerBuffer := make([]byte, headerByteSize)
	dataBuffer := make([]byte, maxMessageSize)
	go func(c *net.TCPConn, s <-chan struct{}) {
		<-s
		c.Close()
	}(conn, sdChan)

	for {
		var headerReadError error
		var totalHeaderBytesRead = 0
		var bytesRead = 0
		// First, read the number of bytes required to determine the message length
		for totalHeaderBytesRead < headerByteSize && headerReadError == nil {
			// While we haven't read enough yet, pass in the slice that represents where we are in the buffer
			bytesRead, headerReadError = readFromConnection(conn, headerBuffer[totalHeaderBytesRead:])
			totalHeaderBytesRead += bytesRead
		}
		if headerReadError != nil {
			if enableLogging {
				if headerReadError != io.EOF {
					// Log the error we got from the call to read
					logger.Infof("Error when trying to read from address %s. Tried to read %d, actually read %d. Underlying error: %s", address, headerByteSize, totalHeaderBytesRead, headerReadError)
				} else {
					// Client closed the conn
					logger.Infof("Address %s: Client closed connection during header read. Underlying error: %s", address, headerReadError)
				}
			}
			conn.Close()
			return
		}
		// Now turn that buffer of bytes into an integer - represnts size of message body
		msgLength, bytesParsed := byteArrayToUInt32(headerBuffer)
		iMsgLength := int(msgLength)
		// Not sure what the correct way to handle these errors are. For now, bomb out
		if bytesParsed == 0 {
			// "Buffer too small"
			if enableLogging {
				logger.Infof("Address %s: 0 Bytes parsed from header. Underlying error: %s", address, headerReadError)
			}
			conn.Close()
			return
		} else if bytesParsed < 0 {
			// "Buffer overflow"
			if enableLogging {
				logger.Infof("Address %s: Buffer Less than zero bytes parsed from header. Underlying error: %s", address, headerReadError)
			}
			conn.Close()
			return
		}
		var dataReadError error
		var totalDataBytesRead = 0
		bytesRead = 0

		// 读取消息，直到满足消息长度
		for totalDataBytesRead < iMsgLength && dataReadError == nil {
			bytesRead, dataReadError = readFromConnection(conn, dataBuffer[totalDataBytesRead:iMsgLength])
			totalDataBytesRead += bytesRead
		}

		if dataReadError != nil {
			if enableLogging {
				if dataReadError != io.EOF {
					// log the error from the call to read
					logger.Infof("Address %s: Failure to read from connection. Was told to read %d by the header, actually read %d. Underlying error: %s", address, msgLength, totalDataBytesRead, dataReadError)
				} else {
					// The client wrote the header but closed the connection
					logger.Infof("Address %s: Client closed connection during data read. Underlying error: %s", address, dataReadError)
				}
			}

			conn.Close()
			return
		}

		// 如果读取消息没有错误，就调用回调函数
		if totalDataBytesRead > 0 && (dataReadError == nil || (dataReadError != nil && dataReadError == io.EOF)) {
			err := cb(context.TODO(), conn, dataBuffer[:iMsgLength])
			if err != nil && enableLogging {
				logger.Infof("Error in Callback: %s", err)
			}
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

	msgLenHeader := intToByteArray(int64(len(packet)), 4)
	toWrite := append(msgLenHeader, packet...)

	toWriteLen := len(toWrite)
	var writeError error
	var totalBytesWritten = 0
	var bytesWritten = 0
	for totalBytesWritten < toWriteLen && writeError == nil {
		bytesWritten, writeError = conn.Write(toWrite[totalBytesWritten:])
		totalBytesWritten += bytesWritten
	}

	err = writeError
	n = totalBytesWritten
	return
}
