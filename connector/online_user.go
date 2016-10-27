package connector

import (
	"github.com/cihub/seelog"
	"time"
	"encoding/json"
	"strconv"
	"gopkg.in/redis.v5"
	"io/ioutil"
	"os"
	"net"
	"github.com/bupt1987/log-websocket/util"
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
	Id    int
	Iso   string
	CName string
	STime int
	ETime int
}

type UserSet struct {
	id        string
	mDumpFile map[string]string
	oHub      *Hub
	cTime     chan int
	cDump     chan *chan int
	cUser     chan *User
	mUser     map[int]*User
	mArea     map[string]*area
	iUserNum  int
	iPcu      int
}

const (
	REDIS_ONLINE_USER_KEY = "dwlog_stock_online_user"
	REDIS_ONLINE_USER_AREA_KEY = "dwlog_online_user_area"
	REDIS_CCU_KEY = "dwlog_ccu"
	REDIS_PCU_KEY = "dwlog_pcu" //当日最高在线人数
	MAX_CHECK_TIME = 420 //test 60, prod 420
	DATE_TIME_FORMAT = "200601021504"
	DATE_FORMAT = "20060102"
)

var today = time.Now().UTC().Format(DATE_FORMAT)

func NewUserSet(id string, hub *Hub) *UserSet {
	root, _ := os.Getwd()
	dumpFiles := map[string]string{
		"user": root + "/dump_" + id + "_user.json",
		"area": root + "/dump_" + id + "_area.json",
	}
	pcu := 0
	_pcu, err := util.GetRedis().HGet(REDIS_PCU_KEY, today).Result()

	if err == redis.Nil {
	} else if err != nil {
		seelog.Errorf("get pcu error: %v", err.Error())
	} else {
		pcu, _ = strconv.Atoi(_pcu)
	}

	oUserSet := &UserSet{
		id: id,
		mDumpFile: dumpFiles,
		oHub: hub,
		cTime: make(chan int, 1),
		cDump: make(chan *chan int, 1),
		cUser: make(chan *User, 1024),
		mUser: make(map[int]*User),
		mArea: make(map[string]*area),
		iUserNum: 0,
		iPcu: pcu,
	}
	oUserSet.loadDump()

	seelog.Infof("pcu: %v, ccu: %v", pcu, oUserSet.iUserNum)

	return oUserSet
}

func (s *UserSet)Run() {
	go func() {
		defer util.PanicExit()

		after := 59 - time.Now().Second()
		s.timeAfter(after)
		oRedis := util.GetRedis()
		for {
			select {
			case user := <-s.cUser:
				if _, ok := s.mUser[user.Id]; !ok {
					s.iUserNum ++
					s.mUser[user.Id] = user

					if user.Iso != "" {
						if _, ok := s.mArea[user.Iso]; !ok {
							s.mArea[user.Iso] = &area{
								IsoCode: user.Iso,
								Name: user.CName,
								UserNum: 1,
							}
						} else {
							s.mArea[user.Iso].UserNum ++
						}
					}
					//seelog.Debugf("New Online User %v %v %v %v",
					//	user.Id,
					//	user.Iso,
					//	user.CName,
					//	time.Unix(int64(user.ETime), 0).Format("2006-01-02 15:04:05"),
					//)
				} else {
					if s.mUser[user.Id].ETime < user.ETime {
						s.mUser[user.Id].ETime = user.ETime
					}
				}

			case <-s.cTime:
				seelog.Debug("======================  Start check online user  ======================")

				now := time.Now()
				_today := now.UTC().Format(DATE_FORMAT)
				iDiffTime := MAX_CHECK_TIME
				if util.IsDev() {
					iDiffTime = 60
				}
				checkTime := int(now.Unix() - int64(iDiffTime))
				iOffLine := 0

				for uid, user := range s.mUser {
					if (user.ETime < checkTime) {
						s.iUserNum --
						iOffLine ++
						delete(s.mUser, uid)
						if _, ok := s.mArea[user.Iso]; ok {
							s.mArea[user.Iso].UserNum --
						}
					}
				}

				bDiffDay := _today != today
				if (s.iUserNum > s.iPcu || bDiffDay) {
					s.iPcu = s.iUserNum
					oRedis.HSet(REDIS_PCU_KEY, _today, strconv.Itoa(s.iPcu))
				}
				if (bDiffDay) {
					today = _today
				}

				seelog.Debugf("Offline user: %v, current online user: %v, Pcu: %v", iOffLine, s.iUserNum, s.iPcu)

				dateTime := now.UTC().Format(DATE_TIME_FORMAT)
				totalData := map[string]interface{}{
					"date_time": dateTime,
					"total": s.iUserNum,
					"pcu": s.iPcu,
					"area": s.mArea,
				}

				sUserNum := strconv.Itoa(s.iUserNum)

				oRedis.Set(REDIS_ONLINE_USER_KEY, s.iUserNum, 0)
				oRedis.Set(REDIS_ONLINE_USER_AREA_KEY, s.json_encode(totalData), 0)
				oRedis.HSet(REDIS_CCU_KEY, dateTime, sUserNum)

				s.push(LOG_TYPE_ONLINE_USER, sUserNum)
				s.push(LOG_TYPE_ONLINE_USER_AREA, totalData)

				seelog.Debug("=================================  End  ===============================")

			case cDumpEnd := <-s.cDump:
				seelog.Info("Start to dump data")
				aUserData := make(map[string]*User)
				for uid, user := range s.mUser {
					aUserData[strconv.Itoa(uid)] = user
				}

				err := ioutil.WriteFile(s.mDumpFile["user"], s.json_encode(aUserData), 0644)
				if err != nil {
					seelog.Error(err);
				} else {
					seelog.Infof("dump %v", s.mDumpFile["user"])
				}

				err = ioutil.WriteFile(s.mDumpFile["area"], s.json_encode(s.mArea), 0644)
				if err != nil {
					seelog.Error(err);
				} else {
					seelog.Infof("dump %v", s.mDumpFile["area"])
				}
				*cDumpEnd <- 1
				seelog.Info("Dump data finished")
				break
			}
		}
	}()
}

