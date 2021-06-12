package rest

import "sync"

type numeralWaitGroup struct {
	cnt int
	cd  *sync.Cond
	lk  sync.RWMutex
}

func newNWG() *numeralWaitGroup {
	res := &numeralWaitGroup{}
	res.cd = sync.NewCond(res.lk.RLocker())
	return res
}

func (n *numeralWaitGroup) Add(delta int) {
	n.lk.Lock()
	defer n.lk.Unlock()

	n.cnt += delta

	if delta < 0 {
		n.cd.Broadcast()
	}
}

func (n *numeralWaitGroup) Done() {
	n.Add(-1)
}

// Wait waits until there is less or equal than "min" tasks
func (n *numeralWaitGroup) Wait(min int) {
	n.lk.RLock()
	defer n.lk.RUnlock()

	for {
		if n.cnt <= min {
			return
		}
		n.cd.Wait()
	}
}
