package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// CheckIfError should be used to naively panics if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}
	err = errors.WithStack(err)
	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %+v", err))
	os.Exit(1)
}

// Info should be used to describe the example commands that are about to run.
func Info(format string, args ...interface{}) {
	fmt.Printf("\x1b[34;1m%s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// Warning should be used to display a warning
func Warning(format string, args ...interface{}) {
	fmt.Printf("\x1b[36;1m%s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// checkExist check file or path exists
func checkExist(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// read context from `BuilderFile`
func mapBuilderFile(filePath string, BuildOpts *FileBuild) error {
	if checkExist(filePath) {
		f, err := os.Open(filePath)
		CheckIfError(err)
		defer f.Close()
		r := bufio.NewReader(f)
		t := reflect.TypeOf(BuildOpts)
		v := reflect.ValueOf(BuildOpts)
		for {
			buf, err := r.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				CheckIfError(err)
			}
			line := string(buf)
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			if strings.Contains(line, "=") {
				firstIndex := strings.Index(line, "=")
				if firstIndex == -1 {
					return errors.New(fmt.Sprintf("file: %s Formate Error.", filePath))
				}
				key := line[:firstIndex]
				val := strings.TrimSpace(line[firstIndex+1:])
				for i := 0; i < t.Elem().NumField(); i++ {
					field := t.Elem().Field(i)
					if key == field.Tag.Get("file") {
						v.Elem().Field(i).Set(reflect.ValueOf(val))
					}
				}
			}
		}
	} else {
		return errors.New(fmt.Sprintf("file: %s Not Found.", filePath))
	}
	return nil
}

// read context from `DockerFile`
func mapDockerFile(filePath string, BuildOpts *FileBuild) error {
	if checkExist(filePath) {
		f, err := os.Open(filePath)
		CheckIfError(err)
		defer f.Close()
		r := bufio.NewReader(f)
		t := reflect.TypeOf(BuildOpts)
		v := reflect.ValueOf(BuildOpts)
		c := make([]string, 0, 10)
		for {
			buf, err := r.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				CheckIfError(err)
			}
			line := string(buf)
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			if strings.HasPrefix(line, "ADD") || strings.HasPrefix(line, "COPY") {
				val := strings.Fields(line)
				if len(val) == 3 {
					c = append(c, strings.TrimSuffix(val[1], "*.jar"))
				}
				for i := 0; i < t.Elem().NumField(); i++ {
					field := t.Elem().Field(i)
					if field.Tag.Get("file") == "Dockerfile" {
						v.Elem().Field(i).Set(reflect.ValueOf(c))
					}
				}
			}
		}
	} else {
		return errors.New(fmt.Sprintf("file: %s Not Found.", filePath))
	}
	return nil
}

func listDir(pathname string) []string {
	dstTarget := []string{}
	err := filepath.Walk(pathname, func(src string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		dstTarget = append(dstTarget, src)
		return nil
	})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
		return nil
	}
	return dstTarget
}

// check all struct var is not nil
func checkAll() error {
	// check var
	err := checkVar(Conf.Env)
	if err != nil {
		return err
	}
	err = checkVar(Conf.File)
	if err != nil {
		return err
	}
	// check path
	if !checkExist(Conf.Path.Workdir) {
		return errors.New(fmt.Sprintf("path not found: %s", Conf.Path.Workdir))
	}
	if !checkExist(path.Join(Conf.Path.Runtime, Conf.Env.AppName)) {
		return errors.New(fmt.Sprintf("path not found: %s", path.Join(Conf.Path.Runtime, Conf.Env.AppName)))
	}
	return nil
}

// usage with checkAll function
func checkVar(any interface{}) error {
	t := reflect.TypeOf(any)
	v := reflect.ValueOf(any)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for {
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				if len(v.FieldByName(field.Name).String()) == 0 {
					return errors.New(fmt.Sprintf("Variable: [%s] not set, pls check it !", field.Name))
				}
			}
		}
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		} else {
			break
		}
	}
	return nil
}

// make string compile faster
func buildString(str ...string) string {
	var build strings.Builder
	for _, s := range str {
		build.WriteString(s)
	}
	return build.String()
}
