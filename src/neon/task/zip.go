package task

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"neon/build"
	"neon/util"
	"os"
	"path/filepath"
)

func init() {
	build.TaskMap["zip"] = build.TaskDescriptor{
		Constructor: Zip,
		Help: `Create a Zip archive.

Arguments:

- zip: the list of globs of files to zip (as a string or list of strings).
- dir: the root directory for glob (as a string, optional).
- exclude: globs of files to exclude (as a string or list of strings,
  optional).
- tofile: the name of the Zip file to create as a string.
- prefix: prefix directory in the archive.

Examples:

    # zip files in build directory in file named build.zip
    - zip: "build/**/*"
      tofile: "build.zip"`,
	}
}

func Zip(target *build.Target, args util.Object) (build.Task, error) {
	fields := []string{"zip", "tofile", "dir", "exclude", "prefix"}
	if err := CheckFields(args, fields, fields[:2]); err != nil {
		return nil, err
	}
	includes, err := args.GetListStringsOrString("zip")
	if err != nil {
		return nil, fmt.Errorf("argument zip must be a string or list of strings")
	}
	var tofile string
	if args.HasField("tofile") {
		tofile, err = args.GetString("tofile")
		if err != nil {
			return nil, fmt.Errorf("argument to of task zip must be a string")
		}
	}
	var dir string
	if args.HasField("dir") {
		dir, err = args.GetString("dir")
		if err != nil {
			return nil, fmt.Errorf("argument dir of task zip must be a string")
		}
	}
	var excludes []string
	if args.HasField("exclude") {
		excludes, err = args.GetListStringsOrString("exclude")
		if err != nil {
			return nil, fmt.Errorf("argument exclude must be string or list of strings")
		}
	}
	var prefix string
	if args.HasField("prefix") {
		prefix, err = args.GetString("prefix")
		if err != nil {
			return nil, fmt.Errorf("argument prefix of task zip must be a string")
		}
	}
	return func() error {
		// evaluate arguments
		for index, pattern := range includes {
			eval, err := target.Build.Context.ReplaceProperties(pattern)
			if err != nil {
				return fmt.Errorf("evaluating pattern: %v", err)
			}
			includes[index] = eval
		}
		eval, err := target.Build.Context.ReplaceProperties(tofile)
		if err != nil {
			return fmt.Errorf("evaluating destination file: %v", err)
		}
		tofile = eval
		eval, err = target.Build.Context.ReplaceProperties(dir)
		if err != nil {
			return fmt.Errorf("evaluating source directory: %v", err)
		}
		dir = eval
		eval, err = target.Build.Context.ReplaceProperties(prefix)
		if err != nil {
			return fmt.Errorf("evaluating destination file: %v", err)
		}
		prefix = eval
		// find source files
		files, err := target.Build.Context.FindFiles(dir, includes, excludes)
		if err != nil {
			return fmt.Errorf("getting source files for zip task: %v", err)
		}
		if len(files) > 0 {
			build.Info("Zipping %d file(s) in '%s'", len(files), tofile)
			// zip files
			err = WriteZip(dir, files, prefix, tofile)
			if err != nil {
				return fmt.Errorf("zipping files: %v", err)
			}
		}
		return nil
	}, nil
}

func WriteZip(dir string, files []string, prefix, to string) error {
	archive, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("creating zip archive: %v", err)
	}
	defer archive.Close()
	zipper := zip.NewWriter(archive)
	zipper.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	defer zipper.Close()
	for _, file := range files {
		var path string
		if dir != "" {
			path = filepath.Join(dir, file)
		} else {
			path = file
		}
		err := writeFileToZip(zipper, path, file, prefix)
		if err != nil {
			return fmt.Errorf("writing file to zip archive: %v", err)
		}
	}
	return nil
}

func writeFileToZip(zipper *zip.Writer, path, name, prefix string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	name = sanitizedName(name)
	if prefix != "" {
		name = prefix + "/" + name
	}
	header.Name = name
	writer, err := zipper.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}
