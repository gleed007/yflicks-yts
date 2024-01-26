# Changelog

<a name="v0.8.3"></a>
## [v0.8.3](https://github.com/atifcppprogrammer/yflicks-yts/compare/v0.8.2...v0.8.3) (2024-01-26)

### Fix

* updated `SearchMovieFilters` with `Quality`, `SortBy` and `OrderBy` types


<a name="v0.8.2"></a>
## [v0.8.2](https://github.com/atifcppprogrammer/yflicks-yts/compare/v0.8.0...v0.8.2) (2024-01-26)

### Feat

* created `MoviePartial` method for creating torrent magnet

### Fix

* updated partials to use `Genre` and `Quality` types


<a name="v0.8.0"></a>
## [v0.8.0](https://github.com/atifcppprogrammer/yflicks-yts/compare/v0.7.2...v0.8.0) (2024-01-25)

### Feat

* **client:** created method for scraping home page content
* **client:** created method for scraping trending movies

### Pull Requests

* Merge pull request [#6](https://github.com/atifcppprogrammer/yflicks-yts/issues/6) from atifcppprogrammer/feature/site-scrape


<a name="v0.7.2"></a>
## [v0.7.2](https://github.com/atifcppprogrammer/yflicks-yts/compare/v0.7.0...v0.7.2) (2024-01-25)

### Feat

* exposed methods returning genre, sortBy and orderBy lists


<a name="v0.7.0"></a>
## [v0.7.0](https://github.com/atifcppprogrammer/yflicks-yts/compare/v0.6.0...v0.7.0) (2024-01-22)

### Feat

* **client:** requiring request timeout for client
* **client:** updated methods to require `context.Context` argument

### Refactor

* moved `internal/validate` package into `yts` package

### Pull Requests

* Merge pull request [#4](https://github.com/atifcppprogrammer/yflicks-yts/issues/4) from atifcppprogrammer/feature/ctx-support


<a name="v0.6.0"></a>
## v0.6.0 (2024-01-16)

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

