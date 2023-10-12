package meta

const EgressConfigKeyPrefix = "EgressConfig_"
const DomainConfigKeyPrefix = "DomainConfig_"

type EgressConfig struct {
	HostNamespace map[string]string `json:"host_namespace"`
}

type DomainConfig struct {
	Domain      string                  `json:"domain"`
	DownStreams map[int]*DownStreamInfo `json:"down_streams"`
	IsZero      bool                    `json:"is_zero"`

	Index int32 `json:"-"`
}

type DownStreamInfo struct {
	Addr        string `json:"addr"`
	IngressAddr string `json:"ingress_addr"`
}

func (s *EgressConfig) Clone() Cloneable {
	n := &EgressConfig{}
	n.HostNamespace = make(map[string]string)
	for k, v := range s.HostNamespace {
		n.HostNamespace[k] = v
	}
	return n
}

func (s *DomainConfig) Clone() Cloneable {
	n := &DomainConfig{}
	n.Domain = s.Domain
	n.IsZero = s.IsZero

	n.DownStreams = make(map[int]*DownStreamInfo)
	for k, v := range s.DownStreams {
		n.DownStreams[k] = v.Clone()
	}
	return n
}

func (s *DownStreamInfo) Clone() *DownStreamInfo {
	return &DownStreamInfo{
		Addr:        s.Addr,
		IngressAddr: s.IngressAddr,
	}
}
