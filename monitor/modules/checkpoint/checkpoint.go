package modules

import (
	global "elastic/modules/global"
	"encoding/json"
	"fmt"
	"os/exec"
)

func DecisionCheckpoint(pauseContainerList []global.PauseContainer, checkpointList []global.CheckpointMetaData) []global.CheckpointMetaData {

	for _, pauseContainer := range pauseContainerList {
		if !pauseContainer.IsCheckpoint {
			checkpointList = append(checkpointList, CheckpointContainer(pauseContainer.PodName, *pauseContainer.ContainerData))
		}
	}

	return checkpointList
}

func CheckpointContainer(podName string, container global.ContainerData) global.CheckpointMetaData {
	var checkpointInfo map[string][]interface{}
	var checkpointMetaData global.CheckpointMetaData

	command := "curl -sk -X POST \"https://localhost:10250/checkpoint/default/" + podName + "/" + container.Name + "\""
	out, _ := exec.Command("bash", "-c", command).Output()

	json.Unmarshal([]byte(out), &checkpointInfo)

	// append checkpointMetaData
	checkpointMetaData.PodName = podName
	checkpointMetaData.PodName = container.Name
	checkpointMetaData.MemoryLimitInBytes = container.Cgroup.MemoryLimitInBytes
	checkpointMetaData.CheckpointName = string(checkpointInfo["items"][0].(string))

	return checkpointMetaData
}

func MakeContainerFromCheckpoint(checkpoint global.CheckpointMetaData) {
	// latest가 이미 있는 경우 기존 latest의 컨테이너 태그가 해제되고 새로 등록한 것이 latest가 됨
	// 굳이 삭제할 필요는 없을 것 같음
	command1 := ("newcontainer=$(sudo buildah from scratch) && sudo buildah add $newcontainer " + checkpoint.CheckpointName + " / && ")
	command2 := "sudo buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=" + checkpoint.ContainerName + " $newcontainer && "
	command3 := "sudo buildah commit $newcontainer " + checkpoint.ContainerName + ":latest && sudo buildah rm $newcontainer"

	// 정상적으로 처리되는 것은 확인하였음
	out1, err := exec.Command("bash", "-c", command1+command2+command3).Output()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(out1))
}

func RestoreContainer(checkpoint global.CheckpointMetaData) {
	// kubernetes master에 연결해서 명령어 보내야 할듯....
	// memory limit 등도 모두 지정해주어야 함
	command1 := "kubectl create -f - <<EOF\napiVersion: v1\nkind: Pod\nmetadata:\n  name: restore-nginx\n  "
	command2 := "labels:\n    app: nginx\nspec:\n  containers:\n  - name: " + checkpoint.ContainerName + "\n    image: localhost/" + checkpoint.ContainerName + ":latest\n  "
	command3 := "nodeName: " + global.NODENAME + "\nEOF"
	fmt.Println(command1 + command2 + command3)
	_, err := exec.Command("bash", "-c", command1+command2+command3).Output()
	if err != nil {
		fmt.Println(err)
	}
}
