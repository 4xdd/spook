package neural

func capNativeWorkers(workers int) int {
	if workers > 1 {
		return 1
	}
	return workers
}
