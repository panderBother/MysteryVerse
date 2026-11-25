package health

import "time"

type Payload struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type Service struct {
	name    string
	version string
	now     func() time.Time
}

func New(name string, version string) *Service {
	return &Service{
		name:    name,
		version: version,
		now:     time.Now,
	}
}

func (s *Service) Live() Payload {
	return s.payload("live")
}

func (s *Service) Ready() Payload {
	return s.payload("ready")
}

func (s *Service) payload(status string) Payload {
	return Payload{
		Service:   s.name,
		Status:    status,
		Timestamp: s.now().UTC(),
		Version:   s.version,
	}
}
