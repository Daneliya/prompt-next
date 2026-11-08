package desktop

type MemoryUsage struct {
	AppBytes   uint64 `json:"appBytes"`
	TotalBytes uint64 `json:"totalBytes"`
}

func (s *PromptService) MemoryUsage() MemoryUsage {
	return currentMemoryUsage()
}
