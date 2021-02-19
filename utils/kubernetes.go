package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	app_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient Cli
type KubeClient struct {
	NameSpace string
	AppName   string
	ImageName string
	Ctx       context.Context
	Cli       *kubernetes.Clientset
}

// NewKubeClient newCli
func NewKubeClient() *KubeClient {
	config, err := clientcmd.BuildConfigFromFlags("", Conf.Kube.KubeConfig)
	CheckIfError(err)
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	CheckIfError(err)
	return &KubeClient{
		NameSpace: Conf.Env.EnvName,
		AppName:   Conf.Env.AppName,
		ImageName: ImageName,
		Ctx:       context.Background(),
		Cli:       clientset,
	}
}

// Update .
func (k KubeClient) Update() {
	deployClient := k.Cli.AppsV1().Deployments(k.NameSpace)
	stsClient := k.Cli.AppsV1().StatefulSets(k.NameSpace)

	// Get Deployments
	deployment, err := deployClient.Get(k.Ctx, k.AppName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		sts, err := stsClient.Get(k.Ctx, k.AppName, v1.GetOptions{})
		CheckIfError(err)
		k.updateStatefulset(sts, stsClient)
		go k.checkPod()
		k.checkStatefulset(stsClient)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting deployment%v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		CheckIfError(err)
	} else {
		k.updateDeployment(deployment, deployClient)
		go k.checkPod()
		k.checkDeployment(deployClient)
	}
}

// UpdateDeployment .
func (k KubeClient) updateDeployment(deployment *app_v1.Deployment, client cli.DeploymentInterface) {
	containers := &deployment.Spec.Template.Spec.Containers
	found := false
	Info("Change kubernetes application images.")
	for i := range *containers {
		c := *containers
		if c[i].Name == k.AppName {
			found = true
			fmt.Println("Pod: ", c[i].Name)
			fmt.Println("Old image ->", c[i].Image)
			fmt.Println("New image ->", k.ImageName)
			if c[i].Image == k.ImageName {
				Info(buildString("Code version not update,Restart pod: ", c[i].Name))
				k.deletePod(c[i].Name)
			}
			c[i].Image = k.ImageName
		}
	}
	if found == false {
		fmt.Println("The application container not exist in the deployment pods.")
		os.Exit(1)
	}
	_, err := client.Update(k.Ctx, deployment, v1.UpdateOptions{})
	CheckIfError(err)
}

// updateStatefulset .
func (k KubeClient) updateStatefulset(statefulset *app_v1.StatefulSet, client cli.StatefulSetInterface) {
	containers := &statefulset.Spec.Template.Spec.Containers
	found := false
	Info("Change kubernetes application images.")
	for i := range *containers {
		c := *containers
		if c[i].Name == k.AppName {
			found = true
			fmt.Println("Old image ->", c[i].Image)
			fmt.Println("New image ->", k.ImageName)
			if c[i].Image == k.ImageName {
				Info(buildString("Code version not update,Restart pod: ", c[i].Name))
			}
			c[i].Image = k.ImageName
		}
	}
	if found == false {
		fmt.Println("The application container not exist in the statefulset pods.")
		os.Exit(1)
	}
	_, err := client.Update(k.Ctx, statefulset, v1.UpdateOptions{})
	CheckIfError(err)
}

// checkDeployment .
func (k KubeClient) checkDeployment(client cli.DeploymentInterface) {
	Info("Checking deployment application status.")
	// Check pod
	checkCount := 0
	checkTimeOut := 60 * 10
	startTime := time.Now().Unix()
	// 等待更新完成
	for {
		// 获取k8s中deployment的状态
		k8sDeployment, err := client.Get(k.Ctx, k.AppName, v1.GetOptions{})
		if err != nil {
			time.Sleep(5 * time.Second)
		}
		status := k8sDeployment.Status
		replicas := k8sDeployment.Spec.Replicas

		// 进行状态判定
		if status.UpdatedReplicas == *replicas &&
			status.Replicas == *replicas &&
			status.AvailableReplicas == *replicas &&
			status.ObservedGeneration == k8sDeployment.Generation {
			// 滚动升级完成
			time.Sleep(time.Duration(1) * time.Second)
			checkCount++
			if checkCount >= 5 {
				Info("Rolling update application success.")
				break
			}
		}
		stopTime := time.Now().Unix()
		timeDiff := int(stopTime - startTime)
		if timeDiff >= checkTimeOut {
			Info(fmt.Sprintf("timeout :%d", checkTimeOut))
			os.Exit(1)
		}
		goto OUTPUT

	OUTPUT:
		time.Sleep(time.Duration(5) * time.Second)
		fmt.Printf("UpdatedReplicas:(%d/%d) Replicas:(%d,%d) AvailableReplicas:(%d/%d) ObservedGeneration:(%d/%d) TIMEOUT: %ds\n",
			status.UpdatedReplicas, *replicas,
			status.Replicas, *replicas,
			status.AvailableReplicas, *replicas,
			status.ObservedGeneration, k8sDeployment.Generation,
			checkTimeOut-timeDiff)
	}
}

