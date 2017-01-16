package build

import (
	"fmt"
	"neon/util"
	"os"
	"reflect"
)

const (
	DEFAULT_FILE_MODE = 0777
)

type Task func() error

type Constructor func(target *Target, args util.Object) (Task, error)

var tasksMap map[string]Constructor

func init() {
	tasksMap = map[string]Constructor{
		"go":     Go,
		"print":  Print,
		"rm":     Rm,
		"cd":     Cd,
		"mkdir":  MkDir,
		"delete": Delete,
		"if":     If,
		"for":    For,
		"while":  While,
		"try":    Try,
		"nop":    Nop,
	}
}

// TASKS DEFINITIONS

func Go(target *Target, args util.Object) (Task, error) {
	source, ok := args["go"].(string)
	if !ok {
		return nil, fmt.Errorf("argument of task go must be a string")
	}
	return func() error {
		_, err := target.Build.Context.Evaluate(source)
		if err != nil {
			return fmt.Errorf("evaluating go source: %v", err)
		}
		return nil
	}, nil
}

func Print(target *Target, args util.Object) (Task, error) {
	message, ok := args["print"].(string)
	if !ok {
		return nil, fmt.Errorf("argument of task print must be a string")
	}
	return func() error {
		evaluated, err := target.Build.Context.ReplaceProperties(message)
		if err != nil {
			return fmt.Errorf("processing print argument: %v", err)
		}
		fmt.Println(evaluated)
		return nil
	}, nil
}

func Rm(target *Target, args util.Object) (Task, error) {
	file, ok := args["rm"].(string)
	if !ok {
		return nil, fmt.Errorf("argument to task rm must be a string")
	}
	return func() error {
		file, err := target.Build.Context.ReplaceProperties(file)
		fmt.Printf("Removing file '%s'\n", file)
		if err != nil {
			return fmt.Errorf("processing rm argument: %v", err)
		}
		err = os.Remove(file)
		if err != nil {
			return fmt.Errorf("removing file '%s': %s", file, err)
		}
		return nil
	}, nil
}

func Cd(target *Target, args util.Object) (Task, error) {
	dir, ok := args["cd"].(string)
	if !ok {
		return nil, fmt.Errorf("argument to task cd must be a string")
	}
	return func() error {
		directory, err := target.Build.Context.ReplaceProperties(dir)
		fmt.Printf("Changing to directory '%s'\n", directory)
		if err != nil {
			return fmt.Errorf("processing cd argument: %v", err)
		}
		err = os.Chdir(directory)
		if err != nil {
			return fmt.Errorf("changing to directory '%s': %s", directory, err)
		}
		return nil
	}, nil
}

func MkDir(target *Target, args util.Object) (Task, error) {
	dir, ok := args["mkdir"].(string)
	if !ok {
		return nil, fmt.Errorf("argument to task mkdir must be a string")
	}
	return func() error {
		directory, err := target.Build.Context.ReplaceProperties(dir)
		fmt.Printf("Making directory '%s'\n", directory)
		if err != nil {
			return fmt.Errorf("processing mkdir argument: %v", err)
		}
		err = os.MkdirAll(directory, DEFAULT_FILE_MODE)
		if err != nil {
			return fmt.Errorf("making directory '%s': %s", directory, err)
		}
		return nil
	}, nil
}

func Delete(target *Target, args util.Object) (Task, error) {
	directories, err := args.GetListStringsOrString("delete")
	if err != nil {
		return nil, fmt.Errorf("delete argument must be string or list of strings")
	}
	return func() error {
		for _, dir := range directories {
			directory, err := target.Build.Context.ReplaceProperties(dir)
			if err != nil {
				return fmt.Errorf("evaluating directory in task delete: %v", err)
			}
			if _, err := os.Stat(directory); err == nil {
				fmt.Printf("Deleting directory '%s'\n", directory)
				err = os.RemoveAll(directory)
				if err != nil {
					return fmt.Errorf("deleting directory '%s': %v", directory, err)
				}
			}
		}
		return nil
	}, nil
}

