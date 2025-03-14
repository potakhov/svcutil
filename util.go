package svcutil

import "reflect"

func getJSONTags(v any) map[string]string {
	tags := make(map[string]string)
	val := reflect.TypeOf(v)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tags[field.Name] = jsonTag
		}
	}

	return tags
}
