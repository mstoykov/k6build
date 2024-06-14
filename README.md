# k6 build service


## local build service

```
k6 build service returns artifacts that satisfies certain dependencies

Usage:
  k6build [flags]

Flags:
  -c, --catalog string           dependencies catalog (default "catalog.json")
  -d, --dependency stringArray   list of dependencies in form package:constrains
  -h, --help                     help for k6build
  -k, --k6-constrains string     k6 version constrains (default "*")
  -p, --platform string          target platform
  -v, --verbose                  print build output
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