package sqlpool

import (
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type (
	SqlPool struct {
		maxConnections uint16
		mut            sync.Mutex
		connections    []*SqlConn
		driver         string
		dsn            string
	}
	SqlConn struct {
		id    uint32
		inUse bool
		DB    *sql.DB
	}
	PoolConfig struct{}
)

var (
	idCounter uint32
)

func getNextId() uint32 {
	return atomic.AddUint32(&idCounter, 1)
}

/*
In the background, ping all connections every 1 minute or so
If a connection doesnt respond to a ping, close it and remove it
and create a new connection instead
*/

func New(driver string, dsn string) *SqlPool {
	p := SqlPool{
		maxConnections: 3,
		connections:    make([]*SqlConn, 0),
		driver:         driver,
		dsn:            dsn,
	}
	return &p
}

func (pool *SqlPool) Get() (*SqlConn, error) {
	pool.mut.Lock()
	defer pool.mut.Unlock()

	// check first if any connections are available
	for _, v := range pool.connections {
		if !v.inUse {
			v.inUse = true // pool.connections[i] ?
			return v, nil
		}
	}

	// check if new connections can be made
	if len(pool.connections) < int(pool.maxConnections) {
		newConn, err := pool.createConnection()
		if err != nil {
			return nil, fmt.Errorf("could not create new connection: %s", err.Error())
		}

		pool.connections = append(pool.connections, newConn)
		return newConn, nil
	}

	return nil, fmt.Errorf("no free connection and limit reached")
}

func (pool *SqlPool) CloseAll() {
	for _, v := range pool.connections {
		v.DB.Close()
	}
}

func (pool *SqlPool) GetMaxConnections() uint16 {
	return pool.maxConnections
}

func (pool *SqlPool) Return(conn *SqlConn) {
	// TODO: implement
}

func (pool *SqlPool) Close(conn *SqlConn) error {
	return conn.DB.Close()
}

func (pool *SqlPool) createConnection() (*SqlConn, error) {
	db, err := sql.Open(pool.driver, pool.dsn)
	if err != nil {
		return nil, fmt.Errorf("could not prepare DB connection: %s", err.Error())
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping DB connection: %s", err.Error())
	}

	c := SqlConn{
		id:    getNextId(),
		inUse: false,
		DB:    db,
	}

	return &c, nil
}
