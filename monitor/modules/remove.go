package modules

import (
	"context"
	"fmt"
	"time"

	global "elastic/modules/global"

	internalapi "k8s.io/cri-api/pkg/apis"
)

func DecisionRemoveContainer(
	client internalapi.RuntimeService, pauseContainerList []global.PauseContainer,
	checkpointContainerList []global.CheckpointMetaData, lenghOfCurrentRunningPods int,
	priorityMap map[string]global.PriorityContainer, removeContainerList []global.CheckpointMetaData) (
	[]global.PauseContainer, []global.CheckpointMetaData, []global.CheckpointMetaData) {

	var offset = lenghOfCurrentRunningPods / 2
	var removeCandidateContainerList []global.PauseContainer

	// Decide which container to remove
	for len(pauseContainerList) > offset {
		var lowestPriority float64 = 9999999999
		var lowestPriorityIndex int

		listSize := len(pauseContainerList)

		// Select from pauseContainerList with lowest priority
		for j := 0; j < listSize; j++ {
			if lowestPriority > priorityMap[pauseContainerList[j].ContainerName].Priority {
				lowestPriorityIndex = j
			}
		}
		removeCandidateContainerList = append(removeCandidateContainerList, pauseContainerList[lowestPriorityIndex])
		pauseContainerList = append(pauseContainerList[:lowestPriorityIndex], pauseContainerList[lowestPriorityIndex+1:]...)
	}

	// When the memory usage of the container is very high
	for i, pauseContainer := range pauseContainerList {
		res := pauseContainer.ContainerData.Resource
		if res[len(res)-1].ConMemUtil > global.CONTAINER_MEMORY_USAGE_THRESHOLD {
			removeCandidateContainerList = append(removeCandidateContainerList, pauseContainer)
			pauseContainerList = append(pauseContainerList[:i], pauseContainerList[i+1:]...)
		}
	}

	// Kill low-priority containers
	for _, removeCandidateContainer := range removeCandidateContainerList {
		RemoveContainer(client, removeCandidateContainer.ContainerId)
		// move from checkpointContainerList to removeContainerList
		for i, checkpoint := range checkpointContainerList {
			if checkpoint.ContainerName == removeCandidateContainer.ContainerName {
				checkpoint.RemoveStartTime = time.Now().Unix()
				removeContainerList = append(removeContainerList, checkpoint)
				checkpointContainerList = append(checkpointContainerList[:i], checkpointContainerList[i+1:]...)
			}
		}
		// removeCount++
	}

	return pauseContainerList, checkpointContainerList, removeContainerList
}

func RemoveContainer(client internalapi.RuntimeService, selectContainerId string) {
	err := client.RemoveContainer(context.TODO(), selectContainerId)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println()
	fmt.Println("Remove Container Id:", selectContainerId)
}
