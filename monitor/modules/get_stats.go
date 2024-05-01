package modules

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	internalapi "k8s.io/cri-api/pkg/apis"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1"

	global "elastic/modules/global"
)

func GetListPodStatsInfo(client internalapi.RuntimeService) []*pb.PodSandboxStats {
	for {
		stats, err := client.ListPodSandboxStats(context.TODO(), &pb.PodSandboxStatsFilter{})
		if err != nil {
			fmt.Println(err)
		} else {
			return stats
		}
	}
}

func GetPodStatsInfo(client internalapi.RuntimeService, podIndex map[string]int64, podInfoSet []global.PodData, currentRunningPods []string) ([]global.PodData, []string) {

	listPodStats := GetListPodStatsInfo(client)
	isPodRunning := false

	for _, podStats := range listPodStats {
		podName := podStats.Attributes.Metadata.Name

		// Do not store namespaces other than default namespaces
		if podStats.Attributes.Metadata.Namespace != "default" {
			continue
		}
		isPodRunning = true

		// Do not store info of notworking pods
		status, _ := client.PodSandboxStatus(context.TODO(), podStats.Attributes.Id, false)
		if status.Status.State == 1 { // exception handling: SANDBOX_NOTREADY
			continue
		}

		// check current running pods
		currentRunningPods = append(currentRunningPods, podName)

		// init pod data
		if _, exists := podIndex[podName]; !exists {
			podInfoSet = InitPodData(podName, podIndex, podInfoSet, podStats)
		}
		pod := &podInfoSet[podIndex[podName]]

		// get containers stats
		// getContainerStatsInfo 함수로 독립하는 것이 더 보기 깔끔할듯
		for _, containerStats := range podStats.Linux.Containers {
			containerName := containerStats.Attributes.Metadata.Name

			var containerResource global.ContainerResourceData

			// init container data
			if _, exists := pod.ContainerIndex[containerName]; !exists {
				pod.Container = InitContainerData(client, pod, containerName, containerStats)
				//exception handling
				if pod.Container == nil {
					continue
				}
			}
			container := &pod.Container[pod.ContainerIndex[containerName]]

			if containerStats.Cpu == nil || containerStats.Memory == nil { // exception handling: nil pointer
				continue
			}
			containerResource.CpuUsageCoreNanoSeconds = containerStats.Cpu.UsageCoreNanoSeconds.Value
			// do not support CpuUsageNanoCores in cri-o runtime
			//podInfo.Container[i].Resource.CpuUsageNanoCores = containerStats.Cpu.UsageNanoCores.Value
			containerResource.MemoryAvailableBytes = containerStats.Memory.AvailableBytes.Value
			containerResource.MemoryWorkingSetBytes = containerStats.Memory.WorkingSetBytes.Value
			containerResource.MemoryUsageBytes = containerStats.Memory.WorkingSetBytes.Value

			// convert nanocores to millicores
			containerResource.CpuUsageCoreMilliSeconds = containerResource.CpuUsageCoreNanoSeconds / uint64(global.NANOCORES_TO_MILLICORES)

			container.Resource = append(container.Resource, containerResource)

			// set time window size
			// Data 변경 시 지역 변수에 접근하면 값이 변경되지 않으니 주의한다.
			if container.TimeWindow < global.MAX_TIME_WINDOW {
				container.TimeWindow++
			} else {
				container.Resource = container.Resource[1:]
			}
		}
	}

	if !isPodRunning {
		fmt.Println("There is no pod running.")
	}

	return podInfoSet, currentRunningPods
}

func InitPodData(podName string, podIndex map[string]int64, podInfoSet []global.PodData, podStats *pb.PodSandboxStats) []global.PodData {
	var podInfo global.PodData

	podInfo.Id = podStats.Attributes.Id
	podInfo.Name = podStats.Attributes.Metadata.Name
	podInfo.Uid = podStats.Attributes.Metadata.Uid
	podInfo.Namespace = podStats.Attributes.Metadata.Namespace

	podIndex[podName] = int64(len(podInfoSet))
	podInfoSet = append(podInfoSet, podInfo) // append dynamic array

	return podInfoSet
}

func InitContainerData(client internalapi.RuntimeService, pod *global.PodData, containerName string, containerStats *pb.ContainerStats) []global.ContainerData {
	var container global.ContainerData

	container.Id = containerStats.Attributes.Id
	container.Name = containerStats.Attributes.Metadata.Name
	container.Attempt = containerStats.Attributes.Metadata.Attempt

	containerStatus, err := client.ContainerStatus(context.TODO(), container.Id, false)
	if err != nil { // exception handling
		fmt.Println(err)
		fmt.Println("Remove Pod Set")
		return nil
	}

	container.CreatedAt = containerStatus.Status.CreatedAt
	container.StartedAt = containerStatus.Status.StartedAt
	container.FinishedAt = containerStatus.Status.FinishedAt

	containerRes := containerStatus.Status.Resources

	// get containerCgroupResources
	container.Cgroup.CpuPeriod = containerRes.Linux.CpuPeriod
	container.Cgroup.CpuQuota = containerRes.Linux.CpuQuota
	container.Cgroup.CpuShares = containerRes.Linux.CpuShares
	container.Cgroup.MemoryLimitInBytes = containerRes.Linux.MemoryLimitInBytes
	//container.Cgroup.OomScoreAdj = containerRes.Linux.OomScoreAdj
	//container.Cgroup.CpusetCpus = containerRes.Linux.CpusetCpus
	//container.Cgroup.CpusetMems = containerRes.Linux.CpusetMems

	//append originalContainerData
	container.OriginalContainerData = containerRes

	//append container in podInfoSet
	pod.ContainerIndex = make(map[string]int64)
	pod.ContainerIndex[containerName] = int64(len(pod.Container))

	pod.Container = append(pod.Container, container)

	return pod.Container
}

