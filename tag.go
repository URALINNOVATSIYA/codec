package codec

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type TagMap map[int]Tag // Keys are field indexes

func (m TagMap) IndexById(id int) int {
	for i, t := range m {
		if t.Id == id {
			return i
		}
	}
	return -1
}

type Tag struct {
	Id         int  // Field identifier
	Deprecated bool // Field is no longer used
}

func ParseTags(t reflect.Type) (TagMap, error) {
	if t == nil || t.Kind() != reflect.Struct {
		return nil, nil
	}
	m := make(TagMap)
	for field := range t.Fields() {
		v, exists := field.Tag.Lookup("codec")
		if !exists {
			continue
		}
		id, deprecated, err := parseTagValue(v)
		if err != nil {
			return nil, err
		}
		if id < 0 {
			continue
		}
		m[field.Index[0]] = Tag{
			Id:         id,
			Deprecated: deprecated,
		}
	}
	return m, nil
}

func parseTagValue(tag string) (id int, deprecated bool, err error) {
	for _, v := range strings.Split(tag, ",") {
		v = strings.TrimSpace(v)
		if v == "deprecated" {
			deprecated = true
			continue
		}
		if !strings.HasPrefix(v, "id") {
			continue
		}
		parts := strings.Split(v, "=")
		if len(parts) < 2 {
			err = errors.New("invalid codec tag format: identifier is not set")
			return
		}
		id, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			err = fmt.Errorf("invalid codec tag format: %s", err)
			return
		}
		if id < 0 {
			err = errors.New("invalid codec tag format: field identifier must not be negative")
			return
		}
	}
	return
}
