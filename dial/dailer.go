package dial

import (
	"fmt"
	"time"

	"github.com/go-ping/ping"
)

//// Dialer is used to test connections
//type Dialer struct {
//	semaphore chan struct{}
//}
//
//// Status saves information about connection
//type Status struct {
//	Ok  bool
//	Err error
//}
//
//// NewDialer returns pointer to new Dialer
//func NewDialer(concurrentConnections int) *Dialer {
//	return &Dialer{
//		semaphore: make(chan struct{}, concurrentConnections),
//	}
//}
//
//// NewWorker is used to send address over NetAddressTimeout to make request and receive status over DialerStatus
//// Blocks until slot in semaphore channel for concurrency is free
//func (d *Dialer) NewWorker() (chan<- NetAddressTimeout, <-chan Status) {
//	netAddressTimeoutCh := make(chan NetAddressTimeout)
//	dialerStatusCh := make(chan Status)
//
//	d.semaphore <- struct{}{}
//	go func() {
//		netAddressTimeout := <-netAddressTimeoutCh
//		conn, err := net.DialTimeout(netAddressTimeout.Network, netAddressTimeout.Address, netAddressTimeout.Timeout)
//
//		dialerStatus := Status{}
//
//		if err != nil {
//			dialerStatus.Ok = false
//			dialerStatus.Err = err
//		} else {
//			dialerStatus.Ok = true
//			conn.Close()
//		}
//		dialerStatusCh <- dialerStatus
//		<-d.semaphore
//	}()
//
//	return netAddressTimeoutCh, dialerStatusCh
//}

// Dialer 用于测试连接
type Dialer struct {
	semaphore chan struct{}
}

// Status 保存关于连接的信息
type Status struct {
	Ok  bool
	Rss time.Duration
	Err error
}

// NewDialer 返回一个新的 Dialer 指针
func NewDialer(concurrentConnections int) *Dialer {
	return &Dialer{
		semaphore: make(chan struct{}, concurrentConnections),
	}
}

// NewWorker 用于发送地址进行 ping 请求，并接收状态
// 阻塞直到并发控制的信号量槽位可用
func (d *Dialer) NewWorker() (chan<- NetAddressTimeout, <-chan Status) {
	netAddressTimeoutCh := make(chan NetAddressTimeout)
	dialerStatusCh := make(chan Status)

	d.semaphore <- struct{}{} // 占用一个信号量槽位
	go func() {
		netAddressTimeout := <-netAddressTimeoutCh
		dialerStatus := Status{}

		// 使用 ping 库执行 ping 请求
		pinger, err := ping.NewPinger(netAddressTimeout.Address)
		if err != nil {
			dialerStatus.Ok = false
			dialerStatus.Err = err
			dialerStatusCh <- dialerStatus
			<-d.semaphore // 释放信号量槽位
			return
		}

		pinger.SetPrivileged(true)
		pinger.Timeout = netAddressTimeout.Timeout
		pinger.Count = 1 // 只 ping 一次
		err = pinger.Run()
		if err != nil {
			dialerStatus.Ok = false
			dialerStatus.Err = err
		}
		stats := pinger.Statistics()
		if stats.PacketsRecv > 0 {
			dialerStatus.Rss = stats.MaxRtt
			dialerStatus.Ok = true
		} else {
			dialerStatus.Ok = false
			dialerStatus.Err = fmt.Errorf("no reply")
		}

		dialerStatusCh <- dialerStatus
		<-d.semaphore // 释放信号量槽位
	}()

	return netAddressTimeoutCh, dialerStatusCh
}
