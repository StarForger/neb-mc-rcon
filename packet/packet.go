package packet

import (
	"bytes"							// manipulation of byte slices
	"encoding/binary"   // translation between numbers and byte sequences
	"errors"						// manipulate errors
	"io"								// basic interfaces to I/O primitives
	"strconv" 					// conversions to and from string
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
	// Payload length max plus packet length minimum
	PacketLengthMax			= 4106 
	// Packet length max plus "length" Int32
	PacketSizeMax				= 4110
	// Two Int32 (requestId and type) plus two bytes (payload terminator and pad)
	PacketLengthMin			= 10 

	typeLoginRequest		= 3
	typeCommandRequest	= 2

	typeLoginResponse		= 2	
	typeCommandResponse	= 0
	typeInvalidResponse	= -1

	payloadRequestMax		= 1024
	payloadResponseMax  = 4096	
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
	ErrorMaxPayloadLength = errors.New("packet: payload length too large")
	ErrorMinPayloadLength = errors.New("packet: payload length too small")
	ErrorMismatchedPayloadLength = errors.New("packet: payload length mismatch")
)

func CreateLoginRequest(password string) (*Packet, error) {
	return createRequest(0, typeLoginRequest, password)
}

func CreateCommandRequest(id int32, body string) (*Packet, error) {
	return createRequest(id, typeCommandRequest, body)
}

func CreateLoginResponse(payload []byte) (*Packet, []byte, error) {
	return createResponse(payload)
}

func CreateCommandResponse(payload []byte) (*Packet, []byte, error) {
	return createResponse(payload)
}

func (p *Packet) GetMetadata() (name string, payloadMax int32, lengthMax int32) {
	name = "unknown"
	payloadMax = 0
	lengthMax = PacketLengthMin
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
		if p.requestType == typeInvalidResponse {
			name = "invalid"
		}
		payloadMax = payloadResponseMax	
	}
	lengthMax = lengthMax + payloadMax
	return
}

func (p *Packet) GetId() (int32) {
	return p.id
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

func (p *Packet) verify() (error) {
	if p.length < PacketLengthMin {
		return ErrorMinPayloadLength
	}
	_, _, lengthMax := p.GetMetadata()
	if p.length > lengthMax {
		return ErrorMaxPayloadLength
	}
	if p.length != PacketLengthMin + int32(len(p.payload)) {
		return ErrorMismatchedPayloadLength
	}
	return nil
}

func (p *Packet) encode() {
	buffer := bytes.NewBuffer(make([]byte, 0, p.length + 4)) // including size of "length"

	// packet size
	binary.Write(buffer, binary.LittleEndian, p.length)

	// request id
	binary.Write(buffer, binary.LittleEndian, p.requestId)

	// type
	binary.Write(buffer, binary.LittleEndian, p.requestType)

	// payload
	buffer.WriteString(p.payload)

	// null terminator
	binary.Write(buffer, binary.LittleEndian, byte(0))

	// pad
	binary.Write(buffer, binary.LittleEndian, byte(0))

	p.encoded = buffer.Bytes()
}

func createRequest(id int32, code int32, body string) (*Packet, error) {
	p := &Packet{
		length: PacketLengthMin + int32(len(body)),
		requestId: createRequestId(id),
		requestType: code,
		payload: body,
		method: "request", 
	}
	if err := p.verify(); err != nil {
		return nil, err
	} 
	p.encode()
	return p, nil
}

func createResponse(data []byte ) (*Packet, []byte, error) {		
	b := bytes.NewBuffer(data)
	var (
		intCheck string
		length int
		requestId int
		requestType int
		payload []byte
		err error
	)

	binary.Read(b, binary.LittleEndian, &intCheck)
	if length, err = strconv.Atoi(intCheck); err != nil {
		return nil, nil, err
	}
	binary.Read(b, binary.LittleEndian, &intCheck)
	if requestId, err = strconv.Atoi(intCheck); err != nil {
		return nil, nil, err
	}
	binary.Read(b, binary.LittleEndian, &intCheck)
	if requestType, err = strconv.Atoi(intCheck); err != nil {
		return nil, nil, err
	}

	if payload, err = b.ReadBytes(0x00); err != nil {
		if err == io.EOF {		
			payload = payload[:len(payload)-1] // remove null terminator
		}
		return nil, nil, err
	}		

	p := &Packet{
		length: int32(length),
		requestId: int32(requestId),
		requestType: int32(requestType),
		payload: string(payload),
		method: "response",
		encoded: data, 
	}
	if err := p.verify(); err != nil {
		return nil, nil, err
	} 
	// remainder should just be null terminator but might be start of next packet	
	if len(data) == 4 + int(p.length)  {
		return p, nil, nil
	}

	return p, data[4 + int(p.length):], nil // remove "length" aswell
} 

func createRequestId(id int32) (int32) {
	// prevent max int overflow
	if id <= 0 || id != id & 0x7fffffff { 
		return int32((time.Now().UnixNano() / 100000) % 100000)
	}
	return id + 1	
}
