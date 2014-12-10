# Interstellar: file sync utility for multiple remote servers with terabytes of files

Usage: is srcHost:src dstHost:dst

You have to define host configuration in ~/.isrc like this:

```
{
  "host1": {
    "User": "alice",
    "Host": "example.com",
    "BaseDir": "/data/foo"
  },
 "host2": {
    "User": "bob",
    "Host": "example.org",
    "BaseDir": "/home/foobar/baz",
    "Through": ["example.net", "example.com"]
  }
}
```

