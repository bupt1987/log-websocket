package connector

import (
	"github.com/cihub/seelog"
	"time"
	"encoding/json"
	"strconv"
)

type UserLog struct {
	Uid        int64
	Ip         string
	Start_Time int64
	End_Time   int64
}

type User struct {
	Uid            int64
	Ip             string
	CountryIsoCode string
	CountryName    string
	StartTime      int64
	EndTime        int64
}

type UserSet struct {
	hub      *Hub
	timeChan chan int
	userChan chan *User
	userSet  map[int64]*User
	userNum  int
}

const (
	REDIS_ONLINE_USER_KEY = "dwlog_stock_online_user"
	MAX_CHECK_TIME = 300
	CHECK_TIME_AFTER = 30
)

func NewUserSet(hub *Hub) *UserSet {
	return &UserSet{
		hub: hub,
		timeChan:  make(chan int, 1),
		userChan:  make(chan *User, 1024),
		userSet:   make(map[int64]*User),
		userNum: 0,
	}
}

func (s *UserSet)Run() {
	s.timeAfter()
	for {
		select {
		case user := <-s.userChan:
			var checkTime = time.Now().Unix() - MAX_CHECK_TIME
			if ( user.EndTime < checkTime) {
				continue
			}
			if _, ok := s.userSet[user.Uid]; !ok {
				s.userNum ++
				s.userSet[user.Uid] = user
				seelog.Debugf("New Online User %v %v %v %v %v", user.Uid, user.Ip, user.CountryIsoCode, user.CountryName, user.EndTime)
			} else {
				if s.userSet[user.Uid].EndTime < user.EndTime {
					s.userSet[user.Uid].EndTime = user.EndTime
				}
			}
		case <-s.timeChan:
			s.timeAfter()
			seelog.Debug("Start check online user")
			var checkTime = time.Now().Unix() - MAX_CHECK_TIME
			for uid, user := range s.userSet {
				if (user.EndTime < checkTime) {
					s.userNum --
					delete(s.userSet, uid)
				}
			}
			seelog.Debugf("current online user: %v", s.userNum)
			GetRedis().Set(REDIS_ONLINE_USER_KEY, s.userNum, 0)

			data, err := json.Marshal(map[string]string{"data": strconv.Itoa(s.userNum), "category": LOG_TYPE_ONLINE_USER})
			if (err == nil) {
				s.hub.Broadcast <- &Msg{Category: LOG_TYPE_ONLINE_USER, Data:data}
			}
		}
	}
}

func (s *UserSet)NewUser(user *User) {
	s.userChan <- user
}

func (s *UserSet)timeAfter() {
	time.AfterFunc(CHECK_TIME_AFTER * time.Second, func() {
		s.timeChan <- 1
	})
}
