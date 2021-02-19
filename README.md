# RundeckDeploy(go)

## tree
```
utils
├── common.go // 工具函数
├── docker.go // docker 连接库
├── kubernetes.go // k8s 连接库
├── parser.go // 初始化变量
└── phabricator.go // git 连接库
conf 
├── prod // 配置环境
│   └── turing.ini // 项目名
└── test
    └── test.ini // 项目名
```
    
## 部署流程
1. 接收环境变量
2. 克隆代码至相应目录
3. 容器构建
4. 容器打包
5. 推送至镜像仓库
6. 配置k8s应用状态
7. 检测k8s应用状态
8. 部署完成
