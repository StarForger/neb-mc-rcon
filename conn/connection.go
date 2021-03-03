package conn

import (	
	"github.com/StarForger/neb-rcon/packet"
	"errors"						// manipulate errors	
	"net"								// interface for network I/O
	"sync"							// basic synchronization primitives such as mutual exclusion locks
	"time"							// for measuring and displaying time
	"log"
	"strconv"
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
	request, err := packet.CreateCommandRequest(c.id, cmd)
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

	response, err := packet.CreateCommandResponse(data)
	if err != nil {
		return "", err
	}	

	name, _ := response.GetMetadata()

	if name != "command" || response.GetMethod() != "response" {
		return "", ErrorResponseMismatch
	}

	c.queue = data[response.GetLength():]
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

func (c *Connection) loginReadAttempt() (*packet.Packet, error) {	
	data, err := c.read()
	if err != nil {
		return nil, nil, err
	}

	loginResponse, err := packet.CreateLoginResponse(data)
	if err != nil {
		return nil, nil, err
	}	

	name, _ := loginResponse.GetMetadata()		

	if name != "login" || loginResponse.GetMethod() != "response" {
		return nil, nil, ErrorResponseMismatch
	}	
	
	return loginResponse, nil
}

func (c *Connection) read() ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.conn.SetReadDeadline(time.Now().Add(timeout)) //TODO longer timeout on commands
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

	log.Printf(strconv.Itoa(size))		

	if size < 4 {		
		s, err := c.conn.Read(c.buffer[size:])
		if err != nil {
			return nil, err
		}
		size += s
	}	

	return bytes.NewBuffer(c.buffer[:size])
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


