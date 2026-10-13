// Copyright (c) 2024 Enflame. All Rights Reserved.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
)

var filePath = flag.String("file", "example.json", "The json or yaml file name")
var fieldPath = flag.String("path", "driver.name", "Specify the path to the fields to be read, separated by '.'")

func main() {
	flag.Parse()
	value, err := getFieldByPath(*filePath, *fieldPath)
	if err != nil {
		fmt.Printf("Error: get field: %s from %s failed: %v\n", *fieldPath, *filePath, err)
		return
	}
	if value == nil {
		fmt.Printf("Error: field: %s in %s not found\n", *fieldPath, *filePath)
		return
	}
	fmt.Println(value)
}

func getFieldByPath(filePath, fieldPath string) (interface{}, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(filePath, ".yaml") {
		content, err = yaml.YAMLToJSON(content)
		if err != nil {
			return nil, err
		}
	}
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	return getFieldFromData(data, strings.Split(fieldPath, "."))
}

func getFieldFromData(data interface{}, fieldPath []string) (interface{}, error) {
	if len(fieldPath) == 0 {
		return data, nil
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%v is not map type", data)
	}
	return getFieldFromData(m[fieldPath[0]], fieldPath[1:])
}
