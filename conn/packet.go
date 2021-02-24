package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv" // conversions to and from string
	"time"
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
// ------------------------	TOTAL					>=14
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
// Pad is null
//
// NB: Little endian integers
// NB: Request payload max size is unreliable at 1446. Should be reliable at 1024
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
	packetLengthMin			= 10 // not including length

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
	requestType int32
	payload			string // parsed to []byte at send
	// Pad is added at end of buffer	
	method			string  // for differentiating requestType codes	
	encoded			[]byte  // entire packet converted to binary	
}

var ( 
	ErrorMaxPayloadLength = errors.New("packet: payload length too large")
	ErrorMinPayloadLength = errors.New("packet: payload length too small")
	ErrorMismatchedPayloadLength = errors.New("packet: payload length mismatch")
)

func CreateLoginRequest(string password) (*Packet, error) {
	return createRequest(0, typeLoginRequest, password)
}

func CreateCommandRequest(int32 id, string body) (*Packet, error) {
	return createRequest(id, typeCommandRequest, body)
}

func CreateLoginResponse([]byte payload) (*Packet, []byte, error) {
	return createResponse(typeLoginResponse, payload)
}

func CreateCommandResponse([]byte payload) (*Packet, []byte, error) {
	return createResponse(typeCommandResponse, body)
}

func (p *Packet) GetMetadata() (string name, int32 payloadMax, int32 lengthMax) {
	name = "unknown"
	payloadMax = 0
	lengthMax = packetTotalMin
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

func (p *Packet) GetLengthMax() {
	_, _, lengthMax := p.getMetadata()
	return lengthMax 
}

func (p *Packet) verify() (error) {
	if p.length < packetTotalMin {
		return ErrorMinPayloadLength
	}
	if p.length > p.GetLengthMax() {
		return ErrorMaxPayloadLength
	}
	if p.length != len(p.payload) + packetLengthMin {
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

func createRequest(int32 id, int32 code, string body) (*Packet, error) {
	p := &Packet{
		length: packetLengthMin + len(body),
		requestId: createRequestId(id),
		requestType: code,
		payload: body,
		method: "request" 
	}
	err := p.verify(); err != nil {
		return nil, err
	} 
	p.encode()
	return p, nil
}

func createResponse([]byte data) (*Packet, []byte, error) {		
	b := bytes.NewBuffer(data)
	var intCheck string

	binary.Read(b, binary.LittleEndian, &intCheck)
	length, err := strconv.Atoi(*intCheck); err != nil {
		return nil, nil, err
	}
	binary.Read(b, binary.LittleEndian, &intCheck)
	requestId, err := strconv.Atoi(*intCheck); err != nil {
		return nil, nil, err
	}
	binary.Read(b, binary.LittleEndian, &intCheck)
	requestType, err := strconv.Atoi(*intCheck); err != nil {
		return nil, nil, err
	}

	payload, err := b.ReadBytes(0x00)
	if err != nil {
		if err == io.EOF {		
			payload = payload[:len(payload)-1] // remove null terminator
		}
		return nil, nil, err
	}		

	p := &Packet{
		length: int32(length),
		requestId: int32(requestId),
		requestType: int32(requestType),
		payload: int32(payload),
		method: "response",
		encoded: data 
	}
	err := p.verify(); err != nil {
		return nil, nil, err
	} 
	// remainder should just be null terminator but might be start of next packet	
	if len(b) == 4 + p.length  {
		return p, nil, nil
	}

	return p, data[4 + p.length:], nil // remove "length" aswell
} 

func createRequestId(int32 id) (int32) {
	if id <= 0 || id != id & 0x7fffffff { // prevent max int overflow
		return int32((time.Now().UnixNano() / 100000) % 100000)
	}
	else {
		return = id + 1
	}	
}
