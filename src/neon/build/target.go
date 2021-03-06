package build

import (
	"fmt"
	"neon/util"
	"os"
)

// Structure for a target
type Target struct {
	Build   *Build
	Name    string
	Doc     string
	Depends []string
	Steps   []Step
}

// Make a target
func NewTarget(build *Build, name string, object util.Object) (*Target, error) {
	target := &Target{
		Build: build,
		Name:  name,
	}
	err := object.CheckFields([]string{"doc", "depends", "steps"})
	if err != nil {
		return nil, fmt.Errorf("parsing target '%s': %v", name, err)
	}
	if err := ParseTargetDoc(object, target); err != nil {
		return nil, err
	}
	if err := ParseTargetDepends(object, target); err != nil {
		return nil, err
	}
	if err := ParseTargetSteps(object, target); err != nil {
		return nil, err
	}
	return target, nil
}

// Parse target documentation
func ParseTargetDoc(object util.Object, target *Target) error {
	if object.HasField("doc") {
		doc, err := object.GetString("doc")
		if err != nil {
			return fmt.Errorf("parsing target '%s': doc field must be a string", target.Name)
		}
		target.Doc = doc
	}
	return nil
}

// Parse target dependencies
func ParseTargetDepends(object util.Object, target *Target) error {
	if object.HasField("depends") {
		depends, err := object.GetListStringsOrString("depends")
		if err != nil {
			return fmt.Errorf("parsing target '%s': depends field must be a string or list of strings", target.Name)
		}
		target.Depends = depends
	}
	return nil
}

// Parse target steps
func ParseTargetSteps(object util.Object, target *Target) error {
	if object.HasField("steps") {
		list, err := object.GetList("steps")
		if err != nil {
			return fmt.Errorf("parsig target '%s': steps must be a list", target.Name)
		}
		var steps []Step
		for index, object := range list {
			step, err := NewStep(target, object)
			if err != nil {
				return fmt.Errorf("in step %d: %v", index+1, err)
			}
			steps = append(steps, step)
		}
		target.Steps = steps
	}
	return nil
}

// Run target
func (target *Target) Run() error {
	target.Build.Stack.Push(target.Name)
	for _, name := range target.Depends {
		if !target.Build.Stack.Contains(name) {
			err := target.Build.RunTarget(name)
			if err != nil {
				return err
			}
		}
	}
	util.PrintColor("%s", util.Yellow("Running target "+target.Name))
	err := os.Chdir(target.Build.Dir)
	if err != nil {
		return fmt.Errorf("changing to build directory '%s'", target.Build.Dir)
	}
	target.Build.Index = NewIndex()
	for index, step := range target.Steps {
		target.Build.Index.Set(index)
		err := step.Run()
		if err != nil {
			return fmt.Errorf("in step %s: %v", target.Build.Index.String(), err)
		}
	}
	return nil
}

// Run target steps
func (target *Target) RunSteps() error {
	target.Build.Stack.Push(target.Name)
	target.Build.Index = NewIndex()
	for index, step := range target.Steps {
		target.Build.Index.Set(index)
		err := step.Run()
		if err != nil {
			return fmt.Errorf("in step %s: %v", target.Build.Index.String(), err)
		}
	}
	return nil
}
