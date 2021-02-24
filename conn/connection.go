package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	
	"time"
)

// Max package length plus "length" size (int32).
const maxBufferSize = 4110

const timeoutSeconds = 10

type Connection struct {
	id				int32
	conn      net.Conn	
	queue 		[]byte
	buffer   	[]byte
	lock    	sync.Mutex	
}

var ( 	
	ErrorPayloadRead = errors.New("connection: payload response can't be read")
	ErrorResponseMismatch = errors.New("connection: response type mismatch")
	ErrorIDMismatch = errors.New("connection: response/request id mismatch")
	ErrorPassword = errors.New("connection: password incorrect")
	ErrorUnknown = errors.New("connection: unknown response")	
)

func Dial(string host, string password) (*Connection, error) {
	c, err := connect(host); err != nil {
		return nil, err
	}
	
	loginPacket, err := c.login(password); err != nil {
		return nil, err
	}

	c.id = loginPacket.requestId

	return c, nil
}

// TODO improve, reuse login
func (c *Connection) Execute(string cmd) (string, error) {
	if request, err := CreateCommandRequest(c.id, cmd); err != nil {
		return nil, err
	}	

	if _, err := c.conn.Write(request.encoded); err != nil {
		return nil, err
	}

	if err := c.read(); err != nil {
		return nil, err
	}

	if response, overflow, err := CreateCommandResponse(c.readbuf); err != nil {
		return nil, err
	}	

	if name, _, _ := response.GetMetadata(); name == "unknown" {		
		return nil, ErrorUnknown	
	}	

	if name != "command" || p.method != "response" {
		return ErrorResponseMismatch
	}

	if response.requestId != request.requestId {
		return nil, ErrorIDMismatch
	}

	c.queuedbuf = overflow
	c.id = response.requestId

	return response.payload, nil	
}	

func (c *Connection) Close() (error) {
	return c.conn.Close()
}

func connect(string host) (*Connection)  {
	const timeout = timeoutSeconds * time.Second
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, err
	}
	c := &Connection{
		conn: conn,
		readbuf: make([]byte, maxBufferSize)
	}
	 return c, nil
}

func (c *Connection) login(string password) (*Packet, error) {
	if loginRequest, err := CreateLoginRequest(password); err != nil {
		return nil, err
	}	

	if _, err := c.conn.Write(loginRequest.encoded); err != nil {
		return nil, err
	}

	if loginResponse, overflow, err := c.loginReadAttempt(); err != nil {
		return nil, err
	}	

	if name, _, _ := loginResponse.GetMetadata(); name == "unknown" {
		// retry reading on first error. sometimes RCON protocol bugs out.
		if loginResponse, overflow, err := c.loginReadAttempt(); err != nil {
			return nil, err
		}	
		if name, _, _ := loginResponse.GetMetadata(); name == "unknown" {
			return nil, ErrorUnknown
		}
	}

	if name == "invalid" {
		return nil, ErrorPassword
	}		

	if name != "login" || p.method != "response" {
		return ErrorResponseMismatch
	}

	if loginResponse.requestId != loginRequest.requestId {
		return nil, ErrorIDMismatch
	}

	c.queuedbuf = overflow

	return loginResponse, nil
}

func (c *Connection) loginReadAttempt() (*Packet, []byte, error) {
	if err := c.read(); err != nil {
		return nil, nil, err
	}

	return CreateLoginResponse(c.readbuf)		
}

func (c *Connection) read(timeout time.Duration) (error) {
	c.readmu.Lock()
	defer c.readmu.Unlock()

	c.conn.SetReadDeadline(time.Now().Add(timeout))
	var size int
	var err error
	if c.queuedbuf != nil {
		copy(c.readbuf, c.queuedbuf)
		size = len(c.queuedbuf)
		c.queuedbuf = nil
	} 
	else if size, err = c.conn.Read(c.readbuf); err != nil {
		return err
	}

	// verify 4 byte length
	if size < 4 {		
		if s, err := r.conn.Read(c.readbuf[size:]); err != nil {
			return err
		}  
		size += s
	}	

	if size != 4 {
		return ErrorPayloadRead
	}

	return nil
}


