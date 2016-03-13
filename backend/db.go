package backend

import (
	"container/list"
	"fmt"
	. "github.com/bytedance/dbatman/database/sql/driver/mysql"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
)

type DB struct {
	sync.Mutex

	addr         string
	user         string
	password     string
	db           string
	maxIdleConns int

	idleConns []*list.List

	connNum int32
	barrel  int
}

func Open(addr string, user string, password string, dbName string) (*DB, error) {
	db := new(DB)

	db.addr = addr
	db.user = user
	db.password = password
	db.db = dbName

	db.barrel = runtime.NumCPU()
	db.idleConns = make([]*list.List, db.barrel, db.barrel)
	for i := range db.idleConns {
		db.idleConns[i] = list.New()
	}

	db.connNum = 0
	return db, nil
}

func (db *DB) Addr() string {
	return db.addr
}

func (db *DB) String() string {
	return fmt.Sprintf("%s:%s@%s/%s?maxIdleConns=%v",
		db.user, db.password, db.addr, db.db, db.maxIdleConns)
}

func (db *DB) Close() error {
	db.Lock()
	defer db.Unlock()

	for i := range db.idleConns {
		if db.idleConns[i].Len() > 0 {
			v := db.idleConns[i].Back()
			co := v.Value.(*BackendConn)
			db.idleConns[i].Remove(v)
			co.Close()
		} else {
			break
		}
	}

	db.connNum = 0
	return nil
}

func (db *DB) Ping() error {
	c, err := db.PopConn()
	if err != nil {
		return err
	}

	err = c.Ping()
	db.PushConn(c, err)
	return err
}

func (db *DB) SetMaxIdleConnNum(num int) {
	db.maxIdleConns = num
}

func (db *DB) GetConnNum() int {
	return int(db.connNum)
}

func (db *DB) newConn() (*BackendConn, error) {
	co := new(BackendConn)

	if err := co.Connect(db.addr, db.user, db.password, db.db); err != nil {
		return nil, err
	}

	return co, nil
}

func (db *DB) tryReuse(co *BackendConn) error {
	if co.IsInTransaction() {
		//we can not reuse a connection in transaction status
		if err := co.Rollback(); err != nil {
			return err
		}
	}

	if !co.IsAutoCommit() {
		//we can not  reuse a connection not in autocomit
		if _, err := co.exec("set autocommit = 1"); err != nil {
			return err
		}
	}

	//connection may be set names early
	//we must use default utf8
	if co.GetCharset() != DEFAULT_CHARSET {
		if err := co.SetCharset(DEFAULT_CHARSET); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) PopConn() (co *BackendConn, err error) {
	idx := rand.Intn(db.barrel)

	db.Lock()
	if db.idleConns[idx].Len() > 0 {
		v := db.idleConns[idx].Front()
		co = v.Value.(*BackendConn)
		db.idleConns[idx].Remove(v)
	}
	db.Unlock()

	if co != nil {
		if err := co.Ping(); err == nil {
			if err := db.tryReuse(co); err == nil {
				//connection may alive
				return co, nil
			}
		}
		co.Close()
	}

	co, err = db.newConn()
	if err == nil {
		atomic.AddInt32(&db.connNum, 1)
	}
	return
}

func (db *DB) PushConn(co *BackendConn, err error) {
	var closeConn *BackendConn = nil

	if err != nil {
		closeConn = co
	} else {
		if db.maxIdleConns > 0 {
			idx := rand.Intn(db.barrel)
			db.Lock()
			if db.idleConns[idx].Len() >= db.maxIdleConns {
				v := db.idleConns[idx].Front()
				closeConn = v.Value.(*BackendConn)
				db.idleConns[idx].Remove(v)
			}
			db.idleConns[idx].PushBack(co)
			db.Unlock()

		} else {
			closeConn = co
		}

	}

	if closeConn != nil {
		atomic.AddInt32(&db.connNum, -1)

		closeConn.Close()
	}
}

type SqlConn struct {
	*BackendConn

	db *DB
}

func (p *SqlConn) Close() {
	if p.BackendConn != nil {
		p.db.PushConn(p.BackendConn, p.BackendConn.pkgErr)
		p.BackendConn = nil
	}
}

func (db *DB) GetConn() (*SqlConn, error) {
	c, err := db.PopConn()
	return &SqlConn{c, db}, err
}
