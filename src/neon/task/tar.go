package task

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"neon/build"
	"neon/util"
	"os"
	"strings"
)

func init() {
	build.TaskMap["tar"] = build.TaskDescriptor{
		Constructor: Tar,
		Help: `Create a tar archive.

Arguments:
- tar: the list of globs of files to tar (as a string or list of strings).
- dir: the root directory for glob (as a string, optional).
- exclude: globs of files to exclude (as a string or list of strings,
  optional).
- to: the name of the tar file to create as a string.
- prefix: prefix directory in the archive.

Examples:
# tar files in build directory in file named build.tar.gz
- tar: "build/**/*"
  to: "build.tar.gz"

Notes:
- If archive filename ends with gz (with a name such as foo.tar.gz or foo.tgz)
  the tar archive is compressed with gzip.`,
	}
}

func Tar(target *build.Target, args util.Object) (build.Task, error) {
	fields := []string{"tar", "to", "dir", "exclude", "prefix"}
	if err := CheckFields(args, fields, fields[:2]); err != nil {
		return nil, err
	}
	includes, err := args.GetListStringsOrString("tar")
	if err != nil {
		return nil, fmt.Errorf("argument tar must be a string or list of strings")
	}
	var to string
	if args.HasField("to") {
		to, err = args.GetString("to")
		if err != nil {
			return nil, fmt.Errorf("argument to of task tar must be a string")
		}
	}
	var dir string
	if args.HasField("dir") {
		dir, err = args.GetString("dir")
		if err != nil {
			return nil, fmt.Errorf("argument dir of task tar must be a string")
		}
	}
	var excludes []string
	if args.HasField("exclude") {
		excludes, err = args.GetListStringsOrString("exclude")
		if err != nil {
			return nil, fmt.Errorf("argument exclude of task tar must be string or list of strings")
		}
	}
	var prefix string
	if args.HasField("prefix") {
		prefix, err = args.GetString("prefix")
		if err != nil {
			return nil, fmt.Errorf("argument prefix of task tar must be a string")
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
		eval, err := target.Build.Context.ReplaceProperties(to)
		if err != nil {
			return fmt.Errorf("evaluating destination file: %v", err)
		}
		to = eval
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
			return fmt.Errorf("getting source files for tar task: %v", err)
		}
		if len(files) > 0 {
			target.Build.Info("Tarring %d file(s)", len(files))
			// tar files
			err = Writetar(files, prefix, to)
			if err != nil {
				return fmt.Errorf("tarring files: %v", err)
			}
		}
		return nil
	}, nil
}

func Writetar(files []string, prefix, to string) error {
	file, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("creating tar archive: %v", err)
	}
	defer file.Close()
	var fileWriter io.WriteCloser = file
	if strings.HasSuffix(to, "gz") {
		fileWriter = gzip.NewWriter(file)
		defer fileWriter.Close()
	}
	writer := tar.NewWriter(fileWriter)
	defer writer.Close()
	for _, name := range files {
		err := writeFileToTar(writer, name, prefix)
		if err != nil {
			return fmt.Errorf("writing file to tar archive: %v", err)
		}
	}
	return nil
}

func writeFileToTar(writer *tar.Writer, filename, prefix string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	name := sanitizedName(filename)
	if prefix != "" {
		name = prefix + "/" + name
	}
	header := &tar.Header{
		Name:    name,
		Mode:    int64(stat.Mode()),
		Uid:     os.Getuid(),
		Gid:     os.Getgid(),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
	}
	if err = writer.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}