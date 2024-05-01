package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func main() {
	workloads := []string{"parsec", "splash2", "splash2x"}
	/*parsec := []string{"blackscholes", "bodytrack", "canneal", "cmake", "dedup", "facesim", "ferret"}
	"fluidanimate", "freqmine", "glib", "gsl", "hooks", "libjpeg", "libtool", "libxml2", "mesa", "netdedup",
	"netferret", "netstreamcluster", "parmacs", "raytrace", "ssl", "streamcluster", "swaptions", "tbblib",
	"uptcpip", "vips", "x264", "yasm", "zlib"}*/
	splash2 := []string{"barnes", "cholesky", "fft", "fmm", "lu_cb", "lu_ncb", "ocean_cp", "ocean_ncp",
		"radiosity", "radix", "raytrace", "volrend", "water_nsquared", "water_spatial"}
	splash2name := []string{"barnes", "cholesky", "fft", "fmm", "lu-cb", "lu-ncb", "ocean-cp", "ocean-ncp",
		"radiosity", "radix", "raytrace", "volrend", "water-nsquared", "water-spatial"}

	/*for _, p := range parsec {
		var command string
		//command = "cd /home/limitstory-virtual/workload && "
		command = "cd /home/limitstory/cloud/workload && "
		command = command + "kubectl apply -f " + workloads[0] + "-" + p + ".yaml"

		cmd := exec.Command("bash", "-c", command)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()

		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		}
	}*/
	/*
		for i, _ := range splash2 {
			var command string
			// command = "cd /home/limitstory-virtual/workload && "
			command = "cd /home/limitstory/cloud/workload && "
			command = command + "kubectl apply -f " + workloads[1] + "-" + splash2name[i] + ".yaml"

			cmd := exec.Command("bash", "-c", command)
			var out bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err := cmd.Run()

			if err != nil {
				fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			}
		}*/
	for i, _ := range splash2 {
		var command string
		// command = "cd /home/limitstory-virtual/workload && "
		command = "cd /home/limitstory/cloud/workload && "
		command = command + "kubectl apply -f " + workloads[2] + "-" + splash2name[i] + ".yaml"

		cmd := exec.Command("bash", "-c", command)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()

		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		}
	}
}
