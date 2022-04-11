package conn

import (	
	"errors"						// manipulate errors	
	"net"								// interface for network I/O
	"sync"							// basic synchronization primitives such as mutual exclusion locks
	"time"							// for measuring and displaying time
	// "log"
)

const (
	connTimeout = 10 * time.Second
	readTimeout = 1 * time.Minute
)

type Connection struct {
	id				int32
	conn      net.Conn	
	buffer   	[]byte	
	queue 		[]byte
	lock    	sync.Mutex		
}

var ( 	
	ErrorResponseMismatch = errors.New("connection: response type mismatch")		
)

func Dial(hostUri string, password string) (*Connection, error) {	
	c, err := connect(hostUri)
	if err != nil {
		return nil, err
	}

	loginPacket, err := c.login(password)
	if err != nil {
		return nil, err
	}

	c.id = loginPacket.GetId()

	return c, nil
}

func (c *Connection) Execute(cmd string) (string, error) {	
	request, err := CreateCommandRequest(c.id, cmd)
	if err != nil {
		return "", err
	}	

	_, err = c.conn.Write(request.GetEncoded())
	if err != nil {
		return "", err
	}

	data, err := c.read()
	if err != nil {
		return "", err
	}

	response, err := CreateCommandResponse(data)
	if err != nil {
		return "", err
	}	

	name, _ := response.GetMetadata()

	if name != "command" || response.GetMethod() != "response" {
		return "", ErrorResponseMismatch
	}

	c.queue = data[response.GetLength() + 4:] // include length
	c.id = response.GetId()

	return response.GetPayload(), nil	
}	

func (c *Connection) Close() (error) {
	return c.conn.Close()
}

func (c *Connection) login(password string) (*Packet, error) {

	loginRequest, err := CreateLoginRequest(password)
	if err != nil {
		return nil, err
	}	

	_, err = c.conn.Write(loginRequest.GetEncoded())
	if err != nil {
		return nil, err
	}

	loginResponse, err := c.loginReadAttempt()
	// Retry authentication once (RCON bug)	
	if err == ErrorResponseMismatch {
		loginResponse, err = c.loginReadAttempt()
	}
	if err != nil {
		return nil, err
	}

	return loginResponse, nil
}

func (c *Connection) loginReadAttempt() (*Packet, error) {	

	data, err := c.read()
	if err != nil {
		return nil, err
	}

	loginResponse, err := CreateLoginResponse(data)
	if err != nil {
		return nil, err
	}	

	name, _ := loginResponse.GetMetadata()		

	if name != "login" || loginResponse.GetMethod() != "response" {
		return nil, ErrorResponseMismatch
	}	
	
	return loginResponse, nil
}

func (c *Connection) read() ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	var size int
	var err error
	if c.queue != nil {
		copy(c.buffer, c.queue)
		size = len(c.queue)
		c.queue = nil
	} else {
		size, err = c.conn.Read(c.buffer)
		if err != nil {
			return nil, err
		}
	}		

	if size < 4 {		
		s, err := c.conn.Read(c.buffer[size:])
		if err != nil {
			return nil, err
		}
		size += s
	}	

	return c.buffer[:size], nil
}

func connect(hostUri string) (*Connection, error)  {	
	conn, err := net.DialTimeout("tcp", hostUri, connTimeout)
	if err != nil {
		return nil, err
	}
	c := &Connection{
		conn: conn,
		buffer: make([]byte, SizeMax),
	}
	return c, nil
}


