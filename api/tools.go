package api

import (
	"fmt"
	"strings"

	"github.com/fberrez/horus/lifx"
	"github.com/juju/errors"
)

func (a *API) updateLifx() error {
	for _, device := range a.config.Lifx {
		err := device.Update()
		if err != nil {
			return err
		}
	}

	return nil
}

// parseSelector parses a selector
func (a *API) parseSelector(selectorStr string) (*selector, error) {
	// Defines error message
	errNotValid := errors.NotValidf("selector `%s`", selectorStr)
	// If the selector does not exist, so it returns the default selector (all).
	if len(selectorStr) == 0 {
		return all, nil
	}

	selector := &selector{}
	// If the selector does not contain a `:`,
	// we suppose that the selector is static.
	// Therefore, it compares its name with the existing selectors.
	if !strings.Contains(selectorStr, ":") {
		for _, s := range a.selectors {
			if selectorStr == s.name {
				// If the found selector is dynamic, the given selector is not valid
				// because it does not use the format `type:value`
				if s.isDynamic {
					annotation := "since it is a dynamic selector, it must have the format `type:value`"
					return nil, errors.Annotate(errNotValid, annotation)
				}

				return s, nil
			}
		}

		return nil, errors.NotFoundf("selector `%s`", selectorStr)
	}

	// Else we suppose it is a dynamic selector.
	parts := strings.Split(selectorStr, ":")
	// If it does not contain a `:` or there are more than one `:`,
	// the selector is considered invalid.
	if len(parts) != 2 {
		annotation := "since it is a dynamic selector, it must have the format `type:value`"
		return nil, errors.Annotate(errNotValid, annotation)
	}

	// It compares the first part of the selector with the existing selectors.
	for _, s := range a.selectors {
		if parts[0] == s.name {
			selector = s
			selector.value = parts[1]
			return selector, nil
		}
	}

	// If it does not found anything, the selector is not found
	return nil, errors.NotFoundf("selector `%s`", selectorStr)
}

// sortBySelector returns an array of devices corresponding to selector.
// If that array is empty or the selector has not been implemented,
// it returns an error.
func (a *API) sortBySelector(selector *selector) ([]*lifx.Lifx, error) {
	devices := []*lifx.Lifx{}

	// For each lifx devices, it will test the correspondence with the selector.
	// For example, if the selector is a sorting by name, it will test if a device
	// corresponds to the value of the selector. If it is successfull, it adds the device
	// to the array contains all corresponding devices.
	for _, device := range a.config.Lifx {
		switch selector.name {
		case all.name:
			return a.config.Lifx, nil
		case label.name:
			// If the value of the selector is identical to the label of the device,
			// it adds it to the array
			if selector.value == device.Label {
				devices = append(devices, device)
				continue
			}
		case uuid.name:
			// If the value of the selector is identical to UUID of the device...
			if selector.value == device.UUID {
				devices = append(devices, device)
				continue
			}
		case groupID.name:
			return nil, errors.NotImplementedf("selector %s", sceneID.name)
		case group.name:
			// If the value of the selector is identical to the group label of the device...
			if selector.value == device.Group.Label {
				devices = append(devices, device)
				continue
			}
		case locationID.name:
			return nil, errors.NotImplementedf("selector %s", sceneID.name)
		case location.name:
			// If the value of the selector is identical to the location label of the device...
			if selector.value == device.Location.Label {
				devices = append(devices, device)
				continue
			}
		case sceneID.name:
			return nil, errors.NotImplementedf("selector %s", sceneID.name)
		default:
			return nil, errors.NotFoundf("sorting by selector %s", selector.name)
		}
	}

	if len(devices) == 0 {
		return nil, errors.NotFoundf("devices corresponding to selector %s", selector)
	}

	return devices, nil
}

// String returns a string-formatted selector
func (s *selector) String() string {
	if s.isDynamic {
		return fmt.Sprintf("%s:%s", s.name, s.value)
	}

	return s.name
}