// checkStatefulset .
func (k KubeClient) checkStatefulset(client cli.StatefulSetInterface) {
	Info("Checking statefulset application status.")
	// Check pod
	checkCount := 0
	checkTimeOut := 60 * 10
	startTime := time.Now().Unix()
	// 等待更新完成
	for {
		// 获取k8s中deployment的状态
		k8sStatefulset, err := client.Get(k.Ctx, k.AppName, v1.GetOptions{})
		if err != nil {
			time.Sleep(5 * time.Second)
		}
		status := k8sStatefulset.Status
		replicas := k8sStatefulset.Spec.Replicas

		// 进行状态判定
		if status.UpdatedReplicas == *replicas &&
			status.Replicas == *replicas &&
			status.CurrentReplicas == *replicas &&
			status.ObservedGeneration == k8sStatefulset.Generation {
			// 滚动升级完成
			time.Sleep(time.Duration(5) * time.Second)
			checkCount++
			if checkCount >= 5 {
				Info("Rolling update application success.")
				break
			}
		}
		stopTime := time.Now().Unix()
		timeDiff := int(stopTime - startTime)
		if timeDiff >= checkTimeOut {
			Warning(fmt.Sprintf("timeout: %ds, Pls check pod status or connect 'WEIMEILONG'.", checkTimeOut))
			os.Exit(1)
		}
		goto OUTPUT

	OUTPUT:
		time.Sleep(time.Duration(5) * time.Second)
		fmt.Printf("UpdatedReplicas:(%d/%d) Replicas:(%d,%d) CurrentReplicas:(%d/%d) ObservedGeneration:(%d/%d) TIMEOUT: %ds\n",
			status.UpdatedReplicas, *replicas,
			status.Replicas, *replicas,
			status.CurrentReplicas, *replicas,
			status.ObservedGeneration, k8sStatefulset.Generation,
			checkTimeOut-timeDiff)
	}
}

// CheckPod .
func (k KubeClient) checkPod() {
	for i := 0; i < 3; i++ {
		time.Sleep(time.Duration(5) * time.Second)
		// 打印每个pod的状态(可能会打印出terminating中的pod, 但最终只会展示新pod列表)
		if podList, err := k.Cli.CoreV1().Pods(k.NameSpace).List(k.Ctx, v1.ListOptions{LabelSelector: "k8s-app=" + k.AppName}); err == nil {
			for _, pod := range podList.Items {
				podName := pod.Name
				podStatus := string(pod.Status.Phase)
				// PodRunning means the pod has been bound to a node and all of the containers have been started.
				// At least one container is still running or is in the process of being restarted.
				if podStatus == string(core_v1.PodRunning) {
					// 汇总错误原因不为空
					if pod.Status.Reason != "" {
						podStatus = pod.Status.Reason
						goto KO
					}
					// condition有错误信息
					for _, cond := range pod.Status.Conditions {
						if cond.Type == core_v1.PodReady { // POD就绪状态
							if cond.Status != core_v1.ConditionTrue { // 失败
								podStatus = cond.Reason
							}
							goto KO
						}
					}
					// 没有ready condition, 状态未知
					podStatus = "Unknown"
				}
			KO:
				if podStatus != string(core_v1.PodRunning) {
					k.deletePod(podName)
				}
			}
		}
	}
}
func (k KubeClient) deletePod(podName string) {
	err := k.Cli.CoreV1().Pods(k.NameSpace).Delete(k.Ctx, podName, v1.DeleteOptions{})
	if err != nil {
		CheckIfError(err)
	}
}
