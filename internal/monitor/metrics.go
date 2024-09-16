package monitor

import "github.com/docker/docker/api/types/container"

type containerMetrics struct {
	ContainerID string
	MemUsed     uint64
	MemAvail    uint64
	CpuPerc     float64
	CpuMaxPerc  float64
}

func getMetrics(containerStats container.StatsResponse) *containerMetrics {
	memUsed := containerStats.MemoryStats.Usage
	memAvail := containerStats.MemoryStats.Limit

	// Adjust CPU percentage calculation
	cpuDelta := float64(containerStats.CPUStats.CPUUsage.TotalUsage - containerStats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(containerStats.CPUStats.SystemUsage - containerStats.PreCPUStats.SystemUsage)
	numCpus := float64(containerStats.CPUStats.OnlineCPUs)
	cpuPerc := 0.0

	if cpuDelta > 0.0 && systemDelta > 0.0 {
		cpuPerc = (cpuDelta / systemDelta) * numCpus * 100
	}

	cpuMaxPerc := numCpus * 100

	return &containerMetrics{
		ContainerID: containerStats.ID,
		MemUsed:     memUsed,
		MemAvail:    memAvail,
		CpuPerc:     cpuPerc,
		CpuMaxPerc:  cpuMaxPerc,
	}
}
