package modules

import (
	"fmt"

	internalapi "k8s.io/cri-api/pkg/apis"

	mod "elastic/modules"
	global "elastic/modules/global"
)

func DecisionScaleUp(client internalapi.RuntimeService, podIndex map[string]int64, podInfoSet []global.PodData, currentRunningPods []string,
	systemInfoSet []global.SystemInfo, priorityMap map[string]global.PriorityContainer, scaleUpCandidateList []global.ScaleCandidateContainer,
	pauseContainerList []global.PauseContainer) ([]global.PodData, []global.ScaleCandidateContainer, []global.PauseContainer) {

	// Scale Candidate List
	var scaleUpMemorySize int64
	var sumLimitMemorySize int64

	memory := systemInfoSet[len(systemInfoSet)-1].Memory

	scaleUpCandidateList, sumLimitMemorySize = AppendToScaleUpCandidateList(scaleUpCandidateList, podIndex, podInfoSet, currentRunningPods)

	// calculate require memory
	for i, scaleCandidate := range scaleUpCandidateList {
		scaleUpCandidateList[i].ScaleSize = CalculateScaleSize(scaleCandidate.ContainerData)
		scaleUpMemorySize += scaleUpCandidateList[i].ScaleSize
	}

	// Memory capacity is sufficient
	// 기존에 os에서 점유하는 메모리가 있기 때문에 그걸 offset으로 빼주어야 할 것이다.
	if float64(sumLimitMemorySize)+float64(scaleUpMemorySize) < float64(memory.Total)*global.MAX_MEMORY_USAGE_THRESHOLD {
		// Scale up all containers
		fmt.Println("scaleUpCandidateList: ", len(scaleUpCandidateList))
		for _, scaleCandidate := range scaleUpCandidateList {
			ScaleUp(client, scaleCandidate.ContainerData, scaleCandidate.ScaleSize)
			// update container info
			mod.UpdateContainerData(client, scaleCandidate.ContainerData)
			// increase the number of scale
			scaleCandidate.ContainerData.NumOfScale++
			// reset TimeWindow
			scaleCandidate.ContainerData.TimeWindow = 0
			// reset container resource slice
			scaleCandidate.ContainerData.Resource = scaleCandidate.ContainerData.Resource[:0]
		}
		// Reset scaleUpCandidateList
		scaleUpCandidateList = scaleUpCandidateList[:0]
	} else { // Memory capacity is not sufficient
		var listSize = len(scaleUpCandidateList)
		sortedScaleUpCandidateList := make([]global.ScaleCandidateContainer, 0, len(scaleUpCandidateList))
		var targetScaleUpList []global.ScaleCandidateContainer
		scaleUpMemorySize = 0

		// Sort scaleUpCandidateList with highest priority
		for i := 0; i < listSize; i++ {
			var highestPriority float64 = 0
			var highestPriorityIndex int
			for j := 0; j < listSize-i; j++ {
				if highestPriority < priorityMap[scaleUpCandidateList[j].ContainerName].Priority {
					highestPriorityIndex = j
				}
			}
			sortedScaleUpCandidateList = append(sortedScaleUpCandidateList, scaleUpCandidateList[highestPriorityIndex])
			scaleUpCandidateList = append(scaleUpCandidateList[:highestPriorityIndex], scaleUpCandidateList[highestPriorityIndex+1:]...)
		}

		// select container to scale up
		var scaleUpIndex int
		var lastScaleUpSize int64
		for scaleUpIndex = 0; scaleUpIndex < len(sortedScaleUpCandidateList); scaleUpIndex++ {
			scaleUpMemorySize += sortedScaleUpCandidateList[scaleUpIndex].ScaleSize
			if float64(memory.Used)+float64(scaleUpMemorySize) > float64(memory.Total)*global.MAX_MEMORY_USAGE_THRESHOLD {
				lastScaleUpSize = int64(float64(memory.Total)*global.MAX_MEMORY_USAGE_THRESHOLD -
					(float64(memory.Used) + float64(scaleUpMemorySize) - float64(sortedScaleUpCandidateList[scaleUpIndex].ScaleSize)))
				scaleUpIndex++
				break
			}
		}
		// append index 0 to scaleUpindex
		targetScaleUpList = sortedScaleUpCandidateList[:scaleUpIndex]

		// Scale up selected container
		for i, scaleCandidate := range targetScaleUpList {
			if lastScaleUpSize != 0 && i == len(targetScaleUpList)-1 {
				scaleCandidate.ScaleSize = lastScaleUpSize
				ScaleUp(client, scaleCandidate.ContainerData, scaleCandidate.ScaleSize)
				// update container info
				mod.UpdateContainerData(client, scaleCandidate.ContainerData)
			} else {
				ScaleUp(client, scaleCandidate.ContainerData, scaleCandidate.ScaleSize)
				// update container info
				mod.UpdateContainerData(client, scaleCandidate.ContainerData)
			}
			// increase the number of scale
			scaleCandidate.ContainerData.NumOfScale++
			// reset TimeWindow
			scaleCandidate.ContainerData.TimeWindow = 0
			// reset container resource slice
			scaleCandidate.ContainerData.Resource = scaleCandidate.ContainerData.Resource[:0]
		}
		// Change scaleUpCandidateList (append index i to end)
		if scaleUpIndex < len(sortedScaleUpCandidateList)-1 { // If all containers are scaled up
			scaleUpCandidateList = sortedScaleUpCandidateList[scaleUpIndex+1:]
		} else { // If all containers are not scaled up
			scaleUpCandidateList = scaleUpCandidateList[:0]
		}

		// logic to pause low priority container
		for _, pauseCandidate := range scaleUpCandidateList {
			// Check for pauses
			if CheckToPauseContainer(*pauseCandidate.ContainerData, pauseContainerList) { // if not pause yet
				PauseContainer(client, pauseCandidate.ContainerData)
				// update container info
				mod.UpdateContainerData(client, pauseCandidate.ContainerData)

				// append pauseContainerList
				AppendPauseContainerList(pauseContainerList, pauseCandidate)
			}
		}
		// logic to continue execute scaleup container
		fmt.Println("pauseContainerList: ", len(pauseContainerList))
		for i, pauseContainer := range pauseContainerList {
			isRequireContinue := true
			for _, scaleUpCandidate := range scaleUpCandidateList {
				if pauseContainer.ContainerName == scaleUpCandidate.ContainerName {
					isRequireContinue = false
					break
				}
			}
			if isRequireContinue {
				ContinueContainer(client, pauseContainer.ContainerData)
				// update container info
				mod.UpdateContainerData(client, pauseContainer.ContainerData)

				// delete pauseContainerList
				pauseContainerList = append(pauseContainerList[:i], pauseContainerList[i+1:]...)
			}
		}
	}
	return podInfoSet, scaleUpCandidateList, pauseContainerList
}

