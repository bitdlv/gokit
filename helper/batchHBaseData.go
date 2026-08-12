package helper

import "sync"

/*   hbaseDataDef: 并发安全   */
type hbaseDataDef struct {
	rowKeyMapData map[string]map[string]map[string][]byte
	mutex         sync.Mutex
}

func NewHBaseDataInfo(size int) *hbaseDataDef {
	return &hbaseDataDef{
		rowKeyMapData: make(map[string]map[string]map[string][]byte, size),
		mutex:         sync.Mutex{},
	}
}

func (hd *hbaseDataDef) Set(key string, value map[string]map[string][]byte) {
	hd.mutex.Lock()
	hd.rowKeyMapData[key] = value
	hd.mutex.Unlock()
}

func (hd *hbaseDataDef) Length() int {
	hd.mutex.Lock()
	len := len(hd.rowKeyMapData)
	hd.mutex.Unlock()

	return len
}

func (hd *hbaseDataDef) Rotate(size int) (oldData map[string]map[string]map[string][]byte) {
	hd.mutex.Lock()
	oldData = hd.rowKeyMapData
	hd.rowKeyMapData = make(map[string]map[string]map[string][]byte, size)
	hd.mutex.Unlock()

	return
}
