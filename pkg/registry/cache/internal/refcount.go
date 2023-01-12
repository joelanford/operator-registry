package internal

import (
	"sync"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type RefCounter[Item any] struct {
	item  *Item
	m     sync.Mutex
	count uint64

	Open  func() (*Item, error)
	Close func(*Item) error
}

func (rc *RefCounter[Item]) With(f func(*Item) error) (err error) {
	item, err := rc.open()
	if err != nil {
		return err
	}
	defer func() {
		closeErr := rc.close()
		err = utilerrors.NewAggregate([]error{err, closeErr})
	}()
	return f(item)
}

func (rc *RefCounter[Item]) open() (*Item, error) {
	rc.m.Lock()
	defer rc.m.Unlock()
	if rc.item == nil {
		db, err := rc.Open()
		if err != nil {
			return nil, err
		}
		rc.item = db
	}
	rc.count += 1
	return rc.item, nil
}

func (rc *RefCounter[Item]) close() error {
	rc.m.Lock()
	defer rc.m.Unlock()
	if rc.count == 1 {
		if err := rc.Close(rc.item); err != nil {
			return err
		}
		rc.item = nil
	}
	rc.count -= 1
	return nil
}
