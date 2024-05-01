package main

import (
	"fmt"
	"time"

	mod "elastic/modules"
	global "elastic/modules/global"

	remote "k8s.io/kubernetes/pkg/kubelet/cri/remote"
)

type CheckContainer struct {
	PodName        string
	ContainerName  string
	ContainerId    string
	MaxMemoryUsage uint64
	SumMemoryUsage uint64
	MinMemoryUsage uint64
	iteration      uint64
}

func main() {
	const ENDPOINT string = "unix:///var/run/crio/crio.sock"

	podIndex := make(map[string]int64)
	var podInfoSet []global.PodData
	var systemInfoSet []global.SystemInfo

	checkMemory := make(map[string]CheckContainer)

	/*
		// kubernetes api 클라이언트 생성하는 모듈
		clientset := mod.InitClient()
		if clientset != nil {
			fmt.Println("123")
		}*/

	//get new internal client service
	client, err := remote.NewRemoteRuntimeService(ENDPOINT, time.Second*2, nil)
	if err != nil {
		panic(err)
	}
	// remote.NewRemoteImageService("unix:///var/run/containerd/containerd.sock", time.Second*2, nil)

	// execute monitoring & resource management logic
	for {
		var currentRunningPods []string
		// definition of data structure to store
		//var selectContainerId = make([]string, 0)
		//var selectContainerResource = make([]*pb.ContainerResources, 0)

		// get system metrics
		// 자원 변경 시 TimeWindow Reset 수행하면서 다른 리소스 자원 기록도 전부 날리는 것으로....

		podInfoSet, currentRunningPods = mod.MonitoringPodResources(client, podIndex, podInfoSet, currentRunningPods, systemInfoSet)

		if len(currentRunningPods) == 0 {
			break
		}

		for i := 0; i < len(currentRunningPods); i++ {
			if len(podInfoSet[i].Container) == 0 {
				continue
			}
			//res := podInfoSet[i].Container[0].Resource
			//fmt.Println(res[len(res)-1].ConMemUtil)
			//fmt.Println(res[len(res)-1].CpuUtil)
			//fmt.Println()
		}

		for _, podName := range currentRunningPods {
			var info CheckContainer

			if len(podInfoSet[podIndex[podName]].Container) == 0 {
				continue
			}
			if _, exists := checkMemory[podName]; !exists {
				info.PodName = podName
				info.ContainerName = podInfoSet[podIndex[podName]].Container[0].Name
				info.ContainerId = podInfoSet[podIndex[podName]].Container[0].Id
				info.MaxMemoryUsage = 0
				info.MinMemoryUsage = 999999999999999999
				info.SumMemoryUsage = 0
				info.iteration = 0
			} else {
				info = checkMemory[podName]
			}
			pod := podInfoSet[podIndex[podName]]
			res := pod.Container[0].Resource
			if info.MinMemoryUsage > res[len(res)-1].MemoryUsageBytes {
				info.MinMemoryUsage = res[len(res)-1].MemoryUsageBytes
			}
			if info.MaxMemoryUsage < res[len(res)-1].MemoryUsageBytes {
				info.MaxMemoryUsage = res[len(res)-1].MemoryUsageBytes
			}
			info.SumMemoryUsage += res[len(res)-1].MemoryUsageBytes
			info.iteration++
			checkMemory[podName] = info
		}
		time.Sleep(time.Second)
	}

	for _, info := range checkMemory {
		fmt.Println(info.PodName)
		fmt.Println("Min:", float64(info.MinMemoryUsage)/1000000)
		fmt.Println("Avg:", float64(info.SumMemoryUsage/info.iteration)/1000000)
		fmt.Println("Max:", float64(info.MaxMemoryUsage)/1000000)
		fmt.Println("===============================================")
	}
}
