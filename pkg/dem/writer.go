package dem

import (
	"encoding/gob"
	"errors"
	"github.com/wladich/elevation_server/pkg/lz4"
	"io"
	"os"
	"sync"
)

type StorageWriter struct {
	storageAbstract
	lock sync.Mutex
	fIdx *os.File
}

func NewWriter(path string, overwrite bool) (*StorageWriter, error) {
	var crflag = os.O_EXCL
	if overwrite {
		crflag = os.O_TRUNC
	}
	var storage StorageWriter
	idxPath := path + ".idx"
	f, err := os.OpenFile(path, os.O_CREATE|crflag|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	storage.fData = f

	f, err = os.OpenFile(idxPath, os.O_CREATE|crflag|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	storage.fIdx = f

	storage.index = &tileFileIndex{}
	return &storage, nil
}

func (storage *StorageWriter) Close() error {
	encoder := gob.NewEncoder(storage.fIdx)
	err1 := encoder.Encode(*storage.index)
	err2 := storage.fIdx.Close()
	err3 := storage.fData.Close()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}

func compressTile(tileData TileRawData) []byte {
	return lz4.CompressHigh(tileData[:], 12)
}

func (storage *StorageWriter) PutTile(tile TileRaw) error {
	compressed := compressTile(tile.Data)
	storage.lock.Lock()
	defer storage.lock.Unlock()
	pos, err := storage.fData.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = storage.fData.Write(compressed)
	if err != nil {
		return err
	}
	x := tile.Index.X + 180*HgtSplitParts
	y := tile.Index.Y + 90*HgtSplitParts
	if x < 0 || y < 0 || x > len(storage.index) || y > len(storage.index[x]) {
		return errors.New("tile index out of range")
	}
	storage.index[x][y] = tileFileIndexRecord{pos, int64(len(compressed))}
	return nil
}
