package task

import (
	"fmt"
	"neon/build"
	"neon/util"
	"os"
)

func init() {
	build.TaskMap["link"] = build.TaskDescriptor{
		Constructor: Link,
		Help:        "Create a symbolic link",
	}
}

func Link(target *build.Target, args util.Object) (build.Task, error) {
	fields := []string{"link", "to"}
	if err := CheckFields(args, fields, fields); err != nil {
		return nil, err
	}
	s, err := args.GetString("link")
	if err != nil {
		return nil, fmt.Errorf("argument link must be a string")
	}
	d, err := args.GetString("to")
	if err != nil {
		return nil, fmt.Errorf("argument to of task link must be a string")
	}
	return func() error {
		source, err := target.Build.Context.ReplaceProperties(s)
		if err != nil {
			return fmt.Errorf("processing link argument: %v", err)
		}
		dest, err := target.Build.Context.ReplaceProperties(d)
		if err != nil {
			return fmt.Errorf("processing to argument of link task: %v", err)
		}
		fmt.Printf("Linking file '%s' to '%s'\n", source, dest)
		err = os.Symlink(source, dest)
		if err != nil {
			return fmt.Errorf("linking files: %v", err)
		}
		return nil
	}, nil
}