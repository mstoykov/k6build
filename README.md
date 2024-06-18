# k6 build service


## local build service

```
k6 build service returns artifacts that satisfies certain dependencies

Usage:
  k6build [flags]

Examples:

# build k6 v0.50.0 with latest version of k6/x/kubernetes
k6build -k v0.50.0 -d k6/x/kubernetes

# build k6 v0.51.0 with k6/x/kubernetes v0.8.0 and k6/x/output-kafka v0.7.0
k6foundry build -k v0.51.0 \
    -d k6/x/kubernetes:v0.8.0 \
    -d k6/x/output-kafka:v0.7.0

# build latest version of k6 with a version of k6/x/kubernetes greater than v0.8.0
k6build -k v0.50.0 -d 'k6/x/kubernetes:>v0.8.0'

# build k6 v0.50.0 with latest version of k6/x/kubernetes using a custom catalog
k6build -k v0.50.0 -d k6/x/kubernetes \
    -c /path/to/catalog.json

# build k6 v0.50.0 using a custom GOPROXY
k6build -k v0.50.0 -e GOPROXY=http://localhost:80


Flags:
  -f, --cache-dir                cache dir (default "/tmp/buildservice")
  -c, --catalog                  dependencies catalog (default "catalog.json")
  -g, --copy-go-env              copy go environment (default true)
  -d, --dependency               list of dependencies in form package:constrains
  -e, --env                      build environment variables
  -h, --help                     help for k6build
  -k, --k6                       k6 version constrains (default "*")
  -p, --platform                 target platform (default GOOS/GOARCH)
  -v, --verbose                  print build process output      
```

### Example

```
k6build  -c catalog.json  -p linux/amd64 -k v0.50.0 \ 
   -d k6/x/kubernetes -d k6/x/output-kafka:v0.7.0311b995
{
  "id": "da39a3ee5e6b4b0d3255bfef95601890afd80709",
  "url": "file:///tmp/buildcache2702618547/2364966898",
  "dependencies": {
    "k6": "v0.50.0",
    "k6/x/kubernetes": "v0.8.0",
    "k6/x/output-kafka": "v0.7.0"
  },
  "platform": "linux/amd64",
  "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```