package meta

const IngressKeyPrefix = "IngressConfig_"

type IngressConfig struct {
	HostInfo map[string]*IngressHostInfo `json:"host_info"`
}

type IngressHostInfo struct {
	Addr             string `json:"addr"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
	QueueSource      string `json:"queue_source"`
}

func (c *IngressConfig) Clone() Cloneable {
	n := &IngressConfig{HostInfo: make(map[string]*IngressHostInfo)}
	if c.HostInfo == nil {
		return n
	}
	for k, v := range c.HostInfo {
		n.HostInfo[k] = &IngressHostInfo{
			Addr:             v.Addr,
			ConcurrencyLimit: v.ConcurrencyLimit,
			QueueSource:      v.QueueSource,
		}
	}
	return n
}
