package lifx

import (
	"encoding/binary"
	"io/ioutil"
	"net"
	"os"

	"github.com/fberrez/horus/client"
	"github.com/fberrez/horus/client/udp"
	"github.com/fberrez/horus/tools"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type (
	// Lifx contains all informations of a LIFX Device.
	Lifx struct {
		// UUID is the UUID of the device.
		UUID string `yaml:"uuid" json:"uuid"`

		// Label is the label of the device;
		Label string `yaml:"label" json:"label"`

		// Connected is the connection status of the device.
		Connected bool `yaml:"connected" json:"connected"`

		// Power is the power status of the device.
		Power Power `yaml:"power" json:"power"`

		// HSBK is the HSBK value of the device.
		HSBK *HSBK `yaml:"hsbk" json:"hsbk"`

		// Infrared is the infrared value of the device.
		Infrared float32 `yaml:"infrared" json:"infrared"`

		// Group contains informations about the group of the device
		Group *Group `yaml:"group" json:"group"`

		// Product contains informations about the product
		Product *Product `yaml:"product" json:"product"`

		// Info contains informations about the time stats of the device.
		Info *Info `yaml:"info" json:"info"`

		// Location contains informations about the location of the device.
		Location *Location `yaml:"location" json:"location"`

		// Address is IPV4 address of the device.
		Address *net.IP `yaml:"address" json:"address"`

		// Port is the port of the device.
		Port string `yaml:"port" json:"port"`

		// Protocol is the network protocol used to communicate with the device (ex: UDP)
		Protocol client.Protocol `yaml:"protocol" json:"protocol"`

		// client is the network client used to send packets to the device.
		client client.Client
	}

	// Capabilities contains the capabilities informations of a product.
	Capabilities struct {
		// HasColor determines if the product has more than 1 color (White).
		HasColor bool `yaml:"hasColor" json:"hasColor"`

		// HasIR determines if the product can be used as a infrared light.
		HasIR bool `yaml:"hasIR" json:"hasIR"`

		// HasMultiZone determines if the product has the 'multizone' capability.
		HasMultiZone bool `yaml:"hasMultiZone" json:"hasMultiZone"`
	}

	// Product contains all informations about the product.
	Product struct {
		// ID is the ID of the product.
		ID uint32 `yaml:"id" json:"id"`

		// Name is the name of the product.
		Name string `yaml:"name" json:"name"`

		// Vendor is the vendor of the product.
		Vendor string `yaml:"vendor" json:"vendor"`

		// Version is the version of the product.
		Version uint32 `yaml:"version" json:"version"`

		// Capabilities contains all informations about the product capabilities.
		Capabilities *Capabilities `yaml:"capabilities" json:"capabilities"`
	}

	// Group contains all informations about the group of a product.
	Group struct {
		// ID is the ID of the group.
		ID [16]byte `yaml:"id" json:"id"`

		// Label is the name of the group.
		Label string `yaml:"label" json:"label"`
	}

	// Location contains all informations about the location of a product.
	Location struct {
		// ID is the ID of the location
		ID [16]byte `yaml:"id" json:"id"`

		// Label is the name of the location
		Label string `yaml:"name" json:"name"`
	}

	// Info contains all informations about the time stats of a product.
	Info struct {
		// Time is the current time in UNIX format
		Time uint16 `yaml:"time" json:"time"`

		// UpTime is the uptime of the product.
		UpTime uint16 `yaml:"upTime" json:"upTime"`

		// DownTime is the downtime of the product.
		DownTime uint16 `yaml:"downTime" json:"downTime"`
	}

	// State contains all returned informations by a Get (101) message.
	State struct {
		// HSBK is the current HSBK of the light
		HSBK *HSBK

		// Power is the current power level of the light
		Power Power

		// Label is the name of the light
		Label string
	}

	// HSBK is used to represent the color and color temperature of a light.
	// The color is represented as an HSB (Hue, Saturation, Brightness) value.
	// The color temperature is represented in K (Kelvin) and is used
	// to adjust the warmness / coolness of a white light, which is most obvious when saturation is close zero.
	HSBK struct {
		// Hue is the color hue.
		// Range from 0 to 65535
		Hue uint16 `yaml:"hue" json:"hue"`

		// Saturation is the color saturation.
		// Range from 0 to 65535.
		Saturation uint16 `yaml:"saturation" json:"saturation"`

		// Brightness is the color brightness.
		// Range from 0 to 65535.
		Brightness uint16 `yaml:"brightness" json:"brightness"`

		// Kelvin is the color temperature.
		// Range from 2500(warm) to 9000(cool)
		Kelvin uint16 `yaml:"kelvin" json:"kelvin"`
	}

	// Power is personalized type.
	// It contains only two possible value: "on" and "off".
	Power string
)

var (
	// Off is a premade HSBK which turns off a light
	Off = &HSBK{
		Hue:        0,
		Saturation: 0,
		Brightness: 0,
		Kelvin:     0,
	}

	// On is a premade HSBK which turns on a light
	On = &HSBK{
		Hue:        65535,
		Saturation: 65535,
		Brightness: 32767,
		Kelvin:     65535,
	}

	productsList map[uint32]*Product
)

const (
	// PowerOn is the power level of a turned-on light
	PowerOn Power = "on"
	// PowerOff is the power level of a turned-off light
	PowerOff Power = "off"

	productsFile          = "PRODUCTS_FILE"
	defaultConfigFilePath = "./lifx/products.yaml"
)

// Send sends a message to a lifx device using the defined protocol.
func (l *Lifx) Send(message *Message) ([]byte, error) {
	// Defines the client, defined by its protocol.
	if l.client == nil {
		switch l.Protocol {
		case client.UDP:
			l.client = &udp.UDP{}
		default:
			return nil, errors.NotFoundf("protocol %s not found", l.Protocol)
		}
	}

	if len(l.Address.String()) == 0 {
		return nil, errors.NewNotValid(nil, "address of a lifx has not been initialized")
	}

	if len(l.Port) == 0 {
		return nil, errors.NewNotValid(nil, "port of a lifx has not been initialized")
	}

	if len(message.EncodeToBytes()) == 0 {
		return nil, errors.NewNotValid(nil, "message has not been initialized")
	}

	return l.client.Send(l.Address, l.Port, message.EncodeToBytes())
}

// Update update a Lifx device by sending multiple messages to that device.
// It parses all informations returned by this device and
// adds it in the lifx device struct.
func (l *Lifx) Update() error {
	// If an error occured, we cannot be sure that the targeted device is connected.
	// Therefore, its connected status is set to false.
	l.Connected = false
	// Sends a Get (101) Message
	bytes, err := l.Send(GetMessageWithoutPayload(Get))
	if err != nil {
		return errors.Annotate(err, "an error occured while sending a Get (101) Message on updating")
	}

	// Decodes the state
	state, err := DecodeToState(bytes)
	if err != nil {
		return err
	}

	// Defines the updated state value
	l.HSBK = state.HSBK
	l.Label = state.Label
	l.Power = state.Power

	// Sends a Get (53) Message
	bytes, err = l.Send(GetMessageWithoutPayload(GetGroup))
	if err != nil {
		return errors.Annotate(err, "an error occured while sending a GetGroup (53) Message on updating")
	}

	// Decodes the state
	group, err := DecodeToGroup(bytes)
	if err != nil {
		return err
	}

	// Defines the updated group value
	l.Group = group

	// Sends a GetInfo (34) Message
	bytes, err = l.Send(GetMessageWithoutPayload(GetInfo))
	if err != nil {
		return errors.Annotate(err, "an error occured while sending a GetInfo (34) Message on updating")
	}

	// Decodes the state
	info, err := DecodeToInfo(bytes)
	if err != nil {
		return err
	}

	// Defines the updated info value
	l.Info = info

	// Sends a GetLocation (48) Message
	bytes, err = l.Send(GetMessageWithoutPayload(GetLocation))
	if err != nil {
		return errors.Annotate(err, "an error occured while sending a GetLocation (48) Message on updating")
	}

	// Decodes the state
	location, err := DecodeToLocation(bytes)
	if err != nil {
		return err
	}

	// Defines the updated location value
	l.Location = location

	// Sends a GetLocation (48) Message
	bytes, err = l.Send(GetMessageWithoutPayload(GetVersion))
	if err != nil {
		return errors.Annotate(err, "an error occured while sending a GetVersion (32) Message on updating")
	}

	// Decodes the state
	product, err := DecodeToProduct(bytes)
	if err != nil {
		return err
	}

	// Defines the updated location value
	l.Product = product

	l.Connected = true
	return nil
}

// SetState send a new state to the lifx device.
func (l *Lifx) SetState(state *State, duration uint32) error {
	// If the label is not nil, it sends a setlabel message to the device.
	if len(state.Label) > 0 {
		err := l.SetLabel(state.Label)
		if err != nil {
			return errors.Annotate(err, "setting state")
		}
	}

	// If the power is not nil, it sends a setpowerdevice message to the device.
	if len(state.Power) > 0 {
		err := l.SetPower(state.Power)
		if err != nil {
			return errors.Annotate(err, "setting state")
		}
	}

	// If the hsbk is not nil, it sends a setcolor message to the device.
	if state.HSBK != nil {
		err := l.SetHSBK(state.HSBK, duration)
		if err != nil {
			return errors.Annotate(err, "setting state")
		}
	}

	return nil
}

// SetLabel sends a SetLabel message to the device.
func (l *Lifx) SetLabel(label string) error {
	// Sends a SetLabel message to the device
	_, err := l.Send(SetLabelMessage(label))
	if err != nil {
		return errors.Annotate(err, "setting new label")
	}

	// Updates device
	l.Label = label

	return nil
}

// SetPower send a SetPowerDevice message to the device.
func (l *Lifx) SetPower(power Power) error {
	// Sends a SetPower message to the device
	_, err := l.Send(SetPowerDeviceMessage(power))
	if err != nil {
		return errors.Annotate(err, "setting power")
	}

	// Updates device
	l.Power = power

	return nil
}

// SetHSBK sends a SetColor message with the given hsbk and duration.
// If it is successfull, it updates the device with the new state.
func (l *Lifx) SetHSBK(hsbk *HSBK, duration uint32) error {
	// Sends a SetColor message to the device
	bytes, err := l.Send(SetColorMessage(hsbk, duration))
	if err != nil {
		return errors.Annotate(err, "setting hsbk")
	}

	// Decodes state
	state, err := DecodeToState(bytes)
	if err != nil {
		return errors.Annotate(err, "setting hsbk")
	}

	// Updates device with returned values
	l.HSBK = state.HSBK
	l.Label = state.Label
	l.Power = state.Power

	return nil
}

// Toggle toggles a light HSBK. It is based on the power level of the device.
// If the power is "on" and the brightness > 0, the HSBK is set to off.
// Else, the HSBK is set to on.
// Finally the packet is sent to the targeted device.
func (l *Lifx) Toggle(brightness uint16, duration uint32) error {
	var bytes []byte
	var err error
	// If the power is on and brightness level greater than 0,
	// it turns off the light.
	if l.Power == PowerOn && l.HSBK.Brightness > 0 {
		bytes, err = l.Send(SetColorMessage(Off, duration))
		if err != nil {
			return errors.Annotate(err, "turning off a device")
		}
	} else {
		// Defines a `on` HSBK with the given brightness.
		on := &HSBK{
			Hue:        On.Hue,
			Saturation: On.Saturation,
			Brightness: brightness,
			Kelvin:     On.Kelvin,
		}

		bytes, err = l.Send(SetColorMessage(on, duration))
		if err != nil {
			return errors.Annotate(err, "turning on a device")
		}
	}

	// Decodes the response to a state
	state, err := DecodeToState(bytes)
	if err != nil {
		return err
	}

	// Updates state values
	l.HSBK = state.HSBK
	l.Label = state.Label
	l.Power = state.Power

	return nil
}

// DecodeToState decodes an array of bytes, given in arguments,
// and returns a State struct.
func DecodeToState(bytes []byte) (*State, error) {
	size := len(bytes)
	// TODO: add size verification
	// Decodes a part of the array to its HSBK value.
	hsbk, err := DecodeToHSBK(bytes[size-52 : size-44])
	if err != nil {
		return nil, errors.Annotate(err, "decoding state")
	}

	// Decodes the label, contained in the array of bytes.
	label := tools.DecodeToString(bytes[size-40 : size-8])

	// Determines the power level.
	power := PowerOff
	if binary.BigEndian.Uint16(bytes[size-42:size-40]) == 65535 {
		power = PowerOn
	}

	// Returns the State value.
	return &State{
		HSBK:  hsbk,
		Label: label,
		Power: power,
	}, nil
}

// DecodeToHSBK decodes an array of bytes, given in arguments,
// and returns its HSBK value.
func DecodeToHSBK(bytes []byte) (*HSBK, error) {
	// The array of bytes must have a length equals to 8.
	// A HSBK struct contains 4 uint16 variables (1 uint16 = 2 bytes)
	if len(bytes) != 8 {
		return nil, errors.NewNotValid(nil, "decoding a HSBK requires 8 bytes.")
	}

	// Defines the different fields.
	hue := binary.LittleEndian.Uint16(bytes[0:2])
	saturation := binary.LittleEndian.Uint16(bytes[2:4])
	brightness := binary.LittleEndian.Uint16(bytes[4:6])
	kelvin := binary.LittleEndian.Uint16(bytes[6:8])

	// Returns the HSBK value of the array of bytes
	return &HSBK{
		Hue:        hue,
		Saturation: saturation,
		Brightness: brightness,
		Kelvin:     kelvin,
	}, nil
}

// DecodeToGroup decodes an array of bytes, given in arguments,
// and returns its Group equivalent.
func DecodeToGroup(bytes []byte) (*Group, error) {
	size := len(bytes)

	// TODO: add size verification
	// Defines the id of the group
	buffer := make([]byte, 0, 16)
	buffer = append(buffer, bytes[size-56:size-40]...)
	id := [16]byte{}
	copy(id[:], buffer)

	// Defines the label of the group
	label := tools.DecodeToString(bytes[size-40 : size-8])

	// Returns the Group value of the array of bytes
	return &Group{
		ID:    id,
		Label: label,
	}, nil
}

// DecodeToInfo decodes an array of bytes, given in arguments,
// and returns its Info equivalent.
func DecodeToInfo(bytes []byte) (*Info, error) {
	size := len(bytes)
	// TODO: add size verification

	// Defines the different fields
	time := binary.LittleEndian.Uint16(bytes[size-24 : size-16])
	upTime := binary.LittleEndian.Uint16(bytes[size-16 : size-8])
	downTime := binary.LittleEndian.Uint16(bytes[size-8 : size])

	// Returns the Info value of the array of bytes
	return &Info{
		Time:     time,
		UpTime:   upTime,
		DownTime: downTime,
	}, nil
}

// DecodeToLocation decodes an array of bytes, given in arguments,
// and returns its Location equivalent.
func DecodeToLocation(bytes []byte) (*Location, error) {
	size := len(bytes)
	// TODO: add size verification

	// Defines the id of the location
	buffer := make([]byte, 0, 16)
	buffer = append(buffer, bytes[size-56:size-40]...)
	location := [16]byte{}
	copy(location[:], buffer)

	// Defines the label of the location
	label := tools.DecodeToString(bytes[size-40 : size-8])

	// Returns the Location value of the array of bytes
	return &Location{
		ID:    location,
		Label: label,
	}, nil
}

// DecodeToProduct decodes an array of bytes, given in arguments,
// and returns its Product equivalent.
func DecodeToProduct(bytes []byte) (*Product, error) {
	size := len(bytes)
	// TODO: add size verification

	// Defines the id of the product
	id := binary.LittleEndian.Uint32(bytes[size-8 : size-4])

	// Returns the Product value of the array of bytes
	return productsList[id], nil
}

// LoadProducts loads the products from a file pointed by PRODUCTS_FILE env variable
// or default to /lifx/products.yaml if empty.
// It initializes the ProductsList global variable.
func LoadProducts() error {
	filename := os.Getenv(productsFile)

	if filename == "" {
		filename = defaultConfigFilePath
	}
	log.WithField("filename", filename).Info("Parsing products file")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var products []*Product

	if err = yaml.Unmarshal(data, &products); err != nil {
		return errors.Annotate(err, "Cannot unmarshal config file")
	}

	productsList = map[uint32]*Product{}
	for _, product := range products {
		productsList[product.ID] = product
	}

	return nil
}
