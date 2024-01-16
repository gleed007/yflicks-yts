# Changelog

<a name="0.6.0"></a>
## 0.6.0 (2024-01-16)

### Feat

* created struct type convering movie details filters to query string
* created struct type converting movie search filters to query string
* created struct types for movies search endpoint
* **client:** created method for movie suggestions endpoint
* **client:** created method for movie details endpoint
* **client:** validating search movies filters before returning query string
* **client:** created YTS client with method for searching movies

### Fix

* corrected typo for `release` target

### Refactor

* moved `StructValidationError` to `validate.go`
* colocated validation logic and filter errors in internal package
* **client:** encapsulated network utilities in internal package

### Pull Requests

* Merge pull request [#2](https://github.com/atifcppprogrammer/yflicks-yts/issues/2) from atifcppprogrammer/feature/api-endpoints

