package main

import (
	"fmt"
	"os"
)

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func main() {
	images := "spirals/parsec-3.0:latest"
	workloads := []string{"parsec", "splash2", "splash2x"}
	parsec := []string{"blackscholes", "bodytrack", "canneal", "cmake", "dedup", "facesim", "ferret",
		"fluidanimate", "freqmine", "glib", "gsl", "hooks", "libjpeg", "libtool", "libxml2", "mesa", "netdedup",
		"netferret", "netstreamcluster", "parmacs", "raytrace", "ssl", "streamcluster", "swaptions", "tbblib",
		"uptcpip", "vips", "x264", "yasm", "zlib"}
	splash2 := []string{"barnes", "cholesky", "fft", "fmm", "lu_cb", "lu_ncb", "ocean_cp", "ocean_ncp",
		"radiosity", "radix", "raytrace", "volrend", "water_nsquared", "water_spatial"}
	splash2name := []string{"barnes", "cholesky", "fft", "fmm", "lu-cb", "lu-ncb", "ocean-cp", "ocean-ncp",
		"radiosity", "radix", "raytrace", "volrend", "water-nsquared", "water-spatial"}

	for _, p := range parsec {
		//str := "/home/limitstory-virtual/workload/" + workloads[0] + "-" + p + ".yaml"
		str := "/home/limitstory/cloud/workload/" + workloads[0] + "-" + p + ".yaml"
		command1 := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: " + workloads[0] + "-" + p + "\n  "
		command2 := "labels:\n    app: " + p + "\nspec:\n  containers:\n  - name: " + p + "\n    "
		command3 := "image: docker.io/" + images + "\n    imagePullPolicy: IfNotPresent    \n    "
		command4 := "args: [\"-a\", \"run\", \"-p\", \"parsec." + p + "\", \"-i\", \"native\"]    \n    "
		command5 := "resources:\n      requests:\n        memory: 10000Mi\n      limits:\n        memory: 16000Mi\n  "
		command6 := "restartPolicy: OnFailure"
		f, err := os.Create(str)
		checkError(err)
		defer f.Close()
		fmt.Fprintf(f, string(command1+command2+command3+command4+command5+command6))
	}

	/*
		for i, p := range splash2 {
			//str := "/home/limitstory-virtual/workload/" + workloads[1] + "-" + splash2name[i] + ".yaml"
			str := "/home/limitstory/cloud/workload/" + workloads[1] + "-" + splash2name[i] + ".yaml"
			command1 := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: " + workloads[1] + "-" + splash2name[i] + "\n  "
			command2 := "labels:\n    app: " + splash2name[i] + "\nspec:\n  containers:\n  - name: " + splash2name[i] + "\n    "
			command3 := "image: docker.io/" + images + "\n    imagePullPolicy: IfNotPresent    \n    "
			command4 := "args: [\"-a\", \"run\", \"-p\", \"splash2." + p + "\", \"-i\", \"native\"]    \n    "
			command5 := "resources:\n      requests:\n        memory: 10000Mi\n      limits:\n        memory: 16000Mi\n  "
			command6 := "restartPolicy: OnFailure"
			f, err := os.Create(str)
			checkError(err)
			defer f.Close()
			fmt.Fprintf(f, string(command1+command2+command3+command4+command5+command6))
		}*/

	for i, p := range splash2 {
		//str := "/home/limitstory-virtual/workload/" + workloads[2] + "-" + splash2name[i] + ".yaml"
		str := "/home/limitstory/cloud/workload/" + workloads[2] + "-" + splash2name[i] + ".yaml"
		command1 := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: " + workloads[2] + "-" + splash2name[i] + "\n  "
		command2 := "labels:\n    app: " + splash2name[i] + "\nspec:\n  containers:\n  - name: " + splash2name[i] + "\n    "
		command3 := "image: docker.io/" + images + "\n    imagePullPolicy: IfNotPresent    \n    "
		command4 := "args: [\"-a\", \"run\", \"-p\", \"splash2x." + p + "\", \"-i\", \"native\"]    \n    "
		command5 := "resources:\n      requests:\n        memory: 10000Mi\n      limits:\n        memory: 16000Mi\n  "
		command6 := "restartPolicy: OnFailure"
		f, err := os.Create(str)
		checkError(err)
		defer f.Close()
		fmt.Fprintf(f, string(command1+command2+command3+command4+command5+command6))
	}
}
