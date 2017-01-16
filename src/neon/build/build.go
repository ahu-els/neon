package build

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"neon/util"
	"path/filepath"
	"sort"
	"unicode/utf8"
)

type Build struct {
	File    string
	Dir     string
	Name    string
	Default []string
	Doc     string
	Context *Context
	Targets map[string]*Target
	Debug   bool
}

func NewBuild(file string, debug bool) (*Build, error) {
	build := &Build{}
	build.Debug = debug
	build.Log("Loading build file '%s'", file)
	source, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("loading build file '%s': %v", file, err)
	}
	build.Log("Parsing build file")
	var object util.Object
	err = yaml.Unmarshal(source, &object)
	if err != nil {
		return nil, fmt.Errorf("build must be a YAML map with string keys")
	}
	build.Log("Build structure: %#v", object)
	build.Log("Reading build first level fields")
	err = object.CheckFields([]string{"name", "default", "doc", "properties", "targets"})
	if err != nil {
		return nil, err
	}
	str, err := object.GetString("name")
	if err == nil {
		build.Name = str
	}
	list, err := object.GetListStringsOrString("default")
	if err == nil {
		build.Default = list
	}
	str, err = object.GetString("doc")
	if err == nil {
		build.Doc = str
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return nil, fmt.Errorf("getting build file path: %v", err)
	}
	build.File = path
	build.Dir = filepath.Dir(path)
	properties, err := object.GetObject("properties")
	if err != nil {
		if err.Error() == "field 'properties' not found" {
			properties = make(map[string]interface{})
		} else {
			return nil, err
		}
	}
	context, err := NewContext(build, properties)
	if err != nil {
		return nil, err
	}
	build.Context = context
	targets, err := object.GetObject("targets")
	if err != nil {
		if err.Error() == "field 'targets' not found" {
			properties = make(map[string]interface{})
		} else {
			return nil, err
		}
	}
	build.Targets = make(map[string]*Target)
	for name, _ := range targets {
		object, err := targets.GetObject(name)
		if err != nil {
			return nil, err
		}
		target, err := NewTarget(build, name, object)
		if err != nil {
			return nil, err
		}
		build.Targets[name] = target
	}
	return build, nil
}

func (build *Build) Run(targets []string) error {
	if len(targets) == 0 {
		if len(build.Default) == 0 {
			return fmt.Errorf("no default target")
		}
		return build.Run(build.Default)
	} else {
		for _, name := range targets {
			target, ok := build.Targets[name]
			if !ok {
				return fmt.Errorf("target '%s' not found", name)
			}
			err := target.Run()
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (build *Build) Help() error {
	newLine := false
	// print build documentation
	if build.Doc != "" {
		fmt.Println(build.Doc)
		newLine = true
	}
	// print build properties
	length := 0
	for _, name := range build.Context.Properties {
		if utf8.RuneCountInString(name) > length {
			length = utf8.RuneCountInString(name)
		}
	}
	if len(build.Context.Properties) > 0 {
		if newLine {
			fmt.Println()
		}
		fmt.Println("Properties:")
		for _, name := range build.Context.Properties {
			value, err := build.Context.GetProperty(name)
			if err != nil {
				return fmt.Errorf("getting property '%s': %v", name, err)
			}
			valueStr, err := PropertyToString(value, true)
			if err != nil {
				return fmt.Errorf("formatting property '%s': %v", name, err)
			}
			util.PrintTargetHelp(name, valueStr, []string{}, length)
		}
		newLine = true
	}
	// print targets documentation
	length = 0
	var targets []string
	for name, _ := range build.Targets {
		if utf8.RuneCountInString(name) > length {
			length = utf8.RuneCountInString(name)
		}
		targets = append(targets, name)
	}
	sort.Strings(targets)
	if len(targets) > 0 {
		if newLine {
			fmt.Println()
		}
		fmt.Println("Targets:")
		for _, name := range targets {
			target := build.Targets[name]
			util.PrintTargetHelp(name, target.Doc, target.Depends, length)
		}
	}
	return nil
}

func (build *Build) Log(message string, args ...interface{}) {
	if build.Debug {
		fmt.Println(fmt.Sprintf(message, args...))
	}
}