package api

import (
	"encoding/binary"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/fberrez/horus/lifx"
	"github.com/juju/errors"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/loopfz/gadgeto/tonic/utils/jujerr"
	log "github.com/sirupsen/logrus"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
	yaml "gopkg.in/yaml.v2"
)

type (
	// API Contains each part of the api settings.
	API struct {
		// fizz is the server which handles http routes of the API.
		fizz *fizz.Fizz

		// config is the configuration of the client.
		config *Config

		// selectors is an array which contains all selectors.
		selectors []*selector
	}

	// Config contains all informations needed to run the application.
	Config struct {
		// Source is the source identifier, used to identify the client from others.
		Source uint32 `yaml:"source" json:"source"`

		// MaxBrightness is the maximum brightness value.
		// It is used on /lights/toggle.
		// Range from 0 to 65535.
		MaxBrightness uint16 `yaml:"maxBrightness" json:"maxBrightness"`

		// Lifx contains informations of all Lifx connected devices.
		Lifx []*lifx.Lifx `yaml:"lifx" json:"lifx"`
	}

	selector struct {
		name      string
		isDynamic bool
		value     string
	}
)

const (
	configFile            = "CONFIG_FILE"
	defaultConfigFilePath = "config.yaml"
)

var (
	all = &selector{
		name:      "all",
		isDynamic: false,
	}

	label = &selector{
		name:      "label",
		isDynamic: true,
	}

	uuid = &selector{
		name:      "uuid",
		isDynamic: true,
	}

	groupID = &selector{
		name:      "group_id",
		isDynamic: true,
	}

	group = &selector{
		name:      "group",
		isDynamic: true,
	}

	locationID = &selector{
		name:      "location_id",
		isDynamic: true,
	}

	location = &selector{
		name:      "location",
		isDynamic: true,
	}

	sceneID = &selector{
		name:      "scene_id",
		isDynamic: true,
	}
)

// New parses the config file and initializes a new API.
func New() (*API, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// Setting source
	binary.LittleEndian.PutUint32(lifx.Source[:], config.Source)

	log.Info("Initializing API")
	f := fizz.New()

	// Initializes the array of selectors
	selectors := append([]*selector{}, all, label, uuid, groupID,
		group, locationID, location, sceneID)

	api := &API{
		fizz:      f,
		config:    config,
		selectors: selectors,
	}

	// API informations
	infos := &openapi.Info{
		Title:       "Horus - Up your local LIFX devices",
		Description: "Horus is an API which handles your LIFX devices in your local network. It uses UDP packets to interact with them. It has been designed to simplify your interactions with your LIFX devices, without cloud connection.",
		Version:     "0.0.1",
	}

	// Defines GET route of API documentation
	f.GET("/unsecured/openapi.json", nil, f.OpenAPI(infos, "json"))

	// Defines groups of routes
	lightsGroup := f.Group("/lights", "Lights", "")

	// Defines Lights group's routes
	lightsGroup.GET("/", []fizz.OperationOption{
		fizz.Summary("Gets a list of corresponding lights in the selector."),
		fizz.Description("Returns a list of lights with their informations."),
		fizz.Response(string(http.StatusNotFound), "cannot find corresponding lights in the selector.", nil, nil),
	}, tonic.Handler(api.getDevices, http.StatusOK))

	lightsGroup.PUT("/state", []fizz.OperationOption{
		fizz.Summary("Updates the state of the corresponding lights."),
		fizz.Description("Updates the lights state with the given settings."),
		fizz.Response(string(http.StatusNotFound), "cannot find corresponding lights in the selector.", nil, nil),
	}, tonic.Handler(api.setState, http.StatusOK))

	lightsGroup.POST("/toggle", []fizz.OperationOption{
		fizz.Summary("Toggles power status of corresponding lights."),
		fizz.Description(""),
		fizz.Response(string(http.StatusNotFound), "cannot find corresponding lights in the selector.", nil, nil),
	}, tonic.Handler(api.toggle, http.StatusOK))

	tonic.SetErrorHook(jujerr.ErrHook)

	return api, nil
}

// ServeHTTP is the implementation of http.Handler.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.WithFields(log.Fields{
		"start_time":  start.Unix(),
		"remote_addr": r.RemoteAddr,
		"request":     r.RequestURI,
	}).Info("Request received.")

	a.updateLifx()
	a.fizz.ServeHTTP(w, r)
}

// LoadConfig loads the config from a file pointed by CONFIG_FILE env variable
// or default to config.yaml if empty.
// It returns a struct containing all informations
func loadConfig() (*Config, error) {
	filename := os.Getenv(configFile)

	if filename == "" {
		filename = defaultConfigFilePath
	}
	log.WithField("filename", filename).Info("Parsing config file")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config

	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Annotate(err, "Cannot unmarshal config file")
	}

	return &config, nil
}
