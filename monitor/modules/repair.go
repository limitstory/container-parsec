package modules

import (
	"context"
	"fmt"

	cp "elastic/modules/checkpoint"
	global "elastic/modules/global"

	internalapi "k8s.io/cri-api/pkg/apis"
)

func DecisionRepairContainer(
	systemInfoSet []global.SystemInfo, pauseContainerList []global.PauseContainer,
	checkpointContainerList []global.CheckpointMetaData, lenghOfCurrentRunningPods int,
	priorityMap map[string]global.PriorityContainer, removeContainerList []global.CheckpointMetaData) (
	[]global.PauseContainer, []global.CheckpointMetaData, []global.CheckpointMetaData) {

	var mem int64
	var repairContainerCandidateList []global.CheckpointMetaData

	// 간단하게 복구로직을 짰는데, 이 부분은 고민이 필요함.
	// 먼저 들어오고 먼저 나가는 방식이 아니라 메모리 조건 만족하면 바로 나갈 수 있게끔??
	// 연산 cost가 너무 커짐... 메모리 순으로 다시 sort해야 한다.
	for {
		mem += removeContainerList[len(removeContainerList)-1].MemoryLimitInBytes

		if mem > int64(systemInfoSet[len(systemInfoSet)-1].Memory.Total) { // 여기에 컨테이너 limit의 합이 들어가야 한다.
			// mem + containerlimitsizesum > total
			break
		}

		removeContainer := removeContainerList[len(removeContainerList)-1]
		repairContainerCandidateList = append(repairContainerCandidateList, removeContainer)
		removeContainerList = append(removeContainerList[:len(removeContainerList)-1])
	}

	for _, repairContainerCandidate := range repairContainerCandidateList {
		cp.MakeContainerFromCheckpoint(repairContainerCandidate)
		cp.RestoreContainer(repairContainerCandidate)
		// restore한 컨테이너를 어떻게 처리할 것인가????
		// 컨테이너 이름 같으면 저쪽에서 새로운 컨테이너로 인식을 못할 것이며,
		// 갱신된 내용은 이쪽에서 updateContainerResources를 통해서 갱신해주어야 할 것
		// restartcount++
	}

	return pauseContainerList, checkpointContainerList, removeContainerList
}

func RepairContainer(client internalapi.RuntimeService, selectContainerId string) {
	err := client.RemoveContainer(context.TODO(), selectContainerId)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println()
	fmt.Println("Remove Container Id:", selectContainerId)
}
