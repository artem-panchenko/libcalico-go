// Copyright (c) 2016 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"github.com/tigera/libcalico-go/lib/common"
	"reflect"
)

// RawString is used a value type to indicate that the value is a bare non-JSON string
type rawString string

var rawStringType = reflect.TypeOf(rawString(""))

// Key represents a parsed datastore key.
type Key interface {
	// DefaultPath() returns a default stringified path for this object,
	// suitable for use in most datastores (and used on the Felix API,
	// for example).
	DefaultPath() (string, error)
	// DefaultDeletePath() returns a default stringified path for deleting
	// this object.
	DefaultDeletePath() (string, error)
	valueType() reflect.Type
}

// Interface used to perform datastore lookups.
type ListInterface interface {
	// DefaultPathRoot() returns a default stringified root path, i.e. path
	// to the directory containing all the keys to be listed.
	DefaultPathRoot() string
	ParseDefaultKey(key string) Key
}

// KVPair holds a parsed key and value as well as datastore specific revision
// information.
type KVPair struct {
	Key      Key
	Value    interface{}
	Revision interface{}
}

// ParseKey parses a datastore key into one of the <Type>Key structs.
// Returns nil if the string doesn't match one of our objects.
func ParseKey(key string) Key {
	if m := matchWorkloadEndpoint.FindStringSubmatch(key); m != nil {
		return WorkloadEndpointKey{
			Hostname:       m[1],
			OrchestratorID: m[2],
			WorkloadID:     m[3],
			EndpointID:     m[4],
		}
	} else if m := matchPolicy.FindStringSubmatch(key); m != nil {
		return PolicyKey{
			Tier: m[1],
			Name: m[2],
		}
	} else if m := matchProfile.FindStringSubmatch(key); m != nil {
		pk := ProfileKey{m[1]}
		switch m[2] {
		case "tags":
			return ProfileTagsKey{ProfileKey: pk}
		case "rules":
			return ProfileRulesKey{ProfileKey: pk}
		case "labels":
			return ProfileLabelsKey{ProfileKey: pk}
		}
		return nil
	} else if m := matchTier.FindStringSubmatch(key); m != nil {
		return TierKey{Name: m[1]}
	} else if m := matchHostIp.FindStringSubmatch(key); m != nil {
		return HostIPKey{Hostname: m[1]}
	} else if m := matchPool.FindStringSubmatch(key); m != nil {
		_, c, _ := common.ParseCIDR(m[1])
		return PoolKey{CIDR: *c}
	} else if m := matchGlobalConfig.FindStringSubmatch(key); m != nil {
		return GlobalConfigKey{Name: m[1]}
	} else if m := matchHostConfig.FindStringSubmatch(key); m != nil {
		return HostConfigKey{Hostname: m[1], Name: m[2]}
	}
	// Not a key we know about.
	return nil
}

func ParseValue(key Key, rawData []byte) (interface{}, error) {
	valueType := key.valueType()
	if valueType == rawStringType {
		return string(rawData), nil
	}
	value := reflect.New(valueType)
	iface := value.Interface()
	err := json.Unmarshal(rawData, iface)
	if err != nil {
		glog.Errorf("Failed to unmarshal %#v into value %#v",
			string(rawData), value)
		return nil, err
	}
	if value.Elem().Kind() != reflect.Struct {
		// Pointer to a map or slice, unwrap.
		iface = value.Elem().Interface()
	}
	return iface, nil
}

func ParseKeyValue(key string, rawData []byte) (Key, interface{}, error) {
	parsedKey := ParseKey(key)
	if parsedKey == nil {
		return nil, nil, errors.New("Failed to parse key")
	}
	value, err := ParseValue(parsedKey, rawData)
	return parsedKey, value, err
}