func If(target *Target, args util.Object) (Task, error) {
	fields := args.Fields()
	if err := FieldsInList(fields, []string{"if", "then", "else"}); err != nil {
		return nil, fmt.Errorf("building if condition: %v", err)
	}
	if err := FieldsMandatory(fields, []string{"if", "then"}); err != nil {
		return nil, fmt.Errorf("building if condition: %v", err)
	}
	condition, err := args.GetString("if")
	if err != nil {
		return nil, fmt.Errorf("evaluating if construct: %v", err)
	}
	thenSteps, err := ParseSteps(target, args, "then")
	if err != nil {
		return nil, err
	}
	var elseSteps []Step
	if FieldInList("else", fields) {
		elseSteps, err = ParseSteps(target, args, "else")
		if err != nil {
			return nil, err
		}
	}
	return func() error {
		result, err := target.Build.Context.Evaluate(condition)
		if err != nil {
			return fmt.Errorf("evaluating 'if' condition: %v", err)
		}
		boolean, ok := result.(bool)
		if !ok {
			return fmt.Errorf("evaluating if condition: must return a bool")
		}
		if boolean {
			err := RunSteps(thenSteps)
			if err != nil {
				return err
			}
		} else {
			err := RunSteps(elseSteps)
			if err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func For(target *Target, args util.Object) (Task, error) {
	fields := args.Fields()
	if err := FieldsInList(fields, []string{"for", "in", "do"}); err != nil {
		return nil, fmt.Errorf("building 'for' loop: %v", err)
	}
	if err := FieldsMandatory(fields, []string{"for", "in", "do"}); err != nil {
		return nil, fmt.Errorf("building 'for' loop: %v", err)
	}
	variable, err := args.GetString("for")
	if err != nil {
		return nil, fmt.Errorf("'for' field of a 'for' loop must be a string")
	}
	list, err := args.GetList("in")
	expression := ""
	if err != nil {
		expression, err = args.GetString("in")
		if err != nil {
			return nil, fmt.Errorf("'in' field of 'for' loop must be a list or string")
		}
	}
	steps, err := ParseSteps(target, args, "do")
	if err != nil {
		return nil, err
	}
	return func() error {
		if expression != "" {
			result, err := target.Build.Context.Evaluate(expression)
			if err != nil {
				return fmt.Errorf("evaluating in field of for loop: %v", err)
			}
			list, err = ToList(result)
			if err != nil {
				return fmt.Errorf("'in' field of 'for' loop must be an expression that returns a list")
			}
		}
		for _, value := range list {
			target.Build.Context.SetProperty(variable, value)
			if err != nil {
				return err
			}
			err := RunSteps(steps)
			if err != nil {
				return err
			}
		}
		return nil
	}, nil

}

func While(target *Target, args util.Object) (Task, error) {
	fields := args.Fields()
	if err := FieldsInList(fields, []string{"while", "do"}); err != nil {
		return nil, fmt.Errorf("building 'while' loop: %v", err)
	}
	if err := FieldsMandatory(fields, []string{"while", "do"}); err != nil {
		return nil, fmt.Errorf("building 'while' loop: %v", err)
	}
	condition, err := args.GetString("while")
	if err != nil {
		return nil, fmt.Errorf("'while' field of a 'while' loop must be a string")
	}
	steps, err := ParseSteps(target, args, "do")
	if err != nil {
		return nil, err
	}
	return func() error {
		for {
			result, err := target.Build.Context.Evaluate(condition)
			if err != nil {
				return fmt.Errorf("evaluating 'while' field of 'while' loop: %v", err)
			}
			loop, ok := result.(bool)
			if !ok {
				return fmt.Errorf("evaluating 'while' condition: must return a bool")
			}
			if !loop {
				break
			}
			err = RunSteps(steps)
			if err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func Try(target *Target, args util.Object) (Task, error) {
	fields := args.Fields()
	if err := FieldsInList(fields, []string{"try", "catch"}); err != nil {
		return nil, fmt.Errorf("building try construct: %v", err)
	}
	if err := FieldsMandatory(fields, []string{"try", "catch"}); err != nil {
		return nil, fmt.Errorf("building try construct: %v", err)
	}
	trySteps, err := ParseSteps(target, args, "try")
	if err != nil {
		return nil, err
	}
	catchSteps, err := ParseSteps(target, args, "catch")
	if err != nil {
		return nil, err
	}
	return func() error {
		err := RunSteps(trySteps)
		if err != nil {
			target.Build.Context.SetProperty("error", err.Error())
			err = RunSteps(catchSteps)
			if err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func Nop(target *Target, args util.Object) (Task, error) {
	fields := args.Fields()
	if err := FieldsInList(fields, []string{"nop"}); err != nil {
		return nil, fmt.Errorf("building nop instruction: %v", err)
	}
	if err := FieldsMandatory(fields, []string{"nop"}); err != nil {
		return nil, fmt.Errorf("building nop instruction: %v", err)
	}
	return func() error {
		return nil
	}, nil
}

// UTILITY FUNCTIONS

func FieldsInList(fields, list []string) error {
	for _, field := range fields {
		found := false
		for _, e := range list {
			if e == field {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown field '%s'", field)
		}
	}
	return nil
}

func FieldInList(field string, list []string) bool {
	for _, f := range list {
		if f == field {
			return true
		}
	}
	return false
}

func FieldsMandatory(fields, mandatory []string) error {
	for _, manda := range mandatory {
		found := false
		for _, field := range fields {
			if manda == field {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("mandatory field '%s' not found", manda)
		}
	}
	return nil
}

func ParseSteps(target *Target, object util.Object, field string) ([]Step, error) {
	list, err := object.GetList(field)
	if err != nil {
		return nil, err
	}
	var steps []Step
	for index, element := range list {
		target.Build.Log("Parsing step %v in %s field", index, field)
		step, err := NewStep(target, element)
		if err != nil {
			return nil, fmt.Errorf("parsing target '%s': %v", target.Name, err)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func RunSteps(steps []Step) error {
	for _, step := range steps {
		err := step.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func ToList(object interface{}) ([]interface{}, error) {
	slice := reflect.ValueOf(object)
	if slice.Kind() == reflect.Slice {
		result := make([]interface{}, slice.Len())
		for i := 0; i < slice.Len(); i++ {
			result[i] = slice.Index(i).Interface()
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("must be a list")
	}
}