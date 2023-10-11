package meta

type MetaObject struct {
	Services map[string]*ServiceInfo
}
type ServiceInfo struct {
	Name string
}

type Cloneable interface {
	Clone() interface{}
}

func (s *MetaObject) Clone() interface{} {
	n := &MetaObject{}

	n.Services = make(map[string]*ServiceInfo)
	for k, v := range s.Services {
		n.Services[k] = v.Clone()
	}
	return n
}

func (s *ServiceInfo) Clone() *ServiceInfo {
	return &ServiceInfo{
		Name: s.Name,
	}
}
