package resource

import (
	"regexp"

	"log"

	"errors"
	"fmt"

	"time"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// AppFs is an abstraction of the file system
// to allow mocking in tests.
var AppFs = afero.NewOsFs()

// Config represents the content of a yaml file that is used as a contract to filter resources for deletion.
type Config map[string]configEntry

// configEntry represents an entry in Config and selects the resources of a particular resource type.
type configEntry struct {
	// regexps to select resources by IDs
	Ids []*string `yaml:",omitempty"`
	// regexps to select resources by tags
	Tags map[string]string `yaml:",omitempty"`
	// select resources by creation time
	Created Created `yaml:",omitempty"`
}

type Created struct {
	Before time.Time `yaml:",omitempty"`
	After  time.Time `yaml:",omitempty"`
}

// Filter selects resources based on a given yaml config.
type Filter struct {
	cfg Config
}

// FilterableResource holds the resource attributes that are matched against the filter.
type FilterableResource struct {
	Type string
	ID   string
	Tags map[string]string
}

// NewFilter creates a new filter based on a given yaml config.
func NewFilter(filename string, c *AWSClient) Filter {
	config := read(filename)

	filter := Filter{
		cfg: config,
	}

	err := filter.Validate(c)
	if err != nil {
		log.Fatal(err)
	}

	return filter
}

// read reads a filter from a yaml file.
func read(filename string) Config {
	var cfg Config

	data, err := afero.ReadFile(AppFs, filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

// ResourceTypes returns all resource types in the config.
func (f Filter) ResourceTypes() []string {
	var resTypes []string

	for key := range f.cfg {
		resTypes = append(resTypes, key)
	}
	return resTypes
}

// Validate checks if all resource types appearing in the config are currently supported.
func (f Filter) Validate(c *AWSClient) error {
	as := Supported(c)
	for _, resType := range f.ResourceTypes() {
		isTerraformType := false
		for _, a := range as {
			if resType == a.TerraformType {
				isTerraformType = true
			}
		}
		if !isTerraformType {
			return fmt.Errorf("found unsupported resource type '%s' in config file", resType)
		}
	}
	return nil
}

// matchID checks whether a resource (given by its type and id) matches the filter.
func (f Filter) matchID(resource FilterableResource) (bool, error) {
	cfgEntry, _ := f.cfg[resource.Type]

	if len(cfgEntry.Ids) == 0 {
		return false, errors.New("no entries set in filter to match IDs")
	}

	for _, regex := range cfgEntry.Ids {
		if ok, err := regexp.MatchString(*regex, resource.ID); ok {
			if err != nil {
				log.Fatal(err)
			}
			return true, nil
		}
	}

	return false, nil
}

// matchTags checks whether a resource (given by its type and tags)
// matches the filter. The keys must match exactly, whereas
// the tag value is checked against a regex.
func (f Filter) matchTags(resource FilterableResource) (bool, error) {
	cfgEntry, _ := f.cfg[resource.Type]

	if len(cfgEntry.Tags) == 0 {
		return false, errors.New("filter has no tag entries for resource type")
	}

	for cfgTagKey, regex := range cfgEntry.Tags {
		if tagVal, ok := resource.Tags[cfgTagKey]; ok {
			if res, err := regexp.MatchString(regex, tagVal); res {
				if err != nil {
					log.Fatal(err)
				}
				return true, nil
			}
		}
	}

	return false, nil
}

// Matches checks whether a resource (given by its type and tags) matches
// the configured filter criteria for tags and ids.
func (f Filter) Matches(resource FilterableResource) bool {
	var matchesTags = false
	var errTags error

	if resource.Tags != nil {
		matchesTags, errTags = f.matchTags(resource)
	}
	matchesID, errID := f.matchID(resource)

	// if the filter has neither an entry to match ids nor tags,
	// select all resources of that type
	if errID != nil && errTags != nil {
		return true
	}

	return matchesID || matchesTags
}
