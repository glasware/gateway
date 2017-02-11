package core

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/pkg/errors"
)

func (e *entropy) doRandomMove(exits []exit) error {
	rand.Seed(time.Now().Unix())
	ex := exits[rand.Intn(len(exits))]

	if _, err := e.Conn.Write([]byte(fmt.Sprintf("%s\n", ex.direction))); err != nil {
		return errors.Wrap(err, "e.Conn.Write")
	}

	e.here = ex.destination
	return nil
}

func (e *entropy) wander(quit chan struct{}) {
	for {
		select {
		case <-quit:
			return
		default:
			if e._wander {
				e.roomMapMutex.Lock()
				defer e.roomMapMutex.Unlock()

				r, ok := e.roomMap[e.here]
				if !ok {
					e._wander = false
					os.Stdout.WriteString("I am lost...\n")
				}

				if err := e.doRandomMove(r.exits); err != nil {
					os.Stdout.WriteString(err.Error())
					return
				}
			}

			time.Sleep(time.Second * 8)
		}
	}
}