# SQL Connection Pool

This little package is supposed to be of help when needing multiple database
connections of type ``*sql.DB``.

## Usage

```golang
pool := sqlpool.New("mysql", "root:password@tcp(127.0.0.1:3306)/dbname")
```