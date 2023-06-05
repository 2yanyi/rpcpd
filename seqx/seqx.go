// Source code file, created by Developer@YANYINGSONG.

package seqx

import (
	"errors"
	"sync"
	"time"
)

/**
 * 机器唯一序列号，一种雪花算法，多机器场景中要设置不同的 machineId。
 */

// 毫秒内的序号自增最大值
const seqMax = 10000

var startTime = time.Now().UnixMilli()

type SeqX struct {
	machine int64 // Machine ID
	seq     int64 // Sequence NO
	unix    int64 // Unix time of milliseconds
	lock    *sync.Mutex
}

func New(machineId int64) *SeqX {
	time.Sleep(time.Second)
	return &SeqX{
		machine: machineId,
		lock:    &sync.Mutex{},
	}
}

func (f *SeqX) NextID() int64 {
	seq := f.nextID()
	if seq == 0 {
		return f.NextID()
	}

	return seq
}

func (f *SeqX) nextID() int64 {
	f.lock.Lock()
	defer f.lock.Unlock()

	currentTime := time.Now().UnixMilli()
	if currentTime < f.unix {

		// !!!
		err := errors.New("error: The system clock was set back")
		println(err.Error())

		f.unix = currentTime
		return 0
	}

	// Generated within the same millisecond.
	if f.unix == currentTime {
		if f.seq >= seqMax {

			// Wait one millisecond and refetch the currentTime.
			for currentTime <= f.unix {
				currentTime = time.Now().UnixMilli()
			}
		}
		f.seq++
	} else {
		f.seq = 0
	}

	// update time.
	f.unix = currentTime

	// Generate a unique ID.
	id := (currentTime - startTime) << 22
	id |= f.machine << 11
	id |= f.seq

	return id
}

func Init(machineId int64) {
	X = New(machineId)
}

var X *SeqX
