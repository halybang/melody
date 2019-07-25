package melody

import (
	//"io"
	"log"
	"net"
	"sync"
	"syscall"

	"github.com/smallnest/epoller"
	//"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type pool struct {
	workers   int
	maxTasks  int
	taskQueue chan net.Conn

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

func newPool(w int, t int) *pool {
	return &pool{
		workers:   w,
		maxTasks:  t,
		taskQueue: make(chan net.Conn, t),
		done:      make(chan struct{}),
	}
}

func (p *pool) Close() {
	p.mu.Lock()
	p.closed = true
	close(p.done)
	close(p.taskQueue)
	p.mu.Unlock()
}

func (p *pool) addTask(conn net.Conn) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.taskQueue <- conn
}

func (p *pool) start() {
	for i := 0; i < p.workers; i++ {
		go p.startWorker()
	}
}

func (p *pool) startWorker() {
	for {
		select {
		case <-p.done:
			return
		case conn := <-p.taskQueue:
			if conn != nil {
				handleConn(conn)
			}
		}
	}
}

func handleConn(conn net.Conn) {
	//_, err := io.CopyN(conn, conn, 8)
	// if err != nil {
	// 	if err := epoller.Remove(conn); err != nil {
	// 		log.Printf("failed to remove %v", err)
	// 	}
	// 	conn.Close()
	// }
	// opsRate.Mark(1)
	// if _, _, err := wsutil.ReadClientData(conn); err != nil {
	// 	if err := poller.Remove(conn); err != nil {
	// 		log.Printf("Failed to remove %v", err)
	// 	}
	// 	conn.Close()
	// } else {
	// 	// This is commented out since in demo usage, stdout is showing messages sent from > 1M connections at very high rate
	// 	//log.Printf("msg: %s", string(msg))
	// }
	msg, op, err := wsutil.ReadClientData(conn)
	if err != nil {
		// handle error
		conn.Close()
		poller.Remove(conn)
		return
	}
	// Echo everything
	err = wsutil.WriteServerMessage(conn, op, msg)
	if err != nil {
		// handle error
	}
}

var poller epoller.Poller
var workerPool *pool

func InitPooler() error {

	works := 10
	connections := 1000000
	workerPool = newPool(works, connections)
	workerPool.start()

	poller, err := epoller.NewPoller()
	if err != nil {
		log.Fatalf("failed to create a poller %v", err)
		return err
	}
	go start(poller, true)
	return nil
}

func setLimit() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}

	log.Printf("set cur limit: %d", rLimit.Cur)
}

func start(poller epoller.Poller, use_pool bool) {
	for {
		connections, err := poller.Wait(100)
		if err != nil {
			log.Printf("failed to epoll wait %v", err)
			continue
		}
		if use_pool == true {
			for _, conn := range connections {
				if conn == nil {
					continue
				}
				workerPool.addTask(conn)
			}
		} else {
			for _, conn := range connections {
				msg, op, err := wsutil.ReadClientData(conn)
				if err != nil {
					// handle error
					conn.Close()
					poller.Remove(conn)
					return
				}
				// Echo everything
				err = wsutil.WriteServerMessage(conn, op, msg)
				if err != nil {
					// handle error
				}
				// mu.RLock()
				// r := readers[conn]
				// w := writers[conn]
				// mu.RUnlock()

				// if r == nil || w == nil {
				// 	continue
				// }

				// cmds, err := r.ReadCommands()
				// if err != nil {
				// 	closeConn(conn)
				// 	continue
				// }

				// for _, cmd := range cmds {
				// 	handleCmd(conn, &cmd, w)
				// }
			}
		}
	}
}