func (s *UserSet)Dump() {
	cDumpEnd := make(chan int, 1)
	s.cDump <- &cDumpEnd
	<- cDumpEnd
}

func (s *UserSet)loadDump() {
	iStartTime := time.Now().UnixNano() / int64(time.Millisecond)
	//load dump 文件
	for t, file := range s.mDumpFile {
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
			iUserNum := 0
			for uid, user := range data {
				_uid, err := strconv.Atoi(uid)
				if err != nil {
					seelog.Errorf("load %v dump uid %v error", t, uid)
					continue
				}
				iUserNum ++;
				s.mUser[_uid] = &User{
					Id: user.Id,
					Iso: user.Iso,
					CName: user.CName,
					STime: user.STime,
					ETime: user.ETime,
				}
			}
			s.iUserNum = iUserNum
			seelog.Infof("%v load finished", file)
			break
		case "area":
			data := make(map[string]area)
			err = json.Unmarshal(res, &data)
			if (err != nil) {
				seelog.Errorf("decode %v dump error", file)
				continue
			}
			iUserNum := 0
			for ios, v := range data {
				iUserNum += v.UserNum
				s.mArea[ios] = &area{
					IsoCode: v.IsoCode,
					Name: v.Name,
					UserNum: v.UserNum,
				}
			}
			seelog.Infof("%v load finished, Area user: %v", file, iUserNum)
			break
		default:
			continue
		}

		if err := os.Remove(file); err != nil {
			seelog.Errorf("delete dump file error: %v", err.Error())
		}
	}

	seelog.Infof("Load dump cost: %vms", time.Now().UnixNano() / int64(time.Millisecond) - iStartTime)
}

func (s *UserSet)push(category string, data interface{}) {
	res := s.json_encode(map[string]interface{}{
		"data": data,
		"category": category,
	});

	if (res != nil) {
		seelog.Debugf("user online push: %v", string(res))
		s.oHub.Broadcast <- &Msg{Category: category, Data:res}
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
	s.cUser <- user
}

func (s *UserSet)timeAfter(after int) {
	if (after <= 0) {
		after = 60
	}
	seelog.Debugf("Check online user will run after %vs", after)
	time.AfterFunc(time.Duration(after) * time.Second, func() {
		s.timeAfter(60)
		s.cTime <- 1
	})
}

type OnlineUserMessage struct {
	UserSet *UserSet
}

func (m *OnlineUserMessage) Process(msg *Msg, conn *net.Conn) {
	var isoCode = ""
	var countryName = ""
	userLog := UserLog{}
	json.Unmarshal(msg.Data, &userLog)

	if userLog.Ip != "" && userLog.Ip != "unknown" {
		ip := net.ParseIP(userLog.Ip)
		city, err := util.GetGeoIp().City(ip)
		if err != nil {
			seelog.Errorf("geoip '%v' error: %v", userLog.Ip, err.Error())
		} else if city.Country.IsoCode != "" {
			isoCode = city.Country.IsoCode
			countryName = city.Country.Names["en"]
		}
	}

	m.UserSet.NewUser(&User{
		Id: userLog.Uid,
		Iso: isoCode,
		CName: countryName,
		STime: userLog.Start_Time,
		ETime: userLog.End_Time,
	})
}
