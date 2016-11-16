package main

import (
	"bytes"
	"sort"
	"time"
)

type unix []int64

func (a unix) Len() int           { return len(a) }
func (a unix) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a unix) Less(i, j int) bool { return a[i] < a[j] }

func (u *User) newRoomTrackerRecord(room string) {
	trackMap := u.Moves.RoomTracker
	if len(trackMap) == 10 {
		var sortMe unix
		for k := range trackMap {
			sortMe = append(sortMe, k)
		}

		sort.Sort(unix(sortMe))
		//change to 4 from 0
		delete(trackMap, sortMe[0])

		trackMap[time.Now().Unix()] = room
	} else {
		trackMap[time.Now().Unix()] = room
	}

	u.Moves.RoomTracker = trackMap
}

func (u *User) getTrackedRooms() string {
	var buffer bytes.Buffer
	tracker := u.Moves.RoomTracker
	var sortMe unix
	for k := range tracker {
		sortMe = append(sortMe, k)
	}
	sort.Sort(unix(sortMe))
	for _, v := range sortMe {
		tm := time.Unix(v, 0)
		buffer.WriteString(" -> " + tm.Format("2006-01-02 15:04:05") + " : " + tracker[v])
	}
	return buffer.String()
}
