/*
 *     Copyright 2020 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"sync"
	"time"

	"d7y.io/dragonfly/v2/internal/dferrors"
	"d7y.io/dragonfly/v2/internal/rpc/base"
)

type Task struct {
	taskID         string
	URL            string
	Filter         string
	BizID          string
	URLMata        *base.UrlMeta
	SizeScope      base.SizeScope
	DirectPiece    []byte
	CreateTime     time.Time
	LastAccessTime time.Time
	PieceList      map[int32]*Piece
	PieceTotal     int32
	ContentLength  int64
	Statistic      *TaskStatistic
	CDNError       *dferrors.DfError
}

func (t *Task) InitProps() {
	if t.PieceList == nil {
		t.CreateTime = time.Now()
		t.LastAccessTime = t.CreateTime
		t.SizeScope = base.SizeScope_NORMAL
		t.Statistic = &TaskStatistic{
			StartTime: time.Now(),
		}
	}
}

func (t *Task) GetPiece(pieceNum int32) *Piece {
	return t.PieceList[pieceNum]
}

func (t *Task) GetOrCreatePiece(pieceNum int32) *Piece {
	t.rwLock.RLock()
	p := t.PieceList[pieceNum]
	if p == nil {
		t.rwLock.RUnlock()
		p = newEmptyPiece(pieceNum, t)
		t.rwLock.Lock()
		t.PieceList[pieceNum] = p
		t.rwLock.Unlock()
	} else {
		t.rwLock.RUnlock()
	}
	return p
}

func (t *Task) AddPiece(p *Piece) {
	t.PieceList[p.PieceNum] = p
}

type TaskStatistic struct {
	lock          sync.RWMutex
	StartTime     time.Time
	EndTime       time.Time
	PeerCount     int32
	FinishedCount int32
	CostList      []int32
}

type StatisticInfo struct {
	StartTime     time.Time
	EndTime       time.Time
	PeerCount     int32
	FinishedCount int32
	Costs         map[int32]int32
}

func (t *TaskStatistic) SetStartTime(start time.Time) {
	t.lock.Lock()
	t.StartTime = start
	t.lock.Unlock()
}

func (t *TaskStatistic) SetEndTime(end time.Time) {
	t.lock.Lock()
	t.EndTime = end
	t.lock.Unlock()
}

func (t *TaskStatistic) AddPeerTaskStart() {
	t.lock.Lock()
	t.PeerCount++
	t.lock.Unlock()
}

func (t *TaskStatistic) AddPeerTaskDown(cost int32) {
	t.lock.Lock()
	t.CostList = append(t.CostList, cost)
	t.lock.Unlock()
}

func (t *TaskStatistic) GetStatistic() (info *StatisticInfo) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	info = &StatisticInfo{
		StartTime:     t.StartTime,
		EndTime:       t.EndTime,
		PeerCount:     t.PeerCount,
		FinishedCount: t.FinishedCount,
		Costs:         make(map[int32]int32),
	}

	if info.EndTime.IsZero() {
		info.EndTime = time.Now()
	}

	count := len(t.CostList)
	count90 := count * 90 / 100
	count95 := count * 95 / 100

	totalCost := int64(0)

	for i, cost := range t.CostList {
		totalCost += int64(cost)
		switch i {
		case count90:
			info.Costs[90] = int32(totalCost / int64(count90))
		case count95:
			info.Costs[95] = int32(totalCost / int64(count95))
		}
	}
	if count > 0 {
		info.Costs[100] = int32(totalCost / int64(count))
	}

	return
}

type Piece struct {
	PieceNum    int32      `protobuf:"varint,1,opt,name=piece_num,json=pieceNum,proto3" json:"piece_num,omitempty"`
	RangeStart  uint64     `protobuf:"varint,2,opt,name=range_start,json=rangeStart,proto3" json:"range_start,omitempty"`
	RangeSize   int32      `protobuf:"varint,3,opt,name=range_size,json=rangeSize,proto3" json:"range_size,omitempty"`
	PieceMd5    string     `protobuf:"bytes,4,opt,name=piece_md5,json=pieceMd5,proto3" json:"piece_md5,omitempty"`
	PieceOffset uint64     `protobuf:"varint,5,opt,name=piece_offset,json=pieceOffset,proto3" json:"piece_offset,omitempty"`
	PieceStyle  PieceStyle `protobuf:"varint,6,opt,name=piece_style,json=pieceStyle,proto3,enum=base.PieceStyle" json:"piece_style,omitempty"`
}

type PieceStyle int32
