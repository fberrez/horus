package lifx

import (
	"encoding/binary"
	"fmt"
)

type (
	// Header is a part of the message which can be sent to a Lifx bulb.
	// It contains a frame, a frame address and a protocol header.
	Header struct {
		frame       [2]byte
		source      [4]byte
		target      [8]byte
		ackRequired bool
		resRequired bool
		sequence    byte
		messageType [2]byte
	}

	MessageType uint16
)

var (
	DefaultFrame         = TAFrame
	Source               = [4]byte{0X00, 0X00, 0X00, 0X00}
	DefaultTarget        = [8]byte{0X00, 0X00, 0X00, 0X00, 0X00, 0X00, 0X00, 0X00}
	DefaultSequence byte = 0X00
	size            uint = 34
)

const (
	GetService        MessageType = 2
	StateService      MessageType = 3
	GetHostInfo       MessageType = 12
	StateHostInfo     MessageType = 13
	GetHostFirmware   MessageType = 14
	StateHostFirmware MessageType = 15
	GetWifiInfo       MessageType = 16
	StateWifiInfo     MessageType = 17
	GetWifiFirmware   MessageType = 18
	StateWifiFirmware MessageType = 19
	GetPowerDevice    MessageType = 20
	SetPowerDevice    MessageType = 21
	StatePower        MessageType = 22
	GetLabel          MessageType = 23
	SetLabel          MessageType = 24
	StateLabel        MessageType = 25
	GetVersion        MessageType = 32
	StateVersion      MessageType = 33
	GetInfo           MessageType = 34
	StateInfo         MessageType = 35
	Acknowledgement   MessageType = 45
	GetLocation       MessageType = 48
	SetLocation       MessageType = 49
	StateLocation     MessageType = 50
	GetGroup          MessageType = 51
	SetGroup          MessageType = 52
	StateGroup        MessageType = 53
	EchoRequest       MessageType = 58
	EchoResponse      MessageType = 59
	Get               MessageType = 101
	SetColor          MessageType = 102
	SetWaveform       MessageType = 103
	GetPowerLight     MessageType = 116
	SetPowerLight     MessageType = 117
)

// NewHeader build a header with given informations.
// It returns a new pointer of Header.
func NewHeader() *Header {
	return &Header{
		frame:       DefaultFrame,
		source:      Source,
		target:      DefaultTarget,
		ackRequired: false,
		resRequired: false,
		sequence:    DefaultSequence,
		messageType: [2]byte{0X00, 0X00},
	}
}

// SetFrame sets the specified frame in the header.
// The frame which is in argument must be written in a little endian format.
func (h *Header) SetFrame(frame [2]byte) *Header {
	h.frame = frame
	return h
}

// SetSource sets the specified source in the header.
// The source which is in argument must be written in a little endian format.
func (h *Header) SetSource(source [4]byte) *Header {
	h.source = source
	return h
}

// SetTarget sets the specified target in the header.
// The target which is in argument must be written in a little endian format.
func (h *Header) SetTarget(target [8]byte) *Header {
	h.target = target
	return h
}

// IsAckRequired sets the ackRequired value in the header.
func (h *Header) IsAckRequired(required bool) *Header {
	h.ackRequired = required
	return h
}

// IsResRequired sets the resRequired value in the header
func (h *Header) IsResRequired(required bool) *Header {
	h.resRequired = required
	return h
}

// SetSequence sets the specified sequence in the header.
// Note that a sequence is only useful if a acknowledge or a response message is required.
// Otherwise, it must be equal to []byte{0X00}.
// The sequence which is in argument must be written in a little endian format.
func (h *Header) SetSequence(sequence byte) *Header {
	h.sequence = sequence
	return h
}

// SetMessageType sets the specified message type in the header.
func (h *Header) SetMessageType(msgType MessageType) *Header {
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(msgType))
	h.messageType = [2]byte{bs[0], bs[1]}
	return h
}

// EncodeToBytes converts a header to an array of bytes.
// This one is written in a big endian format.
func (h *Header) EncodeToBytes() []byte {
	// Adds first part of the header
	buffer := make([]byte, 0, 34)
	buffer = append(buffer, h.frame[0:]...)
	buffer = append(buffer, h.source[0:]...)
	buffer = append(buffer, h.target[0:]...)
	// Adds 6 reserved bytes
	for i := 0; i < 6; i++ {
		buffer = append(buffer, 0X00)
	}

	// Computes ack and res bytes and adds them to the array
	buffer = append(buffer, h.encodeAckResToByte())
	// Adds the Sequence
	buffer = append(buffer, h.sequence)
	// Adds 8 reserved bytes
	for i := 0; i < 8; i++ {
		buffer = append(buffer, 0X00)
	}
	// Adds message type
	buffer = append(buffer, h.messageType[0:]...)
	// Adds 2 reserved bytes
	buffer = append(buffer, 0X00, 0X00)

	// Copies the buffer to a fixed size array
	return buffer[0:34]
}

// ackResBytes converts the ack and res bool present in the header to a byte.
func (h *Header) encodeAckResToByte() byte {
	if !h.ackRequired && !h.resRequired {
		return 0X00
	}
	if !h.ackRequired && h.resRequired {
		return 0X01
	}
	if h.ackRequired && !h.resRequired {
		return 0X10
	}

	return 0X11
}

// decodeAckRes decode a byte to its corresponding value in two bools.
// The first bool is the acknowledgement-required setting
// The second bool is the response-required setting
func decodeAckRes(byte byte) (bool, bool) {
	if byte == 0X00 {
		return false, false
	}
	if byte == 0X01 {
		return false, true
	}
	if byte == 0X10 {
		return true, false
	}

	return true, true
}

// getSize returns the size of the header in bytes.
func (h *Header) getSize() uint {
	return size
}

// String returns the string containing all informations containing in the header.
func (h *Header) String() string {
	pattern := "frame: %v source: %v target: %v ackRequired: %v resRequired: %v sequence: %v messageType: %v"
	return fmt.Sprintf(pattern, h.frame, h.source, h.target, h.ackRequired,
		h.resRequired, h.sequence, h.messageType)
}
