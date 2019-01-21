package api

import (
	"github.com/fberrez/horus/lifx"
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
)

type (
	// SelectorIn is the input struct, used in requests containing only a selector
	SelectorIn struct {
		// Selector is a unique identifier to select lights
		// which will be controlled by the request.
		Selector string `query:"selector" description:"The selector to limit which lights are controlled. More informations about format here: https://api.developer.lifx.com/docs/selectors" default:"all"`
	}

	// StateIn is the input struct, used in requests which edit lights state
	StateIn struct {
		// Selector is a unique identifier to select lights
		// which will be controlled by the request.
		Selector string `query:"selector" description:"The selector to limit which lights are controlled. More informations about format here: https://api.developer.lifx.com/docs/selectors" default:"all"`

		// HSBK is the color of the light
		HSBK *HSBKIn `json:"hsbk" decription:"HSBK contains Hue, Saturation, Brightness and Kelvin. Used to represented the color."`

		// Duration determines how long in milliseconds will take the power action. Range: 0 – 4294967295 (~49 days)
		// Its default value is 0.
		Duration uint32 `json:"duration" description:"The time is seconds to spend perfoming the power toggle." validate:"min=0,max=4294967295" default:"0"`

		// Power is the current power level of the light
		Power string `json:"power" description:"The power state you want to set on the selector. on or off" enum:"on,off"`

		// Label is the name of the light
		Label string `json:"label" description:"new label of the LIFX device"`
	}

	// HSBKIn is used to represent the color and color temperature of a light.
	// The color is represented as an HSB (Hue, Saturation, Brightness) value.
	// The color temperature is represented in K (Kelvin) and is used
	// to adjust the warmness / coolness of a white light, which is most obvious when saturation is close zero.
	HSBKIn struct {
		// Hue is the color hue.
		// Range from 0 to 65535
		Hue uint16 `yaml:"hue" json:"hue" description:"The color hue" validate:"min=0,max=65535,required" `

		// Saturation is the color saturation.
		// Range from 0 to 65535.
		Saturation uint16 `yaml:"saturation" json:"saturation" description:"The color saturation" validate:"min=0,max=65535,required"`

		// Brightness is the color brightness.
		// Range from 0 to 65535.
		Brightness uint16 `yaml:"brightness" json:"brightness" description:"The color brightness" validate:"min=0,max=65535,required"`

		// Kelvin is the color temperature.
		// Range from 2500(warm) to 9000(cool)
		Kelvin uint16 `yaml:"kelvin" json:"kelvin" description:"The color temperature" validate:"min=2500,max=9000,required"`
	}

	// DurationIn is used on the toggle route. It contains a selector and a duration in milliseconds.
	DurationIn struct {
		// Selector is a unique identifier to select lights
		// which will be controlled by the request.
		// Its default value is `all`
		Selector string `query:"selector" description:"The selector to limit which lights are controlled. More informations about format here: https://api.developer.lifx.com/docs/selectors"`

		// Duration determines how long in milliseconds will take the power action. Range: 0 – 4294967295 (~136 years)
		// Its default value is 0.
		Duration uint32 `json:"duration" description:"The time is seconds to spend perfoming the power toggle." validate:"min=0,max=4294967295" default:"0"`
	}

	// ResultOut contains all the information concerning
	// the operation success status.
	ResultOut struct {
		// UUID is the UUID of the LIFX device
		UUID string `json:"uuid" description:"UUID of the LIFX device"`

		// Label is the label of the LIFX device
		Label string `json:"label" description:"Label of the LIFX device"`

		// Error contains informations about the error when the operation has not been successfull.
		Error error `json:"error" description:"Informations concerning the error are here"`
	}
)

// getLights returns the list of corresponding lights in the selector.
func (a *API) getDevices(c *gin.Context, in *SelectorIn) ([]*lifx.Lifx, error) {
	logger := log.WithField("action", "get-devices")

	if len(a.config.Lifx) == 0 {
		return nil, errors.NewNotProvisioned(nil, "list of Lifx devices")
	}

	// Updates the list of lifx devices
	err := a.updateLifx()
	if err != nil {
		return nil, err
	}

	// Parses the selector
	selector, err := a.parseSelector(in.Selector)
	if err != nil {
		return nil, err
	}

	logger.WithField("selector", selector).Debug("selector found")
	return a.sortBySelector(selector)
}

// setState sets a new state to the corresponding lights in the selector.
func (a *API) setState(c *gin.Context, in *StateIn) ([]*ResultOut, error) {
	logger := log.WithField("action", "set-state")

	// Parses the selector
	selector, err := a.parseSelector(in.Selector)
	if err != nil {
		return nil, err
	}

	logger.WithField("selector", selector).Debug("selector found")

	devices, err := a.sortBySelector(selector)

	if err != nil {
		return nil, err
	}

	var hsbk *lifx.HSBK
	if in.HSBK != nil {
		hsbk = &lifx.HSBK{
			Hue:        in.HSBK.Hue,
			Saturation: in.HSBK.Saturation,
			Brightness: in.HSBK.Brightness,
			Kelvin:     in.HSBK.Kelvin,
		}
	}

	state := &lifx.State{
		Power: lifx.Power(in.Power),
		HSBK:  hsbk,
		Label: in.Label,
	}

	// results contains the result of each performed operation.
	results := []*ResultOut{}
	for _, device := range devices {
		err := device.SetState(state, in.Duration)
		result := &ResultOut{
			UUID:  device.UUID,
			Label: device.Label,
			Error: err,
		}
		results = append(results, result)
	}

	return results, nil
}

// toggle toggles the power of the corresponding lights in the selector.
func (a *API) toggle(c *gin.Context, in *DurationIn) ([]*ResultOut, error) {
	logger := log.WithField("action", "toggle")

	if len(a.config.Lifx) == 0 {
		return nil, errors.NewNotProvisioned(nil, "list of Lifx devices")
	}

	// Parses the selector
	selector, err := a.parseSelector(in.Selector)
	if err != nil {
		return nil, err
	}

	logger.WithField("selector", selector).Debug("selector found")
	// Sorts the array of known devices to return every corresponding devices
	// to the selector.
	devices, err := a.sortBySelector(selector)
	if err != nil {
		return nil, err
	}

	// results contains the result of each performed operation.
	results := []*ResultOut{}
	for _, device := range devices {
		err := device.Toggle(a.config.MaxBrightness, in.Duration)
		result := &ResultOut{
			UUID:  device.UUID,
			Label: device.Label,
			Error: err,
		}
		results = append(results, result)
	}

	return results, nil
}
