package connector

import (
	"github.com/cihub/seelog"
	"time"
	"encoding/json"
	"strconv"
	"gopkg.in/redis.v5"
)

type area struct {
	IsoCode string
	Name    string
	UserNum int
}

type UserLog struct {
	Uid        int64
	Ip         string
	Start_Time int64
	End_Time   int64
}

type User struct {
	Uid         int64
	Ip          string
	IsoCode     string
	CountryName string
	StartTime   int64
	EndTime     int64
}

type UserSet struct {
	hub      *Hub
	timeChan chan int
	userChan chan *User
	userSet  map[int64]*User
	areaSet  map[string]*area
	userNum  int
	pcu      int
}

const (
	REDIS_ONLINE_USER_KEY = "dwlog_stock_online_user"
	REDIS_ONLINE_USER_AREA_KEY = "dwlog_online_user_area"
	REDIS_CCU_KEY = "dwlog_ccu"
	REDIS_PCU_KEY = "dwlog_pcu" //当日最高在线人数
	MAX_CHECK_TIME = 360 //test 60, prod 360
	CHECK_TIME_AFTER = 60 //test 10, prod 60
	DATE_TIME_FORMAT = "200601021504"
	DATE_FORMAT = "20060102"
)

var today = time.Now().UTC().Format(DATE_FORMAT)
var oRedis = GetRedis()

func NewUserSet(hub *Hub) *UserSet {
	pcu := 0
	_pcu, err := oRedis.HGet(REDIS_PCU_KEY, today).Result()

	if err != nil {
		seelog.Error(err.Error())
	} else if err != redis.Nil {
		pcu, _ = strconv.Atoi(_pcu)
	}

	seelog.Debugf("PCU: %v", pcu)

	return &UserSet{
		hub: hub,
		timeChan:  make(chan int, 1),
		userChan:  make(chan *User, 1024),
		userSet:   make(map[int64]*User),
		areaSet:   make(map[string]*area),
		userNum: 0,
		pcu: pcu,
	}
}

func (s *UserSet)Run() {
	s.timeAfter()
	for {
		select {
		case user := <-s.userChan:
			if _, ok := s.userSet[user.Uid]; !ok {
				s.userNum ++
				s.userSet[user.Uid] = user

				if user.IsoCode != "" {
					if _, ok := s.areaSet[user.IsoCode]; !ok {
						s.areaSet[user.IsoCode] = &area{
							IsoCode: user.IsoCode,
							Name: user.CountryName,
							UserNum: 1,
						}
					} else {
						s.areaSet[user.IsoCode].UserNum ++
					}
				}

				seelog.Debugf("New Online User %v %v %v %v %v", user.Uid, user.Ip, user.IsoCode, user.CountryName, user.EndTime)
			} else {
				if s.userSet[user.Uid].EndTime < user.EndTime {
					s.userSet[user.Uid].EndTime = user.EndTime
				}
			}
		case <-s.timeChan:
			s.Analysis()
		}
	}
}

func (s *UserSet)Analysis() {
	s.timeAfter()
	seelog.Debug("Start check online user")

	now := time.Now()
	_today := now.UTC().Format(DATE_FORMAT)

	var checkTime = now.Unix() - MAX_CHECK_TIME
	for uid, user := range s.userSet {
		if (user.EndTime < checkTime) {
			s.userNum --
			delete(s.userSet, uid)
			if _, ok := s.areaSet[user.IsoCode]; ok {
				s.areaSet[user.IsoCode].UserNum --
			}
		}
	}
	seelog.Debugf("current online user: %v", s.userNum)

	bDiffDay := _today != today
	if (s.userNum > s.pcu || bDiffDay) {
		s.pcu = s.userNum
		oRedis.HSet(REDIS_PCU_KEY, _today, strconv.Itoa(s.pcu))
	}
	if (bDiffDay) {
		today = _today
	}

	seelog.Debugf("PCU: %v", s.pcu)

	dateTime := now.UTC().Format(DATE_TIME_FORMAT)
	totalData := map[string]interface{}{
		"date_time": dateTime,
		"total": s.userNum,
		"pcu": s.pcu,
		"area": s.areaSet,
	}

	sUserNum := strconv.Itoa(s.userNum)

	oRedis.Set(REDIS_ONLINE_USER_KEY, s.userNum, 0)
	oRedis.Set(REDIS_ONLINE_USER_AREA_KEY, s.json_encode(totalData), 0)
	oRedis.HSet(REDIS_CCU_KEY, dateTime, sUserNum)

	s.push(LOG_TYPE_ONLINE_USER, sUserNum)
	s.push(LOG_TYPE_ONLINE_USER_AREA, totalData)
}

func (s *UserSet)push(category string, data interface{}) {
	res := s.json_encode(map[string]interface{}{
		"data": data,
		"category": category,
	});

	if (res != nil) {
		seelog.Debugf("user online push: %v", string(res))
		s.hub.Broadcast <- &Msg{Category: category, Data:res}
	}
}

func (s *UserSet)json_encode(data interface{}) []byte {
	res, _ := json.Marshal(data)
	return res
}

func (s *UserSet)NewUser(user *User) {
	s.userChan <- user
}

func (s *UserSet)timeAfter() {
	time.AfterFunc(CHECK_TIME_AFTER * time.Second, func() {
		s.timeChan <- 1
	})
}
