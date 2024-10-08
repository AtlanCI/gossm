package gossm

import (
	"os"
	"time"

	"github.com/AtlanCI/gossm/conf"
	"github.com/AtlanCI/gossm/dial"
	"github.com/AtlanCI/gossm/logger"
	"github.com/AtlanCI/gossm/track"
)

type Monitor struct {
	// Holds settings and servers
	config *conf.Config

	// Channel used to schedule checks for servers
	checkerCh chan *conf.Server

	// Channel used for receive servers that couldn't be reached
	notifierCh chan *conf.Server

	// To reduce notification spam, tracker is used to delay notifications
	notificationTracker map[*conf.Server]*track.TimeTracker

	// Used to test connections
	dialer *dial.Dialer

	// Sending to stop channel makes program exit
	stop chan struct{}

	// TODO: For each server, keep map with time and up/down status
	serverStatusData *ServerStatusData
}

func NewMonitor(c *conf.Config) *Monitor {
	m := &Monitor{
		config:              c,
		checkerCh:           make(chan *conf.Server),
		notifierCh:          make(chan *conf.Server),
		notificationTracker: make(map[*conf.Server]*track.TimeTracker),
		dialer:              dial.NewDialer(c.Settings.Monitor.MaxConnections),
		stop:                make(chan struct{}),
		serverStatusData:    NewServerStatusData(c.Servers),
	}
	m.initialize()
	return m
}

func (m *Monitor) initialize() {
	for _, server := range m.config.Servers {
		// Initialize notificationTracker
		m.notificationTracker[server] = NewTrackerWithExpBackoff(m.config.Settings.Monitor.ExponentialBackoffSeconds)

		// Set default CheckInterval and Timeout for servers who miss them
		switch {
		case server.CheckInterval <= 0:
			server.CheckInterval = m.config.Settings.Monitor.CheckInterval
		case server.Timeout <= 0:
			server.Timeout = m.config.Settings.Monitor.Timeout
		}
	}
}

// NewTrackerWithExpBackoff creates TimeTracker with ExpBackoff as Delayer
func NewTrackerWithExpBackoff(expBackoffSeconds int) *track.TimeTracker {
	return track.NewTracker(track.NewExpBackoff(expBackoffSeconds))
}

// Run runs monitor infinitely
func (m *Monitor) Run() {
	m.RunForSeconds(0)
}

// RunForSeconds runs monitor for runningSeconds seconds or infinitely if 0 is passed as an argument
func (m *Monitor) RunForSeconds(runningSeconds int) {
	if runningSeconds != 0 {
		go func() {
			runningSecondsTime := time.Duration(runningSeconds) * time.Second
			<-time.After(runningSecondsTime)
			m.stop <- struct{}{}
		}()
	}

	for _, server := range m.config.Servers {
		go m.scheduleServer(server)
	}

	logger.Logln("Starting monitor.")
	m.monitor()
}

func (m *Monitor) scheduleServer(s *conf.Server) {
	// Initial
	m.checkerCh <- s

	// Periodic
	tickerSeconds := time.NewTicker(time.Duration(s.CheckInterval) * time.Second)
	for range tickerSeconds.C {
		m.checkerCh <- s
	}
}

func (m *Monitor) monitor() {
	go m.listenForChecks()
	go m.listenForNotifications()

	// Wait for termination signal then exit monitor
	<-m.stop
	logger.Logln("Terminating.")
	os.Exit(0)
}

func (m *Monitor) listenForChecks() {
	for server := range m.checkerCh {
		m.checkServerStatus(server)
	}
}

func (m *Monitor) listenForNotifications() {
	for server := range m.notifierCh {
		timeTracker := m.notificationTracker[server]
		if timeTracker.IsReady() {

		}
	}
}

func (m *Monitor) checkServerStatus(server *conf.Server) {
	// NewWorker() blocks if there aren't free slots in dialer for concurrency
	worker, output := m.dialer.NewWorker()
	go func() {
		logger.Logln("Checking", server)

		formattedAddress := server.IPAddress
		timeoutSeconds := time.Duration(server.Timeout) * time.Second
		worker <- dial.NetAddressTimeout{NetAddress: dial.NetAddress{Address: formattedAddress}, Timeout: timeoutSeconds}
		dialerStatus := <-output

		m.serverStatusData.SetStatusAtTimeForServer(server, time.Now(), dialerStatus.Ok, dialerStatus.Rss)

		// Handle error
		if !dialerStatus.Ok {
			logger.Logln(dialerStatus.Err)
			logger.Logln("ERROR", server)
			go func() {
				m.notifierCh <- server
			}()
			return
		}

		// Handle success
		logger.Logln("OK", server)
		// Reset time tracker for server
		if m.notificationTracker[server].HasBeenRan() {
			m.notificationTracker[server] = NewTrackerWithExpBackoff(m.config.Settings.Monitor.ExponentialBackoffSeconds)
		}
	}()
}
