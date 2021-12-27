# SQL Connection Pool

This little package is supposed to be of help when needing multiple database
connections of type ``*sql.DB``.
Available drivers are ``mysql``, ``postgresql`` and ``sqlite``.

## Installation & Import

install with : ``go get -u github.com/KaiserWerk/SQL-Connection-Pool``

import as e.g. ``sqlpool "github.com/KaiserWerk/SQL-Connection-Pool"``

## Usage

Create a new pool using default values, just supply the required driver and a DSN: 
```golang
pool := sqlpool.New("mysql", "root:password@tcp(127.0.0.1:3306)/dbname", nil)
defer pool.Close()
```

You can supply a ``*PoolConfig`` as third parameter which allows you to alter the
maximum connection limit and the connection check interval, e.g.

```golang
pool := sqlpool.New("mysql", "root:password@tcp(127.0.0.1:3306)/dbname", &sqlpool.PoolConfig{
    MaxConn: 10,
    MonitorInterval: 2 * time.Minute,
})
defer pool.Close()
```

``pool.Close()`` closes all existing connections.

Get a connection (this either returns an unused existing one or creates a new connection if
the maximum is not reached yet):
```golang
conn, err := pool.Get()
```

``DB`` contains the actual ``*sql.DB`` connection. Use it to execute queries:
```golang
result, err := conn.DB.Query("SELECT * FROM user") // or Exec() or whatever
```

Return a connection when it is no longer needed. This makes the connection available 
again to be obtained via ``Get()``:
```golang
err := pool.Return(conn)
```

## Niche methods

There are a few method which might prove to be helpful.

Return the current total connection count:
```golang
num := pool.GetConnectionCount()
```

Return the maximum connection count:
```golang
max := pool.GetMaxConnectionCount()
```
