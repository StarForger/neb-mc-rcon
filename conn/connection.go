package conn

import (	
	"github.com/StarForger/neb-rcon/packet"
	"errors"						// manipulate errors	
	"net"								// interface for network I/O
	"sync"							// basic synchronization primitives such as mutual exclusion locks
	"time"							// for measuring and displaying time
)

// Timeout 10 seconds
const timeout = 10 * time.Second

type Connection struct {
	id				int32
	conn      net.Conn	
	buffer   	[]byte	
	queue 		[]byte
	lock    	sync.Mutex		
}

var ( 	
	ErrorPayloadRead 				= errors.New("connection: payload response can't be read")
	ErrorResponseMismatch 	= errors.New("connection: response type mismatch")
	ErrorIDMismatch 				= errors.New("connection: response/request id mismatch")
	ErrorPassword 					= errors.New("connection: password incorrect")
	ErrorUnknown 						= errors.New("connection: unknown response")	
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

	c.id = loginPacket.requestId

	return c, nil
}

// TODO improve, reuse login
func (c *Connection) Execute(cmd string) (string, error) {	
	request, err := packet.CreateCommandRequest(c.id, cmd)
	if err != nil {
		return "", err
	}	
	_, err = c.conn.Write(request.GetEncoded())
	if err != nil {
		return "", err
	}

	if err := c.read(); err != nil {
		return "", err
	}

	response, overflow, err := packet.CreateCommandResponse(c.buffer)
	if err != nil {
		return "", err
	}	

	name, _, _ := response.GetMetadata()
	if name == "unknown" {		
		return "", ErrorUnknown	
	}	

	if name != "command" || response.GetMethod() != "response" {
		return "", ErrorResponseMismatch
	}

	if response.GetId() != request.GetId() {
		return "", ErrorIDMismatch
	}

	c.queue = overflow
	c.id = response.GetId()

	return response.GetPayload, nil	
}	

func (c *Connection) Close() (error) {
	return c.conn.Close()
}

func (c *Connection) login(password string) (*Packet, error) {
	loginRequest, err := packet.CreateLoginRequest(password)
	if err != nil {
		return nil, err
	}	

	_, err := c.conn.Write(loginRequest.encoded)
	if err != nil {
		return nil, err
	}

	loginResponse, overflow, err := c.loginReadAttempt()
	if err != nil {
		return nil, err
	}	

	name, _, _ := loginResponse.GetMetadata()
	if name == "unknown" {
		// retry reading on first error. sometimes RCON protocol bugs out.
		if loginResponse, overflow, err := c.loginReadAttempt(); err != nil {
			return nil, err
		}	
		if name, _, _ = loginResponse.GetMetadata(); name == "unknown" {
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

	return packet.CreateLoginResponse(c.buffer)		
}

func (c *Connection) read() (error) {
	c.readmu.Lock()
	defer c.readmu.Unlock()

	c.conn.SetReadDeadline(time.Now().Add(timeout))
	var size int
	var err error
	if c.queuedbuf != nil {
		copy(c.readbuf, c.queuedbuf)
		size = len(c.queuedbuf)
		c.queuedbuf = nil
	} else if size, err = c.conn.Read(c.readbuf); err != nil {
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

func connect(hostUri string) (*Connection, error)  {	
	conn, err := net.DialTimeout("tcp", hostUri, timeout)
	if err != nil {
		return nil, err
	}
	c := &Connection{
		conn: conn,
		readbuf: make([]byte, Packet.PacketSizeMax),
	}
	 return c, nil
}


