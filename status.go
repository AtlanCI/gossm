package gossm

import (
	"sync"
	"time"

	"github.com/AtlanCI/gossm/conf"
)

type ServerStatusData struct {
	rwmu         sync.RWMutex
	ServerStatus map[*conf.Server][]*statusAtTime `json:"serverStatus"`
}

type statusAtTime struct {
	Time time.Time `json:"time"`
	// bool represent server online or offline
	Status bool          `json:"online"`
	TSS    time.Duration `json:"tss"`
}

func NewServerStatusData(servers conf.Servers) *ServerStatusData {
	serverStatusData := &ServerStatusData{
		ServerStatus: make(map[*conf.Server][]*statusAtTime),
	}

	for _, server := range servers {
		serverStatusData.ServerStatus[server] = make([]*statusAtTime, 0, 100)
	}

	return serverStatusData
}

// SetStatusAtTimeForServer updates map with new entry containing current time and server status at that time
func (s *ServerStatusData) SetStatusAtTimeForServer(server *conf.Server, timeNow time.Time, status bool, RTT time.Duration) {
	s.rwmu.Lock()
	defer s.rwmu.Unlock()
	s.ServerStatus[server] = append(s.ServerStatus[server], &statusAtTime{Time: timeNow, Status: status, TSS: RTT})
}

func (s *ServerStatusData) GetServerStatus() map[*conf.Server][]*statusAtTime {
	s.rwmu.RLock()
	defer s.rwmu.RUnlock()
	return s.ServerStatus
}
