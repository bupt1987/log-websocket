package analysis

import (
	"github.com/bupt1987/log-websocket/util"
	"container/list"
	"sync"
	"github.com/cihub/seelog"
)

const (
	REDIS_KEY_SESSION_LENGTH = "dwlog_session_length"
)

var sessions *list.List = new(list.List)
var lock = new(sync.RWMutex)

func LockAddSession() {
	lock.RLock()
}

func UnLockAddSession() {
	lock.RUnlock()
}

func AddSession(uid int, start int, end int) {
	sessions.PushFront(map[string]int{
		"uid": uid,
		"start": start,
		"length": end - start,
	})
}

func PushSession() {
	go func() {
		PushSessionImmediately()
	}()
}

func PushSessionImmediately() {
	lock.Lock()
	defer lock.Unlock()

	reids := util.GetRedis()
	for iter := sessions.Front(); iter != nil; iter = iter.Next() {
		reids.LPush(REDIS_KEY_SESSION_LENGTH, util.JsonEncode(iter.Value))
	}
	sessions.Init()
	if util.IsDev() {
		seelog.Debug("Session length pushed")
	}
}
