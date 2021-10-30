package sqlpool

import (
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type (
	SqlPool struct {
		maxConnections  uint16
		mut             sync.Mutex
		connections     map[uint32]*SqlConn
		driver          string
		dsn             string
		monitorInterval time.Duration
	}
	SqlConn struct {
		id      uint32
		inUse   bool
		lastUse time.Time
		DB      *sql.DB
	}
	PoolConfig struct {
		MaxConn         uint16
		MonitorInterval time.Duration
	}
)

const (
	defaultMaxConnections  = 3
	defaultMonitorInterval = time.Minute
	defaultIdleTime = 5 * time.Minute
)

var (
	idCounter uint32
)

func New(driver string, dsn string, config *PoolConfig) *SqlPool {
	p := SqlPool{
		maxConnections:  defaultMaxConnections,
		connections:     make(map[uint32]*SqlConn),
		driver:          driver,
		dsn:             dsn,
		monitorInterval: defaultMonitorInterval,
	}
	if config != nil {
		if config.MaxConn > 0 {
			p.maxConnections = config.MaxConn
		}
		if config.MonitorInterval != 0 {
			p.monitorInterval = config.MonitorInterval
		}
	}

	go func(pool *SqlPool) {
		for {
			pool.monitor()
			time.Sleep(pool.monitorInterval)
		}
	}(&p)

	return &p
}

func (pool *SqlPool) monitor() {
	pool.mut.Lock()
	defer pool.mut.Unlock()

	wg := new(sync.WaitGroup)
	for id, conn := range pool.connections {
		go func(wg *sync.WaitGroup, id uint32, connection *SqlConn) {
			if connection.lastUse.Sub(time.Now()) > defaultIdleTime {
				_ = connection.DB.Close()
				delete(pool.connections, id)
				return
			}

			if err := connection.DB.Ping(); err != nil {
				_ = connection.DB.Close()
				delete(pool.connections, id)
				return
			}

		}(wg, id, conn)
	}

	wg.Wait()
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

		newConn.inUse = true
		newConn.lastUse = time.Now()
		pool.connections[newConn.id] = newConn

		return newConn, nil
	}

	return nil, fmt.Errorf("no free connection and limit reached")
}

func (pool *SqlPool) CloseAll() {
	pool.mut.Lock()
	defer pool.mut.Unlock()
	for _, v := range pool.connections {
		_ = v.DB.Close()
	}
}

func (pool *SqlPool) GetConnectionCount() int {
	return len(pool.connections)
}

func (pool *SqlPool) GetMaxConnectionCount() uint16 {
	return pool.maxConnections
}

func (pool *SqlPool) Return(conn *SqlConn) error {
	for _, v := range pool.connections {
		if conn.id == v.id {
			v.inUse = false
			return nil
		}
	}

	return fmt.Errorf("this connection is not part of the connection pool")
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

func getNextId() uint32 {
	return atomic.AddUint32(&idCounter, 1)
}