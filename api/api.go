package api

import (
	"encoding/binary"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/fberrez/horus/lifx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		// APIKey is the key of the API
		APIKey uuid.UUID `yaml:"apiKey" json:"apiKey"`

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

	id = &selector{
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
	selectors := append([]*selector{}, all, label, id, groupID,
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
		Version:     "0.0.2",
	}

	// Defines groups of routes
	lightsGroup := f.Group("/lights", "Lights", "")
	unsecuredGroup := f.Group("/unsecured", "Unsecured", "")

	// Defines Unsecured group's routes
	unsecuredGroup.GET("/openapi.json", nil, f.OpenAPI(infos, "json"))
	unsecuredGroup.GET("/generate", []fizz.OperationOption{
		fizz.Summary("Generates an API key."),
		fizz.Description("Returns an API key which must be used in /lights routes."),
	}, tonic.Handler(api.generateKey, http.StatusOK))

	// Defines Lights group's middlewares
	lightsGroup.Use(gin.HandlerFunc(api.verifyKey))

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

// verifyKey verifies the value of the API key
func (a *API) verifyKey(c *gin.Context) {
	log.Debug("verifying api key")
	key := c.Query("key")

	// If the api key has not been initialized
	if a.config.APIKey == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errors.BadRequestf("api key not generated").Error(),
		})
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// If the key has not been set as a query parameter
	if len(key) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errors.BadRequestf("missing api key").Error(),
		})
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// If the key is not valid
	if key != a.config.APIKey.String() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": errors.Unauthorizedf("api key not valid").Error(),
		})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	return
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
