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

	c.id = loginPacket.GetId()

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

	return response.GetPayload(), nil	
}	

func (c *Connection) Close() (error) {
	return c.conn.Close()
}

func (c *Connection) login(password string) (*packet.Packet, error) {
	loginRequest, err := packet.CreateLoginRequest(password)
	if err != nil {
		return nil, err
	}	

	_, err = c.conn.Write(loginRequest.GetEncoded())
	if err != nil {
		return nil, err
	}

	loginResponse, overflow, err := c.loginReadAttempt()
	// Retry authentication once (RCON bug)	
	if err == ErrorUnknown {
		loginResponse, overflow, err = c.loginReadAttempt()
	}
	if err != nil {
		return nil, err
	}

	if loginResponse.GetId() != loginRequest.GetId() {
		return nil, ErrorIDMismatch
	}

	c.queue = overflow

	return loginResponse, nil
}

func (c *Connection) loginReadAttempt() (*packet.Packet, []byte, error) {	
	if err := c.read(); err != nil {
		return nil, nil, err
	}

	loginResponse, overflow, err := packet.CreateLoginResponse(c.buffer)
	if err != nil {
		return nil, nil, err
	}	

	name, _, _ := loginResponse.GetMetadata()

	if name == "unknown" {
		return nil, nil, ErrorUnknown
	}

	if name == "invalid" {
		return nil, nil, ErrorPassword
	}		

	if name != "login" || loginResponse.GetMethod() != "response" {
		return nil, nil, ErrorResponseMismatch
	}	
	
	return loginResponse, overflow, nil
}

func (c *Connection) read() (error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.conn.SetReadDeadline(time.Now().Add(timeout))
	var size int
	var err error
	if c.queue != nil {
		copy(c.buffer, c.queue)
		size = len(c.queue)
		c.queue = nil
	} else if size, err = c.conn.Read(c.buffer); err != nil {
		return err
	}

	if size < 4 {		
		s, err := c.conn.Read(c.buffer[size:])
		if err != nil {
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
		buffer: make([]byte, packet.PacketSizeMax),
	}
	 return c, nil
}


