package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/otiai10/copy"
)

// Clone git clone
func (g *IniGit) Clone() {
	buildSpace := path.Join(Conf.Path.Runtime, Conf.Env.AppName)
	workSpace := path.Join(Conf.Path.Workdir, Conf.Env.AppName)
	if !checkExist(g.PrivateKey) {
		log.Fatal("privatekey not found.")
	}
	sshKey, err := ioutil.ReadFile(g.PrivateKey)
	CheckIfError(err)
	publicKey, err := ssh.NewPublicKeys("git", []byte(sshKey), "")
	CheckIfError(err)

	var remote = git.NewRemote(
		filesystem.NewStorage(
			osfs.New(workSpace),
			cache.NewObjectLRUDefault(),
		),
		&config.RemoteConfig{
			URLs: []string{Conf.File.GitAddr},
		},
	)
	refs, err := remote.List(&git.ListOptions{Auth: publicKey})
	CheckIfError(err)
	var ref = plumbing.NewBranchReferenceName(Conf.Env.GitVer)
	if !checkReferenceName(ref, refs) {
		ref = plumbing.NewTagReferenceName(Conf.Env.GitVer)
		if !checkReferenceName(ref, refs) {
			log.Fatalf(fmt.Sprintf("repo remote not exists: %s", Conf.Env.GitVer))
		}
	}
	if !checkExist(workSpace) {
		_, err = git.PlainClone(workSpace, false, &git.CloneOptions{
			URL:               Conf.File.GitAddr,
			Auth:              publicKey,
			SingleBranch:      true,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			// Progress:          os.Stdout,
			ReferenceName: ref,
		})
		CheckIfError(err)
		_, commit := Conf.Git.GetHead()
		timestamp := time.Now().Unix()
		image := buildString(Conf.Harbor.Addr, "/", Conf.Harbor.Dist, "/", Conf.Env.AppName, ":", Conf.Env.GitVer, "-", commit, "-", strconv.FormatInt(timestamp, 10))
		ImageName = strings.ToLower(image)
		// sync runtime -> experiment
		opt := copy.Options{
			Skip: func(src string) (bool, error) {
				return strings.HasSuffix(src, ".git"), nil
			},
		}
		err := copy.Copy(buildSpace, workSpace, opt)
		CheckIfError(err)
	} else {
		log.Fatal(fmt.Sprintf("target dir exists: %s", workSpace))
	}
}

// GetHead git version
func (g *IniGit) GetHead() (commit, short string) {
	gitRepo, err := git.PlainOpen(path.Join(Conf.Path.Workdir, Conf.Env.AppName))
	CheckIfError(err)
	head, err := gitRepo.Head()
	CheckIfError(err)
	headCommit, err := gitRepo.CommitObject(head.Hash())
	return headCommit.String(), head.Hash().String()[:9]
}

// Clean clean workspace
func (g *IniGit) Clean() {
	workSpace := path.Join(Conf.Path.Workdir, Conf.Env.AppName)
	if checkExist(workSpace) {
		if strings.Contains(workSpace, "experiment") {
			os.RemoveAll(workSpace)
			Info("Clean workSpace")
		}
	}
}

func checkReferenceName(ref plumbing.ReferenceName, refs []*plumbing.Reference) bool {
	for _, ref2 := range refs {
		if ref.String() == ref2.Name().String() {
			return true
		}
	}
	return false
}
