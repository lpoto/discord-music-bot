// Package config adds support for loading configuration from multiple yaml files.
package config

import (
	"fmt"
	"io/ioutil"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/go-playground/validator/v10"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

func LoadConfiguration(configFiles []string, target interface{}) error {
	for _, configFilePath := range configFiles {
		log.WithFields(log.Fields{"File": configFilePath}).Info("Parsing config file")
		rawContent, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return err
		}
		cfg := newZeroFor(target)
		err = yaml.Unmarshal(rawContent, cfg)
		if err != nil {
			return err
		}
		err = mergo.Merge(target, cfg, mergo.WithOverride)
		if err != nil {
			return err
		}

	}
	return nil
}

// When loading YAML we need a zero value of a specific type in order to drive the parsing, but YAML parser does not
// support deep merging (it will just override at the top level) - so `mergo` is used.
// So this means that we now need a `target` zero value for each of the config files, but we like to keep the public API
// which mimics that of YAML (and JSON parsing). Thus the need for a function that will take a pointer to an arbitrary
// struct type and produce a pointer to a new zero value for that type.
// WARNING: this will crash if passed and interface value to something other than a pointer
func newZeroFor(target interface{}) interface{} {
	return reflect.New(reflect.TypeOf(target).Elem()).Interface()
}

// ValidateConfiguration takes (should take) a struct and validates its fields against predefined `validate` tags.
// The underlying validate.Struct method returns two types of errors. validator.InvalidValidationError for when the
// validation breaks, e.g. when a wrong type is passed as the argument (check validate.StructCtx). In this case, we wrap
// things with a plain error. The other case are actual validation errors. In this case a validator.ValidationErrors is
// returned, meaning our abstraction leaks and the assumption/recommendation is to use only error's Error() method,
// i.e. not to resort to type assertions on the returned instance.
func ValidateConfiguration(target interface{}) error {
	validate := validator.New()
	err := validate.Struct(target)
	if _, ok := err.(*validator.InvalidValidationError); ok {
		return fmt.Errorf("could not validate input (%v): %v", target, err)
	}
	return err
}

// LoadAndValidateConfiguration is a convenience method that does two logical steps in one go. Make sure to always check
// for errors returned, certain fields might be loaded while others could fail.
func LoadAndValidateConfiguration(configFiles []string, target interface{}) (err error) {
	err = LoadConfiguration(configFiles, target)
	if err != nil {
		return
	}
	err = ValidateConfiguration(target)
	return
}
