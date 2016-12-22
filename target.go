package main

import (
	"fmt"
	"github.com/fatih/color"
)

type Target struct {
	Name    string
	Build   *Build
	Depends []string
	Steps   []string
}

func (target *Target) Init(build *Build, name string) {
	target.Build = build
	target.Name = name
}

func (target *Target) Run() {
	for _, depend := range target.Depends {
		dependency := target.Build.Target(depend)
		dependency.Run()
	}
	color.Yellow("Running target " + target.Name)
	for _, step := range target.Steps {
		output := target.Build.Execute(step)
		fmt.Println(output)
	}
}
