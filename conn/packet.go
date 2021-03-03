package conn

import (
	"bytes"							// manipulation of byte slices
	"encoding/binary"   // translation between numbers and byte sequences
	"errors"						// manipulate errors
	"io"								// basic interfaces to I/O primitives
	"time"							// for measuring and displaying time
)

// From https://wiki.vg/RCON
// ######## PACKET ########
//
// Format:
// ----	NAME								TYPE					SIZE (bytes)
// ---- Length							int32					4			
// ---- Request ID					int32					4
// ----	Type								int32					4
// ---- Payload						  []byte				>=1 (null terminated)
// ---- Pad									byte					1
//
// Length is the the total size of the packet not including the length itself.
//
// Request ID is client generated.
// 
// Type:
// ----	NAME								REQUEST				RESPONSE
// ---- Login								3							2
// ---- Command							2							0
//
// Payload:
// ---- NAME								MAX SIZE (bytes)
// ---- Request							1446/1024 (see note) 
// ---- Response						4096															
//
// Pad is null and added during encode
//
// NB: Little endian integers
// NB: Request payload max size is unreliable at 1446. Should be reliable at 1024
// NB: Packet length (without "length" itself) minimum is 10 (4 + 4 + 1 + 1)
// NB: Packet length maximum is the payload max plus the packet minimum (4106)
//
// ######## RESPONSE ########
// 
// Response Request ID:
// ----	DESCRIPTION												REQUEST ID
// ---- Authorised/Password correct				As request
// ---- Unauthorised/Password incorrect		-1
//
// With fragmentation, the final packet can be determined by:
// ---- Packet length < 4096
// ---- Wait x seconds
// ---- Send request with different Request ID, wait for same Request ID
// ---- Send request with different Request ID and invalid type (x), wait for response with payload: 'Unknown request x'
//

const (	
	// Two Int32 (requestId and type) plus two bytes (payload terminator and pad)
	LengthMin							= 10 
	// Payload max plus packet length minimum
	LengthMax							= 4106 
	// Packet length max plus "length" Int32
	SizeMax								= 4110

	idInvalid							=	-1

	typeLoginRequest			= 3
	typeCommandRequest		= 2
	typeLoginResponse			= 2	
	typeCommandResponse		= 0	

	payloadRequestMax			= 1024
	payloadResponseMax  	= 4096	
)

type Packet struct {
	length			int32
	requestId		int32
	requestType int32   // type named requestType
	payload			string 	// parsed to []byte at encode 
	method			string  // for differentiating requestType codes	
	encoded			[]byte  // entire packet encoded to binary	
}

var ( 
	ErrorMaxLength 								= errors.New("packet: length too large")
	ErrorMinLength 								= errors.New("packet: length too small")
	ErrorMismatchType							= errors.New("packet: type mismatch")
	ErrorInvalidId								= errors.New("packet: unauthorised/incorrect password")
	ErrorMismatchedPayloadLength 	= errors.New("packet: payload length mismatch")
	ErrorUnknown 									= errors.New("packet: unknown type")	
)

func CreateLoginRequest(password string) (*Packet, error) {
	return createRequest(0, typeLoginRequest, password)
}

func CreateCommandRequest(id int32, body string) (*Packet, error) {
	return createRequest(id, typeCommandRequest, body)
}

func CreateLoginResponse(payload []byte) (*Packet, error) {
	return createResponse(typeLoginResponse, payload)
}

func CreateCommandResponse(payload []byte) (*Packet, error) {
	return createResponse(typeCommandResponse, payload)
}

func (p *Packet) GetMetadata() (name string, payloadMax int32) {
	name = "unknown"
	payloadMax = 0
	switch p.method {
	case "request": 
		if p.requestType == typeLoginRequest {
			name = "login"
		}
		if p.requestType == typeCommandRequest {
			name = "command"
		}
		payloadMax = payloadRequestMax		
	case "response":
		if p.requestType == typeLoginResponse {
			name = "login"
		}
		if p.requestType == typeCommandResponse {
			name = "command"
		}
		payloadMax = payloadResponseMax	
	}
	return
}

func (p *Packet) GetLength() (int32) {
	return p.length
}

func (p *Packet) GetId() (int32) {
	return p.requestId
}

func (p *Packet) GetMethod() (string) {
	return p.method
}

func (p *Packet) GetPayload() (string) {
	return p.payload
}

func (p *Packet) GetEncoded() ([]byte) {
	return p.encoded
}

func (p *Packet) verify(code int32) (error) {
	_, payloadMax := p.GetMetadata()

	if p.length < LengthMin {
		return ErrorMinLength
	}
	
	if p.length > payloadMax + LengthMin {
		return ErrorMaxLength
	}

	if p.requestId == idInvalid {
		return ErrorInvalidId
	}

	if p.requestType != code {
		return ErrorMismatchType
	}
	
	if len(p.payload) != int(p.length) - LengthMin {
		return ErrorMismatchedPayloadLength
	}

	return nil
}

func (p *Packet) encode() (error) {
	// make buffer including size of "length"
	buffer := bytes.NewBuffer(make([]byte, 0, p.length + 4)) 

	// packet size
	if err := binary.Write(buffer, binary.LittleEndian, p.length); err != nil {
		return err
	}

	// request id
	if err := binary.Write(buffer, binary.LittleEndian, p.requestId); err != nil {
		return err
	}

	// type
	if err := binary.Write(buffer, binary.LittleEndian, p.requestType); err != nil {
		return err
	}

	// payload
	buffer.WriteString(p.payload)

	// null terminator
	if err := binary.Write(buffer, binary.LittleEndian, byte(0)); err != nil {
		return err
	}

	// pad
	if err := binary.Write(buffer, binary.LittleEndian, byte(0)); err != nil {
		return err
	}

	// assign to encoded
	p.encoded = buffer.Bytes()

	return nil
}

func (p *Packet) decode(data []byte) (error) {
	// make buffer from encoded data
	buffer := bytes.NewBuffer(data)
	
	// packet size
	if err := binary.Read(buffer, binary.LittleEndian, &p.length); err != nil && err != io.EOF {
		return err
	}

	// request id
	if err := binary.Read(buffer, binary.LittleEndian, &p.requestId); err != nil && err != io.EOF {
		return err
	}

	// type
	if err := binary.Read(buffer, binary.LittleEndian, &p.requestType); err != nil && err != io.EOF {
		return err
	}

	// payload
	payload, err := buffer.ReadBytes(0x00)
	if err != io.EOF {
		if err != nil {
			return err
		}
		payload = payload[:len(payload)-1] // remove null terminator		
	} 
	
	p.payload = string(payload)
	p.encoded = data[:4 + int(p.length)]

	return nil
}

func createRequest(id int32, code int32, body string) (*Packet, error) {
	p := &Packet{
		length: LengthMin + int32(len(body)),
		requestId: createRequestId(id),
		requestType: code,
		payload: body,
		method: "request", 
	}

	if err := p.encode(); err != nil{
		return nil, err
	}

	if err := p.verify(code); err != nil {
		return nil, err
	}	

	return p, nil
}

func createResponse(code int32, data []byte) (*Packet, error) {		
	p := &Packet{
		method: "response",
	}

	if err := p.decode(data); err != nil{
		return nil, err
	}	

	if err := p.verify(code); err != nil {
		return nil, err
	} 	

	return p, nil
} 

func createRequestId(id int32) (int32) {
	// prevent max int overflow
	if id <= 0 || id != id & 0x7fffffff { 
		return int32((time.Now().UnixNano() / 100000) % 100000)
	}
	return id + 1	
}