func DecisionScaleDown(client internalapi.RuntimeService, podIndex map[string]int64, podInfoSet []global.PodData, currentRunningPods []string, systemInfoSet []global.SystemInfo) []global.PodData {
	// Scale Candidate List
	var scaleDownCandidateList []global.ScaleCandidateContainer

	podInfoSet, scaleDownCandidateList = AppendToScaleDownCandidateList(client, scaleDownCandidateList, podIndex, podInfoSet, currentRunningPods)

	for _, scaleCandidate := range scaleDownCandidateList {
		res := scaleCandidate.ContainerData.Resource
		scaleCandidate.ScaleSize = int64(float64(res[len(res)-1].MemoryUsageBytes) / float64(global.CONTAINER_MEMORY_SLO))
		fmt.Println("Scale Size: ", scaleCandidate.ScaleSize)
		ScaleDown(client, scaleCandidate.ContainerData, scaleCandidate.ScaleSize)
		// reset TimeWindow
		scaleCandidate.ContainerData.TimeWindow = 0
		// update container info
		mod.UpdateContainerData(client, scaleCandidate.ContainerData)

		// reset container resource slice
		scaleCandidate.ContainerData.Resource = scaleCandidate.ContainerData.Resource[:0]
	}

	return podInfoSet
}

func AppendToScaleUpCandidateList(scaleUpCandidateList []global.ScaleCandidateContainer, podIndex map[string]int64, podInfoSet []global.PodData, currentRunningPods []string) ([]global.ScaleCandidateContainer, int64) {
	var sumLimitMemorySize int64

	for _, podName := range currentRunningPods {
		pod := podInfoSet[podIndex[podName]]
		for i, container := range pod.Container {
			res := container.Resource

			// exception handling
			if len(res) == 0 {
				continue
			}

			conMemUtil := res[len(res)-1].ConMemUtil
			sumLimitMemorySize += container.Cgroup.MemoryLimitInBytes

			// Register scale candidates
			if conMemUtil > global.CONTAINER_MEMORY_SLO_UPPER {
				// No need to add to the array if it has already added
				var scaleUpCandiate global.ScaleCandidateContainer

				if CheckToAppendScaleCandidateList(container, scaleUpCandidateList) {
					scaleUpCandiate.PodName = podName
					scaleUpCandiate.PodId = pod.Id
					scaleUpCandiate.ContainerName = container.Name
					scaleUpCandiate.ContainerId = container.Id
					scaleUpCandiate.ContainerData = &pod.Container[i]

					scaleUpCandidateList = append(scaleUpCandidateList, scaleUpCandiate)
				}
			}
		}
	}
	fmt.Println(len(scaleUpCandidateList))
	return scaleUpCandidateList, sumLimitMemorySize
}

