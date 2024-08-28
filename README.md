<!-- #region cli -->
# k6build

**Build k6 with various builders.**

Build k6 using one of the supported builders.

## Commands

* [k6build cache](#k6build-cache)	 - k6build cache server
* [k6build local](#k6build-local)	 - build custom k6 binary locally
* [k6build remote](#k6build-remote)	 - build a custom k6 using a remote build server
* [k6build server](#k6build-server)	 - k6 build service

---
# k6build cache

k6build cache server

## Synopsis


Starts a k6build cache server. 

The cache server offers a REST API for storing and downloading objects.

Objects can be retrieved by a download url returned when the object is stored.

The --download-url specifies the base URL for downloading objects. This is necessary to allow
downloading the objects from different machines.


```
k6build cache [flags]
```

## Examples

```

# start the cache server serving an external url
k6build cache --download0url http://external.url

# store object from same host
curl -x POST http://localhost:9000/cache/objectID -d "object content" | jq .
{
	"Error": "",
	"Object": {
	  "ID": "objectID",
	  "Checksum": "17d3eb873fe4b1aac4f9d2505aefbb5b53b9a7f34a6aadd561be104c0e9d678b",
	  "URL": "http://external.url:9000/cache/objectID/download"
	}
      }

# download object from another machine using the external url
curl http://external.url:9000/cache/objectID/download

```

## Flags

```
  -c, --cache-dir string      cache directory (default "/tmp/cache/objectstore")
  -d, --download-url string   base url used for downloading objects. If not specified http://localhost:<port> is used
  -h, --help                  help for cache
  -l, --log-level string      log level (default "INFO")
  -p, --port int              port server will listen (default 9000)
```

## SEE ALSO

* [k6build](#k6build)	 - Build k6 with various builders.

---
# k6build local

build custom k6 binary locally

## Synopsis


k6build local builder creates a custom k6 binary artifacts that satisfies certain
dependencies. Requires the golang toolchain and git.


```
k6build local [flags]
```

## Examples

```

# build k6 v0.51.0 with latest version of k6/x/kubernetes
k6build local -k v0.51.0 -d k6/x/kubernetes

platform: linux/amd64
k6: v0.51.0
k6/x/kubernetes: v0.9.0
checksum: 7f06720503c80153816b4ef9f58571c2fce620e0447fba1bb092188ff87e322d

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build local -k v0.51.0 \
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

platform: linux/amd64
k6: v0.51.0
k6/x/kubernetes: v0.8.0
k6/x/output-kafka": v0.7.0
checksum: f4af178bb2e29862c0fc7d481076c9ba4468572903480fe9d6c999fea75f3793


# build k6 v0.50.0 with latest version of k6/x/kubernetes using a custom catalog
k6build local -k v0.50.0 -d k6/x/kubernetes \
    -c /path/to/catalog.json -q

# build k6 v0.50.0 using a custom GOPROXY
k6build local -k v0.50.0 -e GOPROXY=http://localhost:80 -q

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
  -o, --output string            path to put the binary as an executable. (default "k6")
  -p, --platform string          target platform (default GOOS/GOARCH)
  -q, --quiet                    don't print artifact's details
  -v, --verbose                  print build process output
```

## SEE ALSO

* [k6build](#k6build)	 - Build k6 with various builders.

---
# k6build remote

build a custom k6 using a remote build server

## Synopsis


Builds custom k6 binaries using a k6build server returning the details of the
binary artifact and optionally download it.


```
k6build remote [flags]
```

## Examples

```

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6build remote -s http://localhost:8000 \
    -k v0.51.0 \
    -p linux/amd64 \
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

id: 62d08b13fdef171435e2c6874eaad0bb35f2f9c7
platform: linux/amd64
k6: v0.51.0
k6/x/kubernetes: v0.9.0
k6/x/output-kafka": v0.7.0
checksum: f4af178bb2e29862c0fc7d481076c9ba4468572903480fe9d6c999fea75f3793
url: http://localhost:8000/cache/62d08b13fdef171435e2c6874eaad0bb35f2f9c7/download


# build k6 v0.51 with k6/x/output-kafka v0.7.0 and download as 'build/k6'
k6build remote -s http://localhost:8000 \
    -p linux/amd64  \
    -k v0.51.0 -d k6/x/output-kafka:v0.7.0 \
    -o build/k6 -q

# check downloaded binary
build/k6 version
k6 v0.51.0 (go1.22.2, linux/amd64)
Extensions:
  github.com/grafana/xk6-output-kafka v0.7.0, xk6-kafka [output]

```

## Flags

```
  -d, --dependency stringArray   list of dependencies in form package:constrains
  -h, --help                     help for remote
  -k, --k6 string                k6 version constrains (default "*")
  -o, --output string            path to download the custom binary as an executable. If not specified, the artifact is not downloaded.
  -p, --platform string          target platform (default GOOS/GOARCH)
  -q, --quiet                    don't print artifact's details
  -s, --server string            url for build server (default "http://localhost:8000")
```

## SEE ALSO

* [k6build](#k6build)	 - Build k6 with various builders.

---
# k6build server

k6 build service

## Synopsis


starts a k6build server


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
      --cache-url string     cache server url (default "http://localhost:9000")
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
