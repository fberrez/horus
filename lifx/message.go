package lifx

import (
	"encoding/binary"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Message contains all required datas of a UDP packet for a LIFX device.
type Message struct {
	// Size is the size of the message in uint16.
	Size [2]byte

	// Header is the header of the message.
	Header *Header

	// payload is the payload of the message.
	payload []byte
}

// NewMessage returns a new Message struct.
func NewMessage() *Message {
	return &Message{
		Size:    [2]byte{0X00, 0X00},
		Header:  NewHeader(),
		payload: []byte{},
	}
}

// SetColorMessage returns a SetColor (102) message
// with parsed data, given in arguments.
func SetColorMessage(hsbk *HSBK, duration uint32) *Message {
	// Encodes HSBK in bytes
	hsbkBytes := encodeHSBKToBytes(hsbk)
	// Encodes duration in bytes
	durationBytes := encodeDurationToBytes(duration)

	// Encode payload
	payload := append([]byte{}, 0X00)
	payload = append(payload, hsbkBytes...)
	payload = append(payload, durationBytes...)
	message := NewMessage().SetPayload(payload)

	// Defines Header
	message.Header.SetMessageType(SetColor).IsResRequired(true).SetSequence(0X10).SetFrame(TAFrame)

	return message
}

// GetMessageWithoutPayload returns a message with a given msgType.
// Note: this message does nnot have any payload (its payload is an empty array of bytes).
func GetMessageWithoutPayload(msgType MessageType) *Message {
	message := NewMessage()
	// Defines header
	message.Header.SetMessageType(msgType).IsResRequired(true).SetFrame(TAFrame)
	message.Header.SetSequence(0X10)

	return message
}

// SetPowerDeviceMessage returns a SetPowerDevice (21) message with the given power status.
func SetPowerDeviceMessage(power Power) *Message {
	message := NewMessage()
	// Defines header
	message.Header.SetMessageType(SetPowerDevice).IsResRequired(true).SetFrame(TAFrame)
	message.Header.SetSequence(0X10)

	// If power is "on", the payload sets the power level on 65535.
	// Else, it sets the power level on 0.
	if power == PowerOn {
		message.SetPayload([]byte{0XFF, 0XFF})
	} else {
		message.SetPayload([]byte{0X00, 0X00})
	}

	return message
}

// SetLabelMessage returns a SetLabel (24) message with the given label.
func SetLabelMessage(label string) *Message {
	message := NewMessage()

	// Defines header
	message.Header.SetMessageType(SetLabel).IsResRequired(true).SetFrame(TAFrame)
	message.Header.SetSequence(0X10)

	payload := [32]byte{}
	copy(payload[:], label)
	message.SetPayload(payload[:])

	return message
}

// EncodeToBytes converts a message to an array of bytes.
func (m *Message) EncodeToBytes() []byte {
	// Updates the size of the message
	m.updateSize()
	// Builds the message by joining differents part of the message in an array of bytes.
	result := append([]byte{}, m.Size[0:]...)
	headerBytes := m.Header.EncodeToBytes()
	result = append(result, headerBytes[0:]...)
	result = append(result, m.payload...)

	return result
}

// updateSize updates the size of the message.
func (m *Message) updateSize() int {
	size := uint16(len(m.Size)) + uint16(m.Header.getSize()) + uint16(len(m.payload))
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, size)
	m.Size = [2]byte{buf[0], buf[1]}

	return int(size)
}

// DecodeToMessage decodes a array of bytes
// and converts the result to a new Message.
func DecodeToMessage(bytes []byte) *Message {
	log.WithFields(log.Fields{
		"from":  "lifx.DecodeToMessage",
		"bytes": fmt.Sprintf("% 02X", bytes),
	}).Debug("Decoding a Message")

	m := &Message{
		Header: &Header{},
	}

	// Decodes size
	copy(m.Size[:], bytes[0:2])

	// Decodes header
	copy(m.Header.frame[:], bytes[2:4])
	copy(m.Header.source[:], bytes[4:8])
	copy(m.Header.target[:], bytes[8:16])
	m.Header.ackRequired, m.Header.resRequired = decodeAckRes(bytes[16])
	m.Header.sequence = bytes[17]
	copy(m.Header.messageType[:], bytes[18:19])

	// Decodes payload
	copy(m.payload, bytes[19:])

	return m
}

// SetPayload sets the specified payload in the message.
func (m *Message) SetPayload(payload []byte) *Message {
	m.payload = payload
	return m
}

func encodeHSBKToBytes(hsbk *HSBK) []byte {
	hue := make([]byte, 2)
	saturation := make([]byte, 2)
	brightness := make([]byte, 2)
	kelvin := make([]byte, 2)
	binary.LittleEndian.PutUint16(hue, hsbk.Hue)
	binary.LittleEndian.PutUint16(saturation, hsbk.Saturation)
	binary.LittleEndian.PutUint16(brightness, hsbk.Brightness)
	binary.LittleEndian.PutUint16(kelvin, hsbk.Kelvin)

	result := append([]byte{}, hue...)
	result = append(result, saturation...)
	result = append(result, brightness...)
	result = append(result, kelvin...)

	return result
}

func encodeDurationToBytes(duration uint32) []byte {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, duration)

	return result
}