func AppendToScaleDownCandidateList(client internalapi.RuntimeService, scaleDownCandidateList []global.ScaleCandidateContainer, podIndex map[string]int64, podInfoSet []global.PodData, currentRunningPods []string) ([]global.PodData, []global.ScaleCandidateContainer) {

	for _, podName := range currentRunningPods {
		pod := podInfoSet[podIndex[podName]]
		for i, container := range pod.Container {
			res := container.Resource
			conMemUtil := res[len(res)-1].ConMemUtil

			// Minimum size per container
			if len(pod.Container[i].Resource) < global.SCALE_DOWN_THRESHOLD {
				continue
			}
			// exception handling: MemoryUsageBytes < global.MIN_SIZE_PER_CONTAINER
			if int64(float64(res[len(res)-1].MemoryUsageBytes)*float64(global.CONTAINER_MEMORY_SLO_LOWER)) < global.MIN_SIZE_PER_CONTAINER {
				if container.Cgroup.MemoryLimitInBytes > global.MIN_SIZE_PER_CONTAINER {
					fmt.Println("호출됨")
					ScaleDown(client, &container, global.MIN_SIZE_PER_CONTAINER)
					// update container info
					mod.UpdateContainerData(client, &container)
					// reset TimeWindow
					pod.Container[i].TimeWindow = 0
					// reset container resource slice
					pod.Container[i].Resource = container.Resource[:0]
				}
				// if MemoryLimitInBytes < global.MIN_SIZE_PER_CONTAINER
				continue
			}

			// Register scale candidates
			if conMemUtil < global.CONTAINER_MEMORY_SLO_LOWER {
				// No need to add to the array if it has already added
				var scaleDownCandiate global.ScaleCandidateContainer
				scaleDownCandiate.PodName = podName
				scaleDownCandiate.PodId = pod.Id
				scaleDownCandiate.ContainerName = container.Name
				scaleDownCandiate.ContainerId = container.Id
				scaleDownCandiate.ContainerData = &pod.Container[i]

				scaleDownCandidateList = append(scaleDownCandidateList, scaleDownCandiate)
			}
		}
	}
	return podInfoSet, scaleDownCandidateList
}

func CalculateScaleSize(scaleCandidate *global.ContainerData) int64 {
	// Determining the size of a formula-based resource allocation
	// Scaleup_c=∑_(t=1)^(T/10)▒〖T/20t*(〖C_MemUsed〗^10(t-1) -〖C_MemUsed〗^10t)
	var scaleSize float64

	reslen := len(scaleCandidate.Resource)
	for i := 0; i <= (reslen-1)/10; i++ {
		if i == reslen/10 {
			scaleSize += float64(scaleCandidate.Resource[reslen-(i*10+1)].MemoryUsageBytes-scaleCandidate.Resource[0].MemoryUsageBytes) *
				(float64(global.MAX_TIME_WINDOW) / float64(20*(i+1))) // 수식 조정이 필요할 수도 잇음
		} else {
			scaleSize += float64(scaleCandidate.Resource[reslen-(i*10+1)].MemoryUsageBytes-scaleCandidate.Resource[reslen-((i+1)*10)].MemoryUsageBytes) *
				(float64(global.MAX_TIME_WINDOW) / float64(20*(i+1)))
		}
	}

	return int64(scaleSize)
}

func CheckToAppendScaleCandidateList(container global.ContainerData, scaleCandidateList []global.ScaleCandidateContainer) bool {
	for _, scaleCandidate := range scaleCandidateList {
		if scaleCandidate.ContainerName == container.Name {
			return false
		}
	}
	return true
}

func ScaleUp(client internalapi.RuntimeService, scaleUpCandidate *global.ContainerData, scaleUpSize int64) {
	scaleUpCandidate.OriginalContainerData.Linux.MemoryLimitInBytes += scaleUpSize
	mod.UpdateContainerResources(client, scaleUpCandidate.Id, scaleUpCandidate.OriginalContainerData)
}

func ScaleDown(client internalapi.RuntimeService, scaleDownCandidate *global.ContainerData, scaleDownSize int64) {
	scaleDownCandidate.OriginalContainerData.Linux.MemoryLimitInBytes = scaleDownSize
	mod.UpdateContainerResources(client, scaleDownCandidate.Id, scaleDownCandidate.OriginalContainerData)
}
