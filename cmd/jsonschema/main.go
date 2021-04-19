package main

import (
	"github.com/norwoodj/helm-docs/pkg/document"
	"github.com/norwoodj/helm-docs/pkg/helm"
	"github.com/vrischmann/envconfig"

	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
)

type Config struct {
	ChartDir      string
	SchemaInPath  string
	SchemaOutPath string
}

type valueRow struct {
	Default         interface{}
	AutoDescription string
}

func main() {
	var cfg Config
	err := envconfig.Init(&cfg)
	exitOnError(err)

	chartDocumentationInfo, err := helm.ParseChartInformation(cfg.ChartDir)
	exitOnError(err)

	out, err := document.GetChartTemplateData(chartDocumentationInfo, "0.0.1")
	exitOnError(err)

	comments := map[string]valueRow{}
	for _, item := range out.Values {
		v := valueRow{
			AutoDescription: item.AutoDescription,
		}
		item.Default = strings.TrimSuffix(item.Default, "`")
		item.Default = strings.TrimPrefix(item.Default, "`")
		if item.Default != "" {
			if item.Default == "nil" {
				item.Default = "null"
			}
			err := json.Unmarshal([]byte(item.Default), &v.Default)
			exitOnError(err)
		}
		comments[item.Key] = v
	}

	inSchemaFile, err := ioutil.ReadFile(cfg.SchemaInPath)
	exitOnError(err)

	var inSchema interface{}
	err = json.Unmarshal(inSchemaFile, &inSchema)
	exitOnError(err)

	descAndDefaultEnricher := func(key *string, index *int, value *interface{}) {
		if key != nil { // It's an object key/value pair...
			if comment, found := comments[*key]; found {
				old, ok := (*value).(map[string]interface{})
				if !ok {
					return
				}
				old["description"] = comment.AutoDescription
				if comment.Default != nil {
					old["default"] = comment.Default
				}
				*value = old
			}
		} else {
			// ignore arrays
		}
	}

	forEachJSONKey(&inSchema, "", descAndDefaultEnricher)

	final, err := json.MarshalIndent(inSchema, "", "  ")
	exitOnError(err)

	err = ioutil.WriteFile(cfg.SchemaOutPath, final, 0644)
	exitOnError(err)
}

func forEachJSONKey(obj *interface{}, key string, handler func(*string, *int, *interface{})) {
	if obj == nil {
		return
	}
	// Yield all key/value pairs for objects.
	o, isObject := (*obj).(map[string]interface{})
	if isObject {
		for k, v := range o {
			if k == "properties" {
				k = ""
			}
			if key == "properties" {
				key = ""
			}

			var path []string
			if key != "" {
				path = append(path, key)
			}
			if k != "" {
				path = append(path, k)
			}

			k = strings.Join(path, ".")
			handler(&k, nil, &v)
			forEachJSONKey(&v, k, handler)
		}
	}
	// Yield each index/value for arrays.
	a, isArray := (*obj).([]interface{})
	if isArray {
		for i, x := range a {
			handler(nil, &i, &x)
			forEachJSONKey(&x, key, handler)
		}
	}
}

func exitOnError(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}
