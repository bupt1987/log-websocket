package connector

import (
	"github.com/cihub/seelog"
	"time"
	"encoding/json"
	"strconv"
	"gopkg.in/redis.v5"
	"io/ioutil"
	"os"
)

type area struct {
	IsoCode string
	Name    string
	UserNum int
}

type UserLog struct {
	Uid        int
	Ip         string
	Start_Time int
	End_Time   int
}

type User struct {
	Uid         int
	Ip          string
	IsoCode     string
	CountryName string
	StartTime   int
	EndTime     int
}

type UserSet struct {
	id        string
	dumpFiles map[string]string
	hub       *Hub
	timeChan  chan int
	userChan  chan *User
	userSet   map[int]*User
	areaSet   map[string]*area
	userNum   int
	pcu       int
}

const (
	REDIS_ONLINE_USER_KEY = "dwlog_stock_online_user"
	REDIS_ONLINE_USER_AREA_KEY = "dwlog_online_user_area"
	REDIS_CCU_KEY = "dwlog_ccu"
	REDIS_PCU_KEY = "dwlog_pcu" //当日最高在线人数
	MAX_CHECK_TIME = 360 //test 60, prod 360
	CHECK_TIME_AFTER = 10 //test 10, prod 60
	DATE_TIME_FORMAT = "200601021504"
	TIME_FORMAT = "2006-01-02 15:04:05"
	DATE_FORMAT = "20060102"
	SIG_ANALYSIS = 1
)

var today = time.Now().UTC().Format(DATE_FORMAT)
var oRedis = GetRedis()
var Root string

func init() {
	Root, _ = os.Getwd()
}

func NewUserSet(id string, hub *Hub) *UserSet {
	dumpFiles := map[string]string{
		"user": Root + "/dump_" + id + "_user.json",
		"area": Root + "/dump_" + id + "_area.json",
	}
	pcu := 0
	_pcu, err := oRedis.HGet(REDIS_PCU_KEY, today).Result()

	if err != nil {
		seelog.Error(err.Error())
	} else if err != redis.Nil {
		pcu, _ = strconv.Atoi(_pcu)
	}

	oUserSet := &UserSet{
		id: id,
		dumpFiles: dumpFiles,
		hub: hub,
		timeChan:  make(chan int, 1),
		userChan:  make(chan *User, 1024),
		userSet:   make(map[int]*User),
		areaSet:   make(map[string]*area),
		userNum: 0,
		pcu: pcu,
	}
	oUserSet.loadDump()

	seelog.Debugf("PCU: %v, CCU: %v", pcu, oUserSet.userNum)

	return oUserSet
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

				seelog.Debugf("New Online User %v %v %v %v %v", user.Uid, user.Ip, user.IsoCode, user.CountryName, time.Unix(int64(user.EndTime), 0).Format(TIME_FORMAT))
			} else {
				if s.userSet[user.Uid].EndTime < user.EndTime {
					s.userSet[user.Uid].EndTime = user.EndTime
				}
			}
		case <-s.timeChan:

			s.timeAfter()
			seelog.Debug("======================  Start check online user  ======================")

			now := time.Now()
			_today := now.UTC().Format(DATE_FORMAT)

			var checkTime = int(now.Unix() - MAX_CHECK_TIME)
			for uid, user := range s.userSet {
				if (user.EndTime < checkTime) {
					s.userNum --
					delete(s.userSet, uid)
					seelog.Debugf("user %v offline", uid)
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

			seelog.Debug("=================================  End  ===============================\n")
		}
	}
}

func (s *UserSet)Dump() {
	aUserData := make(map[string]User)
	for uid, user := range s.userSet {
		aUserData[strconv.Itoa(uid)] = *user
	}

	err := ioutil.WriteFile(s.dumpFiles["user"], s.json_encode(aUserData), 0644)
	if err != nil {
		seelog.Error(err);
	} else {
		seelog.Debugf("dump %v", s.dumpFiles["user"])
	}

	aAreaData := make(map[string]area)
	for iso, area := range s.areaSet {
		aAreaData[iso] = *area
	}

	err = ioutil.WriteFile(s.dumpFiles["area"], s.json_encode(aAreaData), 0644)
	if err != nil {
		seelog.Error(err);
	} else {
		seelog.Debugf("dump %v", s.dumpFiles["area"])
	}
}

func (s *UserSet)loadDump() {
	//load dump 文件
	for t, file := range s.dumpFiles {
		if _, err := os.Stat(file); err != nil {
			continue
		}
		res, err := ioutil.ReadFile(file)
		if (err != nil) {
			seelog.Errorf("load %v dump error: %v", file, err.Error())
			continue
		}

		if (len(res) == 0) {
			continue
		}

		switch t {
		case "user":
			data := make(map[string]User)
			err = json.Unmarshal(res, &data)
			if (err != nil) {
				seelog.Errorf("decode %v dump error", file)
				continue
			}

			for uid, user := range data {
				_uid, err := strconv.Atoi(uid)
				if err != nil {
					seelog.Errorf("load %v dump uid %v error", t, uid)
					continue
				}
				seelog.Debugf("load %v dump uid %v", t, _uid)
				s.userSet[_uid] = &User{
					Uid: user.Uid,
					Ip: user.Ip,
					IsoCode: user.IsoCode,
					CountryName: user.CountryName,
					StartTime: user.StartTime,
					EndTime: user.EndTime,
				}
			}
			break
		case "area":
			data := make(map[string]area)
			err = json.Unmarshal(res, &data)
			if (err != nil) {
				seelog.Errorf("decode %v dump error", file)
				continue
			}
			for ios, v := range data {
				seelog.Debugf("load %v dump area %v", t, ios)
				s.userNum += v.UserNum
				s.areaSet[ios] = &area{
					IsoCode: v.IsoCode,
					Name: v.Name,
					UserNum: v.UserNum,
				}
			}
			break
		default:
			continue
		}
	}
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
	res, err := json.Marshal(data)
	if (err != nil) {
		seelog.Errorf("json_encode error: %v", err.Error())
	}
	return res
}

func (s *UserSet)NewUser(user *User) {
	s.userChan <- user
}

func (s *UserSet)timeAfter() {
	time.AfterFunc(CHECK_TIME_AFTER * time.Second, func() {
		s.timeChan <- SIG_ANALYSIS
	})
}
