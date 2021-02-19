package main

import "github.com/olonglongo/rundeck-deploy/utils"

func main() {

	// -1. clean workspace
	utils.Conf.Git.Clean()
	// 1. git clone
	utils.Conf.Git.Clone()
	commit, _ := utils.Conf.Git.GetHead()
	utils.Info(commit)
	// 3. 实例化客户端
	d := utils.NewDockerClient()
	// 4. 构建程序
	d.Compile()
	// 5. 打包程序
	d.Build()
	// 6. 实例化客户端
	k := utils.NewKubeClient()
	// 7. 更新镜像
	k.Update()
}