func UpdateContainerData(client internalapi.RuntimeService, containerData *global.ContainerData) {

	containerStatus, _ := client.ContainerStatus(context.TODO(), containerData.Id, false)
	/* 문제 발생 시 다시 exception handling 고민
	if err != nil { // exception handling
		fmt.Println(err)
		fmt.Println("Remove Pod Set")
		podInfoSet = RemovePodofPodInfoSet(podInfoSet, i)
		break
	}*/

	// 이 시간도 굳이 갱신을 해야 하나??
	containerData.CreatedAt = containerStatus.Status.CreatedAt
	containerData.StartedAt = containerStatus.Status.StartedAt
	containerData.FinishedAt = containerStatus.Status.FinishedAt

	containerRes := containerStatus.Status.Resources

	// get containerCgroupResources
	containerData.Cgroup.CpuPeriod = containerRes.Linux.CpuPeriod
	containerData.Cgroup.CpuQuota = containerRes.Linux.CpuQuota
	containerData.Cgroup.CpuShares = containerRes.Linux.CpuShares
	containerData.Cgroup.MemoryLimitInBytes = containerRes.Linux.MemoryLimitInBytes
	fmt.Println(containerRes.Linux.MemoryLimitInBytes)
	//container.Cgroup.OomScoreAdj = containerRes.Linux.OomScoreAdj
	//container.Cgroup.CpusetCpus = containerRes.Linux.CpusetCpus
	//container.Cgroup.CpusetMems = containerRes.Linux.CpusetMems

	//append originalContainerData
	containerData.OriginalContainerData = containerRes
}

/*
func GetContainerCgroupStatsInfo(client internalapi.RuntimeService, container *global.ContainerData) {
	cpuinfo := map[string]interface{}{}
	meminfo := map[string]interface{}{}

	command := "crictl inspect " + container.Id + " | jq '.info.runtimeSpec.linux.resources|"
	cpucommand := command + ".cpu'"
	memcommand := command + ".memory'"
	cpuout, _ := exec.Command("bash", "-c", cpucommand).Output()
	memout, _ := exec.Command("bash", "-c", memcommand).Output()

	json.Unmarshal(cpuout, &cpuinfo)
	json.Unmarshal(memout, &meminfo)

	fmt.Println(cpuinfo)

	// crictl inspect id
	container.Cgroup.CpuPeriod = int64(cpuinfo["period"].(float64))
	container.Cgroup.CpuQuota = int64(cpuinfo["quota"].(float64))
	container.Cgroup.CpuShares = int64(cpuinfo["shares"].(float64))
	container.Cgroup.MemoryLimitInBytes = int64(meminfo["limit"].(float64))
	// container.Cgroup.OomScoreAdj = status.Resources.Linux.OomScoreAdj
	// container.Cgroup.CpusetCpus = status.Resources.Linux.CpusetCpus
	// container.Cgroup.CpusetMems = status.Resources.Linux.CpusetMems
}*/

func GetSystemStatsInfo(systemInfoSet []global.SystemInfo) []global.SystemInfo {
	var getCpu global.Cpu
	var getMemory global.Memory
	var getSystemInfo global.SystemInfo

	per_cpu, err := cpu.Times(true)
	if err != nil {
		panic(err)
	}
	// fmt.Println(per_cpu)

	for i := 0; i < len(per_cpu); i++ {
		getCpu.User += per_cpu[i].User
		getCpu.System += per_cpu[i].System
		getCpu.Nice += per_cpu[i].Nice
		getCpu.Irq += per_cpu[i].Irq
		getCpu.Softirq += per_cpu[i].Softirq
		getCpu.Steal += per_cpu[i].Steal

		getCpu.Idle += per_cpu[i].Idle
		getCpu.Iowait += per_cpu[i].Iowait
	}
	getCpu.TotalCore = getCpu.User + getCpu.System + getCpu.Nice + getCpu.Irq + getCpu.Softirq + getCpu.Steal + getCpu.Idle + getCpu.Iowait
	getCpu.TotalMilliCore = getCpu.TotalCore * global.CORES_TO_MILLICORES // core to millicore

	memory, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}
	// fmt.Println(memory)

	getMemory.Total = memory.Total
	getMemory.Available = memory.Available
	getMemory.Used = memory.Total - memory.Available
	getMemory.UsedPercent = float64(getMemory.Used) / float64(memory.Total) * 100

	getSystemInfo.Cpu = getCpu
	getSystemInfo.Memory = getMemory

	systemInfoSet = append(systemInfoSet, getSystemInfo)

	return systemInfoSet
}
