package analysis

import (
	"github.com/cihub/seelog"
	"time"
)

type UserLog struct {
	Uid  int64
	Ip   string
	Time int64
}

type OnlineUser struct {
	Uid            int64
	Ip             string
	CountryIsoCode string
	CountryName    string
	Time           int64
}

var userChan = make(chan *OnlineUser, 100)
var users = make(map[int64]*OnlineUser)
var onlineUserNum = 0

func Run() {
	for {
		select {
		case user := <-userChan:
			if _, ok := users[user.Uid]; !ok {
				onlineUserNum ++
				users[user.Uid] = user
			} else {
				if users[user.Uid].Time < user.Time {
					users[user.Uid] = user
				}
			}
			seelog.Infof("%v %v %v %v %v", user.Uid, user.Ip, user.CountryIsoCode, user.CountryName, user.Time)
		case <-time.After(time.Second * 60):
			seelog.Info("Start check online user")

			var checkTime = time.Now().Unix() - 30
			for uid, user := range users {
				if (user.Time < checkTime) {
					onlineUserNum --
					delete(users, uid)
					seelog.Infof("user %v off line", uid)
				}
			}

			seelog.Infof("current online user: %v", onlineUserNum)
		}
	}
}

func ActiveUser(user *OnlineUser) {
	userChan <- user
}
