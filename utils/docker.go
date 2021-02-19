package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"github.com/moby/term"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
)

// var (
// 	workDir    string
// 	mountDir   []mount.Mount
// 	compileCmd []string
// )

// DockerClient Cli
type DockerClient struct {
	Username string
	Password string
	Address  string
	Auth     string
	Ctx      context.Context
	Cli      *client.Client
}

// NewDockerClient newCli
func NewDockerClient() *DockerClient {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	CheckIfError(err)
	jsonBytes, err := json.Marshal(map[string]string{
		"username": Conf.Harbor.User,
		"password": Conf.Harbor.Pass,
	})
	CheckIfError(err)
	return &DockerClient{
		Auth: base64.StdEncoding.EncodeToString(jsonBytes),
		Ctx:  context.Background(),
		Cli:  cli,
	}
}

// Compile such as `mvn clean / yarn install`
func (d *DockerClient) Compile() {
	switch strings.ToLower(Conf.Env.CodeType) {
	case "php", "web":
		return
	case "vue":
		workDir := "/data"
		mountDir := []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: path.Join(Conf.Path.Workdir, Conf.Env.AppName),
				Target: "/data",
			},
			{
				Type:   mount.TypeBind,
				Source: path.Join(Conf.Path.Workdir, ".cache"),
				Target: "/usr/local/share/.cache",
			},
		}
		buildOpts := strings.Replace(Conf.File.BuilderOpts, "{{mode}}", os.Getenv("RD_OPTION_MODE"), -1)
		d.create("yarn install", workDir, mountDir)
		time.Sleep(time.Duration(5) * time.Second)
		d.create(buildOpts, workDir, mountDir)
	default:
		// Java maven build
		workDir := "/build/code"
		mountDir := []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: path.Join(Conf.Path.Workdir, Conf.Env.AppName),
				Target: "/build/code",
			},
			{
				Type:   mount.TypeBind,
				Source: path.Join(Conf.Path.Workdir, ".m2"),
				Target: "/build/.m2",
			},
		}
		d.create(Conf.File.BuilderOpts, workDir, mountDir)
	}
}

func (d *DockerClient) create(cmd, workDir string, mountDir []mount.Mount) {
	// // pull non images
	// reader, err := d.Cli.ImagePull(d.Ctx, Conf.File.BuilderImage, types.ImagePullOptions{
	// 	RegistryAuth: d.Auth,
	// })
	// CheckIfError(err)
	// outPutTerm(reader)
	command := strings.ReplaceAll(strings.TrimSpace(cmd), "\"", "")
	resp, err := d.Cli.ContainerCreate(d.Ctx, &container.Config{
		Image:      Conf.File.BuilderImage,
		Cmd:        strings.Fields(strings.ReplaceAll(command, "'", "")),
		Tty:        false,
		WorkingDir: workDir,
	}, &container.HostConfig{
		AutoRemove: true,
		Mounts:     mountDir,
	}, nil, nil, Conf.Env.AppName)
	CheckIfError(err)

	if err := d.Cli.ContainerStart(d.Ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		CheckIfError(err)
	}

	reader, err := d.Cli.ContainerLogs(d.Ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	CheckIfError(err)
	outPutTerm(reader)
	go stdcopy.StdCopy(os.Stdout, os.Stderr, reader)

	statusCh, errCh := d.Cli.ContainerWait(d.Ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			CheckIfError(err)
		}
	case retCode := <-statusCh:
		if retCode.StatusCode != 0 {
			os.Exit(int(retCode.StatusCode))
		}
	}
}

// Build such as `docker build`
func (d *DockerClient) Build() {
	includeFiles := make([]string, 0, 50)
	switch strings.ToLower(Conf.Env.CodeType) {
	case "php", "vue":
		for _, fileName := range listDir(path.Join(Conf.Path.Workdir, Conf.Env.AppName)) {
			jarFile := strings.ReplaceAll(fileName, path.Join(Conf.Path.Workdir, Conf.Env.AppName), ".")
			includeFiles = append(includeFiles, jarFile)
		}
		includeFiles = append(includeFiles, "Dockerfile")
		includeFiles = append(includeFiles, Conf.File.CopyFiles...)
	default:
		Info("Build Context:")
		for _, fileName := range listDir(path.Join(Conf.Path.Workdir, Conf.Env.AppName)) {
			if strings.HasSuffix(fileName, ".jar") {
				jarFile := strings.ReplaceAll(fileName, path.Join(Conf.Path.Workdir, Conf.Env.AppName), ".")
				includeFiles = append(includeFiles, jarFile)
			}
		}
		includeFiles = append(includeFiles, "Dockerfile")
		includeFiles = append(includeFiles, Conf.File.CopyFiles...)
		for _, file := range includeFiles {
			fmt.Println(file)
		}
	}

	// build image
	Info(buildString("Build Image: ", ImageName))
	tarOptions := archive.TarOptions{
		Compression:  archive.Uncompressed,
		IncludeFiles: includeFiles,
		RebaseNames:  map[string]string{path.Join(Conf.Path.Workdir, Conf.Env.AppName): "."}}
	buildContext, err := archive.TarWithOptions(path.Join(Conf.Path.Workdir, Conf.Env.AppName), &tarOptions)
	CheckIfError(err)
	defer buildContext.Close()
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{ImageName},
	}
	buildResponse, err := d.Cli.ImageBuild(d.Ctx, buildContext, buildOptions)
	CheckIfError(err)
	defer buildResponse.Body.Close()
	termFd, isTerm := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(buildResponse.Body, os.Stdout, termFd, isTerm, nil)
	CheckIfError(err)

	// PUSH
	Info(buildString("Push Image: ", ImageName))
	out, err := d.Cli.ImagePush(d.Ctx, ImageName, types.ImagePushOptions{
		RegistryAuth: d.Auth,
	})
	CheckIfError(err)
	outPutTerm(out)
	// Delete images
	_, err = d.Cli.ImageRemove(d.Ctx, ImageName, types.ImageRemoveOptions{})
	CheckIfError(err)
}

// outPutTerm .
func outPutTerm(buf io.ReadCloser) {
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(buf, os.Stderr, termFd, isTerm, nil)
}
