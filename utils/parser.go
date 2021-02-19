package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"gopkg.in/ini.v1"
	"github.com/olonglongo/rundeck-deploy/asset"
)

var (
	// Conf Main (other var from init)
	Conf = Config{
		Env: &EnvVar{
			File:     strings.TrimSpace("/opt/.env"),
			Project:  strings.TrimSpace(os.Getenv("RD_JOB_PROJECT")),
			AppName:  strings.TrimSpace(os.Getenv("RD_OPTION_APP_NAME")),
			EnvName:  strings.TrimSpace(os.Getenv("RD_OPTION_ENV_NAME")),
			GitVer:   strings.TrimSpace(os.Getenv("RD_OPTION_GIT_VERSION")),
			CodeType: strings.TrimSpace(os.Getenv("RD_OPTION_CODE_TYPE")),
		},
		Git:    new(IniGit),
		Kube:   new(IniK8s),
		Path:   new(IniPath),
		Harbor: new(IniHarbor),
		File:   new(FileBuild),
	}
	// ImageName build image
	ImageName string
)

// Config Main struct
type Config struct {
	Env    *EnvVar
	Git    *IniGit
	Kube   *IniK8s
	Path   *IniPath
	Harbor *IniHarbor
	File   *FileBuild
}

// EnvVar env struct
type EnvVar struct {
	File     string // environment file
	Project  string // project name
	AppName  string // application name
	EnvName  string // environment name
	GitVer   string // code version
	CodeType string // code build type
}

// IniPath path struct
type IniPath struct {
	Runtime string `ini:"runtime"` // runtime dir
	Workdir string `ini:"workdir"` // workspace dir
}

// IniHarbor harbor struct
type IniHarbor struct {
	User string `ini:"user"` // user name
	Pass string `ini:"pass"` // user pass
	Addr string `ini:"addr"` // harbor addr
	Dist string `ini:"dist"` // harbor dist
}

// IniGit git struct
type IniGit struct {
	PrivateKey string `ini:"id_rsa"` // code info
}

// IniK8s k8s struct
type IniK8s struct {
	KubeConfig string `ini:"config"` // kube config
}

// FileBuild file struct
type FileBuild struct {
	GitAddr      string   `file:"GIT_ADDRESS"`  // code addr
	BuilderImage string   `file:"BUILDER"`      // build image
	BuilderOpts  string   `file:"BUILDER_OPTS"` // build opts
	CopyFiles    []string `file:"Dockerfile"`   // `COPY`/`ADD` params
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// read env
	env, err := ioutil.ReadFile(Conf.Env.File)
	CheckIfError(err)
	conf := path.Join("conf", strings.TrimSpace(string(env)), Conf.Env.Project+".ini")
	// read asset
	config, err := asset.Asset(conf)
	CheckIfError(err)
	cfg, err := ini.Load(config)
	CheckIfError(err)
	// map config to struct
	err = cfg.Section("path").MapTo(&Conf.Path)
	CheckIfError(err)
	err = cfg.Section("git").MapTo(&Conf.Git)
	CheckIfError(err)
	err = cfg.Section("k8s").MapTo(&Conf.Kube)
	CheckIfError(err)
	err = cfg.Section("harbor").MapTo(&Conf.Harbor)
	CheckIfError(err)
	// map file
	appDir := path.Join(Conf.Path.Runtime, Conf.Env.AppName)
	err = mapDockerFile(path.Join(appDir, "Dockerfile"), Conf.File)
	CheckIfError(err)
	err = mapBuilderFile(path.Join(appDir, "Builderfile"), Conf.File)
	CheckIfError(err)
	// check all env
	err = checkAll()
	CheckIfError(err)
}
