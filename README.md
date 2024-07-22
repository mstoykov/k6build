<!-- #region cli -->
# k6build

**Build k6 with various builders.**

Build k6 using one of the supported builders.

## Commands

* [k6build client](#k6build-client)	 - build k6 using a remote build server
* [k6build local](#k6build-local)	 - build using a local builder
* [k6build server](#k6build-server)	 - k6 build service

---
# k6build client

build k6 using a remote build server

## Synopsis


k6build client connects to a remote build server


```
k6build client [flags]
```

## Examples

```

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build client -s http://localhost:8000 \
    -k v0.51.0 \
    -p linux/amd64 
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

{
    "id": "62d08b13fdef171435e2c6874eaad0bb35f2f9c7",
    "url": "http://localhost:8000/cache/62d08b13fdef171435e2c6874eaad0bb35f2f9c7/download",
    "dependencies": {
	"k6": "v0.51.0",
	"k6/x/kubernetes": "v0.9.0",
	"k6/x/output-kafka": "v0.7.0"
    },
    "platform": "linux/amd64",
    "checksum": "f4af178bb2e29862c0fc7d481076c9ba4468572903480fe9d6c999fea75f3793"
}

# build k6 v0.51 with k6/x/output-kafka v0.7.0 and download to 'build/k6'
k6build client -s http://localhost:8000
    -p linux/amd64 
    -k v0.51.0 -d k6/x/output-kafka:v0.7.0
    -o build/k6 -q

# check binary
build/k6 version
k6 v0.51.0 (go1.22.2, linux/amd64)
Extensions:
  github.com/grafana/xk6-output-kafka v0.7.0, xk6-kafka [output]



# build latest version of k6 with a version of k6/x/kubernetes greater than v0.8.0
k6build client -s http://localhost:8000 \
    -p linux/amd64 \
    -k v0.50.0 -d 'k6/x/kubernetes:>v0.8.0'
{
   "id": "18035a12975b262430b55988ffe053098d020034",
   "url": "http://localhost:8000/cache/18035a12975b262430b55988ffe053098d020034/download",
   "dependencies": {
       "k6": "v0.50.0",
	"k6/x/kubernetes": "v0.9.0"
    },
   "platform": "linux/amd64",
   "checksum": "255e5d62852af5e4109a0ac6f5818936a91c986919d12d8437e97fb96919847b"
}

```

## Flags

```
  -d, --dependency stringArray   list of dependencies in form package:constrains
  -h, --help                     help for client
  -k, --k6 string                k6 version constrains (default "*")
  -o, --output string            path to download the artifact as an executable. If not specified, the artifact is not downloaded.
  -p, --platform string          target platform (default GOOS/GOARCH)
  -q, --quiet                    don't print artifact's details
  -s, --server string            url for build server (default "http://localhost:8000")
```

## SEE ALSO

* [k6build](#k6build)	 - Build k6 with various builders.

---
# k6build local

build using a local builder

## Synopsis


k6build local builder returns artifacts that satisfies certain dependencies


```
k6build local [flags]
```

## Examples

```

# build k6 v0.50.0 with latest version of k6/x/kubernetes
k6build local -k v0.50.0 -d k6/x/kubernetes

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build local -k v0.51.0 \
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

# build latest version of k6 with a version of k6/x/kubernetes greater than v0.8.0
k6build local -k v0.50.0 -d 'k6/x/kubernetes:>v0.8.0'

# build k6 v0.50.0 with latest version of k6/x/kubernetes using a custom catalog
k6build local -k v0.50.0 -d k6/x/kubernetes \
    -c /path/to/catalog.json

# build k6 v0.50.0 using a custom GOPROXY
k6build local -k v0.50.0 -e GOPROXY=http://localhost:80

```

## Flags

```
  -f, --cache-dir string         cache dir (default "/tmp/buildservice")
  -c, --catalog string           dependencies catalog (default "catalog.json")
  -g, --copy-go-env              copy go environment (default true)
  -d, --dependency stringArray   list of dependencies in form package:constrains
  -e, --env stringToString       build environment variables (default [])
  -h, --help                     help for local
  -k, --k6 string                k6 version constrains (default "*")
  -p, --platform string          target platform (default GOOS/GOARCH)
  -v, --verbose                  print build process output
```

## SEE ALSO

* [k6build](#k6build)	 - Build k6 with various builders.

---
# k6build server

k6 build service

## Synopsis


starts a k6build server that server


```
k6build server [flags]
```

## Examples

```

# start the build server using a custom catalog
k6build server -c /path/to/catalog.json

# start the server the build server using a custom GOPROXY
k6build server -e GOPROXY=http://localhost:80
```

## Flags

```
  -f, --cache-dir string     cache dir (default "/tmp/buildservice")
  -c, --catalog string       dependencies catalog (default "catalog.json")
  -g, --copy-go-env          copy go environment (default true)
  -e, --env stringToString   build environment variables (default [])
  -h, --help                 help for server
  -l, --log-level string     log level (default "INFO")
  -p, --port int             port server will listen (default 8000)
  -v, --verbose              print build process output
```

## SEE ALSO

* [k6build](#k6build)	 - Build k6 with various builders.

<!-- #endregion cli -->
